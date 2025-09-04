package middleware

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/yeisme/notevault/pkg/log"
)

// GinLoggerMiddleware 使用zerolog记录Gin请求日志的中间件.
func GinLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		method := c.Request.Method
		clientIP := c.ClientIP()

		// 执行下一个中间件/处理器
		c.Next()

		// 计算延迟
		latency := time.Since(start)

		// 获取状态码
		statusCode := c.Writer.Status()

		// 如果有查询参数，添加到路径中
		if raw != "" {
			path = path + "?" + raw
		}

		// 获取错误信息（如果有）
		var errorMsg string
		if len(c.Errors) > 0 {
			errorMsg = c.Errors.String()
		}

		// 使用zerolog记录日志
		logger := log.Logger()
		event := logger.Info().
			Int("status", statusCode).
			Dur("latency", latency).
			Str("method", method).
			Str("path", path).
			Str("client_ip", clientIP)

		if errorMsg != "" {
			event = event.Str("error", errorMsg)
		}

		event.Msg("HTTP request")
	}
}
