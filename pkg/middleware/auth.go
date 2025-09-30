package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/yeisme/notevault/pkg/configs"
)

// AuthMiddleware 基于 oauth2-proxy 注入的请求头做统一身份认证校验。
//   - 优先要求存在 X-Auth-Request-Email 或 X-Forwarded-Email
//   - 支持通过配置跳过某些路径（如 /metrics, /health）
//   - 开发模式可允许 query user 兜底（由 configs.auth.dev_allow_query 控制）.
func AuthMiddleware(conf configs.AuthConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !conf.Enabled || isSkippedPath(c.Request.URL.Path, conf.SkipPaths) {
			c.Next()
			return
		}

		email := strings.TrimSpace(c.GetHeader("X-Auth-Request-Email"))
		if email == "" {
			email = strings.TrimSpace(c.GetHeader("X-Forwarded-Email"))
		}

		if email == "" {
			if conf.DevAllowQuery && c.Query("user") != "" {
				c.Next()
				return
			}

			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})

			return
		}

		c.Next()
	}
}

func isSkippedPath(path string, skips []string) bool {
	if path == "" || len(skips) == 0 {
		return false
	}

	for _, p := range skips {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		if strings.HasPrefix(path, p) {
			return true
		}
	}

	return false
}
