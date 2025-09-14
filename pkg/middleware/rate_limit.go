package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"github.com/yeisme/notevault/pkg/configs"
)

// RateLimitMiddleware 返回一个基于配置的限流中间件.
func RateLimitMiddleware(cfg configs.RateLimitConfig) gin.HandlerFunc {
	if !cfg.Enabled || cfg.RPS <= 0 {
		return func(c *gin.Context) { c.Next() }
	}

	// 选择 key 维度
	keyMode := strings.ToLower(strings.TrimSpace(cfg.Key))
	// 全局 limiter
	if keyMode == "global" || keyMode == "" {
		limiter := rate.NewLimiter(rate.Limit(cfg.RPS), cfg.Burst)

		return func(c *gin.Context) {
			if !limiter.Allow() {
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
				return
			}

			c.Next()
		}
	}

	// 多键 limiter
	var (
		mu       sync.Mutex
		limiters = map[string]*rate.Limiter{}
	)

	// 获取限流器
	getLimiter := func(key string) *rate.Limiter {
		mu.Lock()
		defer mu.Unlock()

		if l, ok := limiters[key]; ok {
			return l
		}

		l := rate.NewLimiter(rate.Limit(cfg.RPS), cfg.Burst)
		limiters[key] = l

		return l
	}

	// 后台清理闲置 limiter（简单实现）
	go func() {
		const (
			cleanupInterval   = 10 * time.Minute
			maxLimiterEntries = 10000
		)

		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()

		for range ticker.C {
			mu.Lock()
			// 简化：不做逐个访问时间统计，仅在 map 较大时重置
			if len(limiters) > maxLimiterEntries { // 粗略的上限
				limiters = map[string]*rate.Limiter{}
			}

			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		var key string

		switch {
		case strings.HasPrefix(keyMode, "header:"): // 按请求头
			h := strings.TrimPrefix(keyMode, "header:")

			key = c.GetHeader(h)
			if key == "" { // fallback 到 IP
				key = clientIP(c)
			}
		case keyMode == "ip": // 按客户端 IP
			key = clientIP(c)
		default:
			key = clientIP(c)
		}

		if key == "" {
			key = "unknown"
		}
		// 获取对应的 limiter 并检查
		if !getLimiter(key).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests,
				gin.H{"error": "rate limit exceeded, request too frequent, please try again later"})

			return
		}

		c.Next()
	}
}

func clientIP(c *gin.Context) string {
	ip := c.ClientIP()
	if ip == "" {
		// 进一步尝试从 RemoteAddr
		host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
		if err == nil {
			ip = host
		} else {
			ip = c.Request.RemoteAddr
		}
	}

	return ip
}
