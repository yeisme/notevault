// Package router 管理路由配置，用于设置HTTP服务的路由规则.
package router

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes 注册所有业务相关路由.
// 为了方便使用 gin 的 Bind ，尽量使用 POST 请求，方便拓展 json yaml 等多种格式.
func RegisterRoutes(g *gin.RouterGroup) {
	RegisterFilesRoutes(g)
	RegisterSharesRoutes(g)
	RegisterTrashRoutes(g)
	RegisterStatsRoutes(g)
}
