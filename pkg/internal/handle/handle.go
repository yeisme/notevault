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
	// 提取用户名：Header 优先 -> query 参数 -> 默认 test-user（便于测试）
	user := c.GetHeader("X-User")
	if user == "" {
		user = c.Query("user")
	}
	// 测试默认值，不为 Debug 或者 Test 模式时
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
