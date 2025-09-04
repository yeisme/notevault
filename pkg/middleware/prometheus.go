package middleware

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/yeisme/notevault/pkg/metrics"
)

// PrometheusMiddleware Prometheus监控中间件.
func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// 执行下一个中间件/处理器
		c.Next()

		// 记录请求计数
		metrics.RequestCounter.WithLabelValues(method, path).Inc()

		// 记录请求持续时间
		duration := time.Since(start).Seconds()
		metrics.RequestDuration.WithLabelValues(method, path).Observe(duration)
	}
}
