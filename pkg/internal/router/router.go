// Package router 管理路由配置，用于设置HTTP和gRPC服务的路由规则.
package router

import (
	"github.com/gin-gonic/gin"

	"github.com/yeisme/notevault/pkg/internal/handle"
)

// ObjHandlers 定义由应用层注入的具体请求处理器. router 包只负责将路径和处理器绑定到 gin 引擎，
// 处理器的实现应由  pkg/internal/handle 提供并注入进来.
type ObjHandlers interface {
	Upload() gin.HandlerFunc
	Download() gin.HandlerFunc
	Delete() gin.HandlerFunc
	List() gin.HandlerFunc
}

// Register 将路由绑定到传入的 gin 路由组，并返回实际使用的 handlers 实例。
// 如果传入的 handlers 为 nil，会使用返回 501 的占位实现以便服务能启动且能清晰地提示未实现。
// 绑定的路径（假定上层会用 files := r.Group("/files")）：
//
//	POST   /        -> Upload
//	GET    /        -> List
//	GET    /:id     -> Download
//	DELETE /:id     -> Delete
func Register(group *gin.RouterGroup, handlers ObjHandlers) ObjHandlers {
	// 提供默认占位实现
	if handlers == nil {
		handlers = &handle.DefaultHandlers{}
	}

	// 绑定路由到提供的 handlers

	group.POST("/", handlers.Upload())
	group.GET("/", handlers.List())
	group.GET("/:id", handlers.Download())
	group.DELETE("/:id", handlers.Delete())

	return handlers
}
