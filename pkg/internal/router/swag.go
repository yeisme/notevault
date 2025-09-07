package router

import (
	"fmt"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/yeisme/notevault/docs"
	"github.com/yeisme/notevault/pkg/configs"
)

// RegisterSwaggerRoute 注册Swagger文档路由.
func RegisterSwaggerRoute(r *gin.Engine) {
	cfg := configs.GetConfig()
	if !cfg.Server.Debug {
		return
	}

	docs.SwaggerInfo.Host = fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	docs.SwaggerInfo.Version = "1.0.0"

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
