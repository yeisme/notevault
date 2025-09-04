// Package app 提供应用程序的初始化和配置功能.
package app

import (
	contextPkg "context"
	"fmt"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/yeisme/notevault/pkg/api"
	"github.com/yeisme/notevault/pkg/configs"
	"github.com/yeisme/notevault/pkg/context"
	"github.com/yeisme/notevault/pkg/internal/storage"
	"github.com/yeisme/notevault/pkg/log"
	"github.com/yeisme/notevault/pkg/metrics"
	"github.com/yeisme/notevault/pkg/middleware"
	"github.com/yeisme/notevault/pkg/tracing"
)

type App struct {
	Engine *gin.Engine
	config *configs.AppConfig
}

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
	)

	mgrCtx := context.WithStorageManager(contextPkg.Background(), manager)

	engine.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(mgrCtx)
		c.Next()
	})

	engine = api.RegisterGroup(engine)

	return &App{
		Engine: engine,
		config: config,
	}
}

func (a *App) Run() error {
	mainAddr := fmt.Sprintf("%s:%d", a.config.Server.Host, a.config.Server.Port) // 通常为: 0.0.0.0:8080

	// 如果没有启用指标，直接运行主服务器
	if !a.config.Metrics.Enabled {
		return a.Engine.Run(mainAddr)
	}

	// 如果启用了指标，检查指标地址配置
	metricsEndpoint := a.config.Metrics.Endpoint // 通常为 :8081

	var metricsAddr string
	if strings.HasPrefix(metricsEndpoint, ":") {
		metricsAddr = fmt.Sprintf("%s%s", a.config.Server.Host, metricsEndpoint)
	} else {
		metricsAddr = metricsEndpoint
	}

	// 经过处理后 metricsAddr 可能是类似 0.0.0.0:8081
	// 如果指标地址等于主地址，则在同一引擎上注册路由并运行单个服务器
	if metricsAddr == mainAddr {
		// 确保注册了指标路由
		_ = metrics.StartMetricsServer(a.config.Metrics, a.Engine)
		return a.Engine.Run(mainAddr)
	}

	// 否则并发启动两个独立的HTTP服务器。
	// 构建一个最小的指标引擎并在其中注册指标端点。
	metricsEngine := gin.Default()
	// register only the metrics-related handlers
	_ = metrics.StartMetricsServer(a.config.Metrics, metricsEngine)

	const serverCount = 2

	errCh := make(chan error, serverCount)

	go func() { errCh <- metricsEngine.Run(metricsAddr) }()
	go func() { errCh <- a.Engine.Run(mainAddr) }()

	// 如果任一服务器返回错误，则通过通道发送该错误
	for range serverCount {
		if err := <-errCh; err != nil {
			return err
		}
	}

	return nil
}
