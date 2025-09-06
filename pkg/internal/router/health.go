package router

import (
	"github.com/gin-gonic/gin"

	"github.com/yeisme/notevault/pkg/internal/handle"
)

// RegisterHealthCheckRoute 注册健康检查路由.
func RegisterHealthCheckRoute(g *gin.RouterGroup) {
	healthRoutes := g.Group("/health")
	{
		healthRoutes.GET("/db", handle.HealthDB)
		healthRoutes.GET("/s3", handle.HealthS3)
		healthRoutes.GET("/mq", handle.HealthMQ)
	}
}
