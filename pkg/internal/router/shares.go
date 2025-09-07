package router

import (
	"github.com/gin-gonic/gin"

	"github.com/yeisme/notevault/pkg/internal/handle"
)

// RegisterSharesRoutes 注册文件分享相关路由.
func RegisterSharesRoutes(g *gin.RouterGroup) {
	// 文件分享路由 - 应用分享专用中间件
	sharesRoutes := g.Group("/shares")

	{
		// ===== 分享管理路由 =====
		sharesRoutes.POST("", handle.DefaultHandler)            // 创建分享链接
		sharesRoutes.GET("", handle.DefaultHandler)             // 获取我的分享列表
		sharesRoutes.DELETE("/:shareId", handle.DefaultHandler) // 删除分享

		// ===== 分享访问路由 =====
		shareAccessGroup := sharesRoutes.Group("/:shareId")
		{
			shareAccessGroup.GET("", handle.DefaultHandler)          // 获取分享详情
			shareAccessGroup.POST("/access", handle.DefaultHandler)  // 访问分享内容
			shareAccessGroup.GET("/download", handle.DefaultHandler) // 下载分享文件
		}

		// ===== 分享权限管理路由 =====
		permissionGroup := sharesRoutes.Group("/:shareId/permissions")
		{
			permissionGroup.GET("", handle.DefaultHandler)                  // 获取分享权限
			permissionGroup.PUT("", handle.DefaultHandler)                  // 更新分享权限
			permissionGroup.POST("/users", handle.DefaultHandler)           // 添加分享用户
			permissionGroup.DELETE("/users/:userId", handle.DefaultHandler) // 移除分享用户
		}
	}
}
