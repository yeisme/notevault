// Package handle 提供请求处理器的实现，用于处理HTTP和gRPC请求.
package handle

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func DefaultHandler(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Not Implemented"})
}
