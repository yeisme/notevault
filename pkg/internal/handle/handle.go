// Package handle 提供请求处理器的实现，用于处理HTTP和gRPC请求.
package handle

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// DefaultHandlers 是在未注入 handlers 时使用的占位实现。
type DefaultHandlers struct{}

func (d *DefaultHandlers) Upload() gin.HandlerFunc {
	return notImplemented
}

func (d *DefaultHandlers) Download() gin.HandlerFunc {
	return notImplemented
}

func (d *DefaultHandlers) Delete() gin.HandlerFunc {
	return notImplemented
}

func (d *DefaultHandlers) List() gin.HandlerFunc {
	return notImplemented
}

// notImplemented 用于当上层未注入处理器时返回 501.
func notImplemented(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
}
