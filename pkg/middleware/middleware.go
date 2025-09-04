// Package middleware 提供中间件
package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/yeisme/notevault/pkg/metrics"
)

// PrometheusMiddleware 创建Gin的Prometheus中间件.
func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// 执行下一个中间件/处理器
		c.Next()

		// 记录指标
		statusCode := c.Writer.Status()
		duration := time.Since(start).Seconds()

		// 使用已有的指标记录器
		metrics.RequestCounter.WithLabelValues(method, path).Inc()
		metrics.RequestDuration.WithLabelValues(method, path).Observe(duration)

		// 可以添加更多指标，比如按状态码统计
		_ = statusCode // 暂时未使用，可扩展
	}
}

// CORSMiddleware CORS中间件.
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
