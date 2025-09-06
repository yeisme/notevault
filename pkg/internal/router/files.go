package router

import (
	"github.com/gin-gonic/gin"

	"github.com/yeisme/notevault/pkg/internal/handle"
)

// RegisterFilesRoutes 注册文件操作相关路由.
func RegisterFilesRoutes(g *gin.RouterGroup) {
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
}
