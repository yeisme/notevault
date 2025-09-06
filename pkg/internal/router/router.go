// Package router 管理路由配置，用于设置HTTP服务的路由规则.
package router

import (
	"github.com/gin-gonic/gin"

	"github.com/yeisme/notevault/pkg/internal/handle"
)

// RegisterRoutes 注册所有业务相关路由.
// 为了方便使用 gin 的 Bind ，尽量使用 POST 请求，方便拓展 json yaml 等多种格式.
func RegisterRoutes(g *gin.RouterGroup) {
	// 文件操作路由
	filesRoutes := g.Group("/files")
	{
		// 单个文件操作
		singleGroup := filesRoutes.Group("/:id")
		{
			// 删除文件
			singleGroup.DELETE("", handle.DefaultHandler)
			// 更新文件（生成预签名 URL）
			singleGroup.POST("", handle.DefaultHandler)
			// 下载文件
			singleGroup.POST("/download", handle.DefaultHandler)
			// 获取文件访问 URL (生成预签名 URL)
			singleGroup.POST("/url", handle.DefaultHandler)
			// 复制文件
			singleGroup.POST("/copy", handle.DefaultHandler)
			// 移动文件
			singleGroup.POST("/move", handle.DefaultHandler)

			// 文件版本管理
			versionGroup := singleGroup.Group("/versions")
			{
				versionGroup.GET("", handle.DefaultHandler)         // 获取版本列表
				versionGroup.POST("", handle.DefaultHandler)        // 创建新版本
				versionGroup.DELETE("/:vid", handle.DefaultHandler) // 删除版本
			}
		}

		// 上传文件（生成预签名 URL），支持批量上传
		filesRoutes.POST("", handle.UploadFile)
		// 列表/搜索
		filesRoutes.POST("/search", handle.DefaultHandler)
		// 创建文件夹
		filesRoutes.POST("/folder", handle.DefaultHandler)

		// 批量文件操作
		batchGroup := filesRoutes.Group("/batch")
		{
			// 批量删除文件
			batchGroup.DELETE("", handle.DefaultHandler)
			// 批量获取文件元数据
			batchGroup.GET("", handle.DefaultHandler)
			// 批量更新文件元数据
			batchGroup.PUT("", handle.DefaultHandler)
			// 批量下载文件
			batchGroup.GET("/download", handle.DefaultHandler)
			// 批量获取文件访问 URL (生成预签名 URL)
			batchGroup.POST("/urls", handle.DefaultHandler)
			// 批量复制文件
			batchGroup.POST("/copy", handle.DefaultHandler)
			// 批量移动文件
			batchGroup.POST("/move", handle.DefaultHandler)
		}

		// 文件元数据管理路由
		metaGroup := filesRoutes.Group("/meta/:id")
		{
			// 文件元数据的增删改查
			metaGroup.GET("", handle.DefaultHandler)
			metaGroup.POST("", handle.DefaultHandler)
			metaGroup.PUT("", handle.DefaultHandler)
			metaGroup.DELETE("", handle.DefaultHandler)
			// 获取文件元数据的预签名 URL
			metaGroup.POST("/url", handle.DefaultHandler)
		}
	}

	// 文件分享路由
	sharesRoutes := g.Group("/shares")
	{
		sharesRoutes.POST("", handle.DefaultHandler)                 // 创建分享链接
		sharesRoutes.GET("/:shareId", handle.DefaultHandler)         // 获取分享详情
		sharesRoutes.DELETE("/:shareId", handle.DefaultHandler)      // 删除分享
		sharesRoutes.POST("/:shareId/access", handle.DefaultHandler) // 访问分享文件
	}

	// 回收站路由
	trashRoutes := g.Group("/trash")
	{
		trashRoutes.GET("", handle.DefaultHandler)              // 获取回收站文件列表
		trashRoutes.POST("/:id/restore", handle.DefaultHandler) // 恢复文件
		trashRoutes.DELETE("/:id", handle.DefaultHandler)       // 永久删除
		trashRoutes.DELETE("", handle.DefaultHandler)           // 清空回收站
	}

	// 统计路由
	statsRoutes := g.Group("/stats")
	{
		statsRoutes.GET("/files", handle.DefaultHandler)   // 文件统计
		statsRoutes.GET("/storage", handle.DefaultHandler) // 存储使用统计
		statsRoutes.GET("/uploads", handle.DefaultHandler) // 上传历史统计
	}
}

// RegisterHealthCheckRoute 注册健康检查路由.
func RegisterHealthCheckRoute(g *gin.RouterGroup) {
	healthRoutes := g.Group("/health")
	{
		healthRoutes.GET("/db", handle.HealthDB)
		healthRoutes.GET("/s3", handle.HealthS3)
		healthRoutes.GET("/mq", handle.HealthMQ)
	}
}
