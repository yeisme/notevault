package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sony/gobreaker"

	"github.com/yeisme/notevault/pkg/configs"
)

// CircuitBreakerMiddleware 基于 gobreaker 的简单熔断.
func CircuitBreakerMiddleware(cfg configs.CircuitBreakerConfig) gin.HandlerFunc {
	if !cfg.Enabled {
		return func(c *gin.Context) { c.Next() }
	}

	settings := gobreaker.Settings{
		Name:        "http-middlewares",
		MaxRequests: cfg.MaxRequestsInHalf,
		Interval:    time.Duration(cfg.IntervalSeconds) * time.Second,
		Timeout:     time.Duration(cfg.TimeoutSeconds) * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			total := counts.Requests
			if total < cfg.MinRequests {
				return false
			}
			// 失败比例
			failureRate := float64(counts.TotalFailures) / float64(total)
			return failureRate >= cfg.FailureRate
		},
	}
	cb := gobreaker.NewCircuitBreaker(settings)

	return func(c *gin.Context) {
		_, err := cb.Execute(func() (any, error) {
			c.Next()
			// 将 5xx 视为失败
			const firstServerErr = http.StatusInternalServerError

			status := c.Writer.Status()
			if status >= firstServerErr {
				return nil, gobreaker.ErrOpenState // 返回非 nil 触发失败计数
			}

			return nil, nil
		})
		if err == gobreaker.ErrOpenState {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "service temporarily unavailable"})
			return
		}
	}
}
