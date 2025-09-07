package router

import (
	"github.com/gin-gonic/gin"

	"github.com/yeisme/notevault/pkg/internal/handle"
)

// RegisterTrashRoutes 注册回收站相关路由.
func RegisterTrashRoutes(g *gin.RouterGroup) {
	// 回收站路由 - 应用回收站专用中间件
	trashRoutes := g.Group("/trash")

	{
		// ===== 回收站文件管理路由 =====
		trashRoutes.GET("", handle.DefaultHandler)         // 获取回收站文件列表
		trashRoutes.POST("/search", handle.DefaultHandler) // 搜索回收站文件

		// ===== 单个文件操作路由 =====
		fileGroup := trashRoutes.Group("/:id")
		{
			fileGroup.POST("/restore", handle.DefaultHandler) // 恢复文件
			fileGroup.DELETE("", handle.DefaultHandler)       // 永久删除文件
			fileGroup.GET("", handle.DefaultHandler)          // 获取文件详情
		}

		// ===== 批量操作路由 =====
		batchGroup := trashRoutes.Group("/batch")
		{
			batchGroup.POST("/restore", handle.DefaultHandler) // 批量恢复
			batchGroup.DELETE("", handle.DefaultHandler)       // 批量永久删除
		}

		// ===== 回收站管理路由 =====
		trashRoutes.DELETE("", handle.DefaultHandler)          // 清空回收站
		trashRoutes.POST("/auto-clean", handle.DefaultHandler) // 自动清理过期文件
	}
}
