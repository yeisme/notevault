// Package router 管理路由配置，用于设置HTTP服务的路由规则.
package router

import (
	"github.com/gin-gonic/gin"

	"github.com/yeisme/notevault/pkg/internal/handle"
)

// RegisterRoutes 注册所有业务相关路由.
func RegisterRoutes(g *gin.RouterGroup) {
	fileRoutes := g.Group("/files")
	{
		fileRoutes.POST("/", handle.DefaultHandler)
		fileRoutes.GET("/:id", handle.DefaultHandler)
		fileRoutes.PUT("/:id", handle.DefaultHandler)
		fileRoutes.DELETE("/:id", handle.DefaultHandler)
		fileRoutes.GET("/", handle.DefaultHandler)
		fileRoutes.GET("/search", handle.DefaultHandler)
	}
}

// RegisterHealthCheckRoute 注册健康检查路由.
func RegisterHealthCheckRoute(g *gin.RouterGroup) {
	healthRoutes := g.Group("/health")
	{
		healthRoutes.GET("/db", handle.HealthDB)
		healthRoutes.GET("/s3", handle.HealthS3)
		healthRoutes.GET("/mq", handle.HealthMQ)
	}
}
