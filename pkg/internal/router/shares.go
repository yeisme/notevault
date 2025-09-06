package router

import (
	"github.com/gin-gonic/gin"

	"github.com/yeisme/notevault/pkg/internal/handle"
)

// RegisterSharesRoutes 注册文件分享相关路由.
func RegisterSharesRoutes(g *gin.RouterGroup) {
	// 文件分享路由
	sharesRoutes := g.Group("/shares")
	{
		sharesRoutes.POST("", handle.DefaultHandler)                 // 创建分享链接
		sharesRoutes.GET("/:shareId", handle.DefaultHandler)         // 获取分享详情
		sharesRoutes.DELETE("/:shareId", handle.DefaultHandler)      // 删除分享
		sharesRoutes.POST("/:shareId/access", handle.DefaultHandler) // 访问分享文件
	}
}
