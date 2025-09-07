// Package api 定义API接口和协议缓冲区，用于gRPC和HTTP服务的接口定义.
package api

import (
	"github.com/gin-gonic/gin"

	"github.com/yeisme/notevault/pkg/internal/router"
)

// RegisterAPIs 注册所有API路由.
func RegisterAPIs(e *gin.Engine) {
	root := e.Group("/")
	router.RegisterHealthCheckRoute(root)

	v1 := e.Group("/api/v1")
	router.RegisterRoutes(v1)

	router.RegisterSwaggerRoute(e)
}
