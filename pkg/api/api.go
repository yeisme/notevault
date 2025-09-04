// Package api 定义API接口和协议缓冲区，用于gRPC和HTTP服务的接口定义.
package api

import (
	"github.com/gin-gonic/gin"

	"github.com/yeisme/notevault/pkg/internal/handle"
	"github.com/yeisme/notevault/pkg/internal/router"
)

// RegisterGroup 注册文件处理相关的路由组到传入的 gin 引擎.
func RegisterGroup(e *gin.Engine) *gin.Engine {
	router.Register(e.Group("/files"), &handle.FileHandlers{})

	return e
}
