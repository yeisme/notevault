package router

import (
	"github.com/gin-gonic/gin"

	"github.com/yeisme/notevault/pkg/internal/handle"
)

// RegisterStatsRoutes 注册统计相关路由.
func RegisterStatsRoutes(g *gin.RouterGroup) {
	// 统计路由 - 应用统计专用中间件
	statsRoutes := g.Group("/stats")

	{
		// ===== 文件统计路由 =====
		fileStatsGroup := statsRoutes.Group("/files")
		{
			fileStatsGroup.GET("", handle.DefaultHandler)       // 文件总数统计
			fileStatsGroup.GET("/type", handle.DefaultHandler)  // 按类型统计
			fileStatsGroup.GET("/size", handle.DefaultHandler)  // 文件大小统计
			fileStatsGroup.GET("/trend", handle.DefaultHandler) // 文件数量趋势
		}

		// ===== 存储统计路由 =====
		storageStatsGroup := statsRoutes.Group("/storage")
		{
			storageStatsGroup.GET("", handle.DefaultHandler)        // 存储使用情况
			storageStatsGroup.GET("/bucket", handle.DefaultHandler) // 按存储桶统计
			storageStatsGroup.GET("/trend", handle.DefaultHandler)  // 存储使用趋势
		}

		// ===== 上传统计路由 =====
		uploadStatsGroup := statsRoutes.Group("/uploads")
		{
			uploadStatsGroup.GET("", handle.DefaultHandler)       // 上传历史统计
			uploadStatsGroup.GET("/daily", handle.DefaultHandler) // 每日上传统计
			uploadStatsGroup.GET("/user", handle.DefaultHandler)  // 按用户统计
		}

		// ===== 系统统计路由 =====
		systemStatsGroup := statsRoutes.Group("/system")
		{
			systemStatsGroup.GET("/performance", handle.DefaultHandler) // 系统性能统计
			systemStatsGroup.GET("/errors", handle.DefaultHandler)      // 错误统计
			systemStatsGroup.GET("/usage", handle.DefaultHandler)       // 系统使用统计
		}

		// ===== 综合统计路由 =====
		statsRoutes.GET("/dashboard", handle.DefaultHandler) // 统计仪表板数据
		statsRoutes.GET("/report", handle.DefaultHandler)    // 生成统计报告
	}
}
