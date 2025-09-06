package router

import (
	"github.com/gin-gonic/gin"

	"github.com/yeisme/notevault/pkg/internal/handle"
)

// RegisterTrashRoutes 注册回收站相关路由.
func RegisterTrashRoutes(g *gin.RouterGroup) {
	// 回收站路由
	trashRoutes := g.Group("/trash")
	{
		trashRoutes.GET("", handle.DefaultHandler)              // 获取回收站文件列表
		trashRoutes.POST("/:id/restore", handle.DefaultHandler) // 恢复文件
		trashRoutes.DELETE("/:id", handle.DefaultHandler)       // 永久删除
		trashRoutes.DELETE("", handle.DefaultHandler)           // 清空回收站
	}
}
