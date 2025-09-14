// Package app 提供应用程序的初始化和配置功能.
package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"

	"github.com/yeisme/notevault/pkg/api"
	"github.com/yeisme/notevault/pkg/configs"
	"github.com/yeisme/notevault/pkg/internal/storage"
	"github.com/yeisme/notevault/pkg/log"
	"github.com/yeisme/notevault/pkg/metrics"
	"github.com/yeisme/notevault/pkg/middleware"
	"github.com/yeisme/notevault/pkg/tracing"
)

const (
	// DefaultShutdownTimeout 定义服务器关闭的默认超时时间.
	DefaultShutdownTimeout = 30 * time.Second
)

// App 定义应用程序的主要结构.
type App struct {
	mainServer    *http.Server
	metricsServer *http.Server
	config        *configs.AppConfig
	log           *zerolog.Logger
	mg            *storage.Manager
	done          chan struct{} // 用于通知 Run 方法退出（带缓冲，避免阻塞）
}

// NewApp 创建并返回一个新的 App 实例.
func NewApp(configPath string) *App {
	ctx := context.Background()
	engine := gin.New()

	// 初始化追踪
	config := configs.GetConfig()
	if err := tracing.InitTracer(config.Tracing); err != nil {
		fmt.Printf("Error initializing tracing: %v\n", err)
		os.Exit(1)
	}

	// 初始化监控
	if err := metrics.InitMetrics(config.Metrics); err != nil {
		fmt.Printf("Error initializing metrics: %v\n", err)
		os.Exit(1)
	}

	manager, err := storage.Init(ctx)
	if err != nil {
		fmt.Printf("Error initializing storage: %v\n", err)
		// 继续运行，当存储无法使用的时候，可以继续运行，通过健康检查暴露错误
	}

	l := log.Logger()
	gin.DefaultWriter = log.NewGinWriter(l, zerolog.InfoLevel)
	gin.DefaultErrorWriter = log.NewGinWriter(l, zerolog.ErrorLevel)

	engine.Use(
		gin.Logger(),
		gin.Recovery(),
		gzip.Gzip(gzip.DefaultCompression),
		middleware.CORSMiddleware(config.Server),
		middleware.TracingMiddleware(),
		middleware.PrometheusMiddleware(),
		middleware.StorageMiddleware(manager),
	)

	var ms *http.Server

	if config.Metrics.Enabled {
		var err error

		ms, err = initMetricsServer(engine, config)
		if err != nil {
			l.Error().Err(err).Msg("Error initializing metrics server")
			os.Exit(1)
		}
	}

	api.RegisterAPIs(engine)

	main := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port),
		Handler: engine,
	}

	return &App{
		mainServer:    main,
		metricsServer: ms,
		config:        config,
		log:           l,
		mg:            manager,
		done:          make(chan struct{}, 1),
	}
}

// Run 启动主服务器和（可选的）监控服务器.
func (a *App) Run() error {
	g, _ := errgroup.WithContext(context.Background())

	// 启动指标服务器
	if a.metricsServer != nil {
		g.Go(func() error {
			a.log.Info().Msgf("Metrics server started on %s", a.config.Metrics.Endpoint)

			err := a.metricsServer.ListenAndServe()
			if err != nil && err != http.ErrServerClosed {
				return err
			}

			return nil
		})
	}

	// 启动主服务器
	g.Go(func() error {
		a.log.Info().Msgf("Starting server on %s:%d", a.config.Server.Host, a.config.Server.Port)

		err := a.mainServer.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			return err
		}

		return nil
	})

	// 等待信号或 done 通道
	g.Go(func() error {
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

		select {
		case <-a.done:
			return nil
		case <-signalChan:
			a.shutdown()
			return nil
		}
	})

	// 等待所有 goroutine 退出
	return g.Wait()
}

// Shutdown 优雅关闭服务器和资源（公开方法）.
func (a *App) Shutdown() {
	a.shutdown()
}

// shutdown 优雅关闭服务器和资源.
func (a *App) shutdown() {
	// 创建关闭上下文
	shutdownCtx, shutdownCancel :=
		context.WithTimeout(context.Background(), DefaultShutdownTimeout)
	defer shutdownCancel()

	// 优雅关闭主服务器
	if err := a.mainServer.Shutdown(shutdownCtx); err != nil && err != http.ErrServerClosed {
		a.log.Error().Err(err).Msg("Error shutting down main server")
	}

	// 优雅关闭监控服务器
	if a.metricsServer != nil {
		if err := a.metricsServer.Shutdown(shutdownCtx); err != nil && err != http.ErrServerClosed {
			a.log.Error().Err(err).Msg("Error shutting down metrics server")
		}
	}

	if err := a.mg.Close(); err != nil {
		a.log.Error().Err(err).Msg("Error closing storage manager")
	}

	// 发送完成信号
	select {
	case a.done <- struct{}{}:
	default:
	}

	a.log.Info().Msg("Application shutdown complete")
}

// initMetricsServer 设置指标处理程序并返回单独的 HTTP.Server
// 当指标端点使用与主服务器不同的端口时.
func initMetricsServer(engine *gin.Engine, config *configs.AppConfig) (*http.Server, error) {
	metricsHandler, err := metrics.StartMetricsServer(config.Metrics)
	if err != nil {
		return nil, err
	}

	// metrics.Endpoint 通常是 ":8081" 或 "0.0.0.0:8081" 等格式，
	// 当端口与主服务器端口一致时，把 /metrics 注册到主 engine；否则新建单独的 server 在后台启动.
	endpoint := strings.TrimSpace(config.Metrics.Endpoint)
	// 只比较端口部分
	var metricsPort int

	// :8081
	if after, ok := strings.CutPrefix(endpoint, ":"); ok {
		if p, convErr := strconv.Atoi(after); convErr == nil {
			metricsPort = p
		}
	} else if strings.Contains(endpoint, ":") { //  0.0.0.0:8081
		parts := strings.Split(endpoint, ":")
		if p, convErr := strconv.Atoi(parts[len(parts)-1]); convErr == nil {
			metricsPort = p
		}
	}

	// 如果端口相同，直接注册到主 gin 引擎
	if metricsPort != 0 && metricsPort == config.Server.Port {
		// 挂载到主 gin 引擎
		engine.GET("/metrics", gin.WrapH(metricsHandler))

		if config.Metrics.Pprof {
			// pprof endpoints are registered under /debug/pprof/
			engine.GET("/debug/pprof/*any", gin.WrapH(metricsHandler))
			engine.GET("/debug", func(c *gin.Context) {
				c.Redirect(http.StatusFound, "/debug/pprof/")
			})
		}

		return nil, nil
	}

	// 否则，启动单独的 HTTP 服务器
	return &http.Server{
		Addr:    endpoint,
		Handler: metricsHandler,
	}, nil
}
