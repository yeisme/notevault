// Package app 提供应用程序的初始化和配置功能.
package app

import (
	contextPkg "context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/yeisme/notevault/pkg/configs"
	"github.com/yeisme/notevault/pkg/internal/storage"
	"github.com/yeisme/notevault/pkg/log"
	"github.com/yeisme/notevault/pkg/metrics"
	"github.com/yeisme/notevault/pkg/middleware"
	"github.com/yeisme/notevault/pkg/tracing"
)

type App struct {
	mainServer    *http.Server
	metricsServer *http.Server
	config        *configs.AppConfig
	log           *zerolog.Logger
}

// NewApp 创建并返回一个新的 App 实例.
func NewApp(configPath string) *App {
	ctx := contextPkg.Background()
	engine := gin.New()

	// 初始化配置
	if err := configs.InitConfig(configPath); err != nil {
		fmt.Printf("Error initializing config: %v\n", err)
		os.Exit(1)
	}

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
		os.Exit(1)
	}

	l := log.Logger()
	gin.DefaultWriter = log.NewGinWriter(l, zerolog.InfoLevel)
	gin.DefaultErrorWriter = log.NewGinWriter(l, zerolog.ErrorLevel)

	engine.Use(
		gin.Recovery(),
		middleware.CORSMiddleware(),
		middleware.TracingMiddleware(),
		middleware.PrometheusMiddleware(),
		middleware.StorageMiddleware(manager),
	)

	var ms *http.Server

	if config.Metrics.Enabled {
		var err error

		ms, err = initMetricsServer(engine, config)
		if err != nil {
			fmt.Printf("Error initializing metrics: %v\n", err)
			os.Exit(1)
		}
	}

	main := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", config.Server.Host, config.Server.Port),
		Handler: engine,
	}

	return &App{
		mainServer:    main,
		metricsServer: ms,
		config:        config,
		log:           l,
	}
}

// initMetricsServer sets up the metrics handler and returns a separate http.Server
// when the metrics endpoint uses a different port than the main server.
func initMetricsServer(engine *gin.Engine, config *configs.AppConfig) (*http.Server, error) {
	metricsHandler, err := metrics.StartMetricsServer(config.Metrics)
	if err != nil {
		return nil, err
	}

	// metrics.Endpoint 通常是 ":8081" 或 "0.0.0.0:8081" 等格式，
	// 当端口与主服务器端口一致时，把 /metrics 注册到主 engine；否则新建单独的 server 在后台启动。
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

	// 单独启动 metrics server
	return &http.Server{
		Addr:    endpoint,
		Handler: metricsHandler,
	}, nil
}

// Run 启动主服务器和（可选的）监控服务器.
func (a *App) Run() error {
	// 在后台启动，不阻塞主程序构造
	go func() {
		if a.metricsServer == nil {
			return
		}

		if err := a.metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("metrics server error: %v\n", err)
		}

		a.log.Info().Msgf("Metrics server started on %s", a.config.Metrics.Endpoint)
	}()

	a.log.Info().Msgf("Starting server on %s:%d", a.config.Server.Host, a.config.Server.Port)

	if err := a.mainServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		a.log.Error().Err(err).Msg("Server error")
		return err
	}

	return nil
}
