// Package middleware 提供角色与权限相关的中间件和辅助方法。
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Role 表示请求方的角色（使用 iota 实现的枚举，数值越大权限越高）。
type Role int

const (
	RoleUser Role = iota + 1
	RoleMember
	RoleEnterprise
	RoleAdmin
)

// String 返回角色的字符串表示。
func (r Role) String() string {
	switch r {
	case RoleAdmin:
		return "admin"
	case RoleEnterprise:
		return "enterprise"
	case RoleMember:
		return "member"
	case RoleUser:
		fallthrough
	default:
		return "user"
	}
}

type roleKey struct{}

// parseRole 从字符串解析角色，未知值降级为 user。
func parseRole(s string) Role {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "admin":
		return RoleAdmin
	case "enterprise":
		return RoleEnterprise
	case "member":
		return RoleMember
	case "user":
		fallthrough
	default:
		return RoleUser
	}
}

// RoleMiddleware 解析 X-Role 并注入到 gin.Context 和 request.Context。
// 缺省角色为 user。
func RoleMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		r := parseRole(c.GetHeader("X-Role"))
		// 保存到 gin context
		c.Set("role", r)
		// 也保存到 request context，便于下游 service 获取
		ctx := context.WithValue(c.Request.Context(), roleKey{}, r)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// GetRole 从 gin.Context 获取当前请求角色。
func GetRole(c *gin.Context) Role {
	if v, ok := c.Get("role"); ok {
		if r, ok2 := v.(Role); ok2 {
			return r
		}
	}
	// 回退到 request context
	if v := c.Request.Context().Value(roleKey{}); v != nil {
		if r, ok := v.(Role); ok {
			return r
		}
	}

	return RoleUser
}

// RequireMinRole 要求最小角色，不满足则返回 403。
func RequireMinRole(minRole Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		r := GetRole(c)
		if r < minRole { // 使用枚举的自然顺序进行最小角色判断
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden: insufficient role"})
			return
		}

		c.Next()
	}
}
