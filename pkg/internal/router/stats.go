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
			fileStatsGroup.GET("", handle.GetFilesStats)            // 文件总数统计
			fileStatsGroup.GET("/type", handle.GetFilesStatsByType) // 按类型统计
			fileStatsGroup.GET("/size", handle.GetFilesStatsBySize) // 文件大小统计
			fileStatsGroup.GET("/trend", handle.GetFilesTrend)      // 文件数量趋势
		}

		// ===== 存储统计路由 =====
		storageStatsGroup := statsRoutes.Group("/storage")

		{
			storageStatsGroup.GET("", handle.StorageStats)           // 存储使用情况
			storageStatsGroup.GET("/bucket", handle.StorageByBucket) // 按存储桶统计
			storageStatsGroup.GET("/trend", handle.GetFilesTrend)    // 存储使用趋势（复用文件趋势）
		}

		// ===== 上传统计路由 =====
		uploadStatsGroup := statsRoutes.Group("/uploads")

		{
			uploadStatsGroup.GET("", handle.UploadsStats)            // 上传历史统计
			uploadStatsGroup.GET("/daily", handle.UploadsDailyStats) // 每日上传统计
			uploadStatsGroup.GET("/user", handle.UploadsByUser)      // 按用户统计（当前用户）
		}

		// ===== 综合统计路由 =====
		statsRoutes.GET("/dashboard", handle.DashboardStats) // 统计仪表板数据
		statsRoutes.GET("/report", handle.ReportStats)       // 生成统计报告
	}
}
