// Package handle 提供请求处理器的实现，用于处理HTTP和gRPC请求.
package handle

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/yeisme/notevault/pkg/rule"
)

func DefaultHandler(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"message": "Not Implemented"})
}

func checkUser(c *gin.Context) (string, error) {
	// 优先从 oauth2-proxy 注入的头中读取（按常见头部顺序）
	// 参考：https://oauth2-proxy.github.io/oauth2-proxy/
	candidates := []string{
		"X-Auth-Request-Email", // nginx auth_request 常用
		"X-Forwarded-Email",    // oauth2-proxy -set-xauthrequest 会设置
		"X-Auth-Request-User",  // 某些 provider 仅有 user
		"X-Forwarded-User",
		"X-User", // 兼容旧自定义头，方便测试
	}

	var user string

	for _, h := range candidates {
		if v := strings.TrimSpace(c.GetHeader(h)); v != "" {
			user = v
			break
		}
	}

	// 兼容 query 参数（仅开发/测试场景使用）
	if user == "" {
		user = strings.TrimSpace(c.Query("user"))
	}

	// 开发/测试默认用户（仅非 Release 模式）
	if user == "" && gin.Mode() != gin.ReleaseMode {
		user = "test-user@example.com"
	}

	user = strings.TrimSpace(user)

	// 使用 validator 验证用户名格式为 email
	if err := rule.ValidateVar(user, "required,email"); err != nil {
		return "", err
	}

	return user, nil
}
