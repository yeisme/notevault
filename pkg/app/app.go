// Package app 提供应用程序的初始化和配置功能.
package app

import (
	contextPkg "context"
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

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

	context.WithStorageManager(ctx, manager)

	l := log.Logger()
	gin.DefaultWriter = log.NewGinWriter(l, zerolog.InfoLevel)
	gin.DefaultErrorWriter = log.NewGinWriter(l, zerolog.ErrorLevel)

	engine.Use(
		gin.Recovery(),
		middleware.CORSMiddleware(),
		middleware.TracingMiddleware(),
		middleware.PrometheusMiddleware(),
	)

	if config.Metrics.Enabled {
		_ = metrics.StartMetricsServer(config.Metrics, engine)
	}

	return &App{
		Engine: engine,
		config: config,
	}
}

func (a *App) Run() error {
	return a.Engine.Run(fmt.Sprintf("%s:%d", a.config.Server.Host, a.config.Server.Port))
}
