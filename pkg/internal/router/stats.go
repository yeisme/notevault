package router

import (
	"github.com/gin-gonic/gin"

	"github.com/yeisme/notevault/pkg/internal/handle"
)

// RegisterStatsRoutes 注册统计相关路由.
func RegisterStatsRoutes(g *gin.RouterGroup) {
	// 统计路由
	statsRoutes := g.Group("/stats")
	{
		statsRoutes.GET("/files", handle.DefaultHandler)   // 文件统计
		statsRoutes.GET("/storage", handle.DefaultHandler) // 存储使用统计
		statsRoutes.GET("/uploads", handle.DefaultHandler) // 上传历史统计
	}
}
