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
		trashRoutes.GET("", handle.ListTrash) // 获取回收站文件列表
		// 可选：支持搜索接口（当前先复用列表 + 前端过滤，后续落库搜索）

		// ===== 单个文件操作路由 =====
		fileGroup := trashRoutes.Group("/:id")
		{
			fileGroup.POST("/restore", handle.RestoreTrash) // 恢复文件
			fileGroup.DELETE("", handle.DeleteTrash)        // 永久删除文件
			// fileGroup.GET("", handle.GetTrashItem)       // 获取文件详情（可选）
		}

		// ===== 批量操作路由 =====
		batchGroup := trashRoutes.Group("/batch")
		{
			batchGroup.POST("/restore", handle.RestoreTrashBatch) // 批量恢复
			batchGroup.DELETE("", handle.DeleteTrashBatch)        // 批量永久删除
		}

		// ===== 回收站管理路由 =====
		trashRoutes.DELETE("", handle.EmptyTrash)              // 清空回收站
		trashRoutes.POST("/auto-clean", handle.AutoCleanTrash) // 自动清理过期文件
	}
}
