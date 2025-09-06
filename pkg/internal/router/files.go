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
		// 上传文件 (生成预签名 URL,带有策略)
		filesRoutes.POST("/upload/urls", handle.UploadFileURL)
		filesRoutes.POST("/upload/urls/policy", handle.UploadFileURLPolicy)
		// 列表/搜索
		filesRoutes.POST("/search", handle.DefaultHandler)
		// 创建文件夹
		filesRoutes.POST("/folder", handle.DefaultHandler)

		// 文件操作 (支持单个和批量，通过请求体中的ID列表处理)
		filesRoutes.DELETE("", handle.DefaultHandler)            // 删除文件 (单个/批量)
		filesRoutes.POST("", handle.DefaultHandler)              // 更新文件 (单个/批量)
		filesRoutes.POST("/download", handle.DefaultHandler)     // 下载文件 (单个/批量)
		filesRoutes.POST("/download/url", handle.GetDownloadURL) // 获取文件访问URL (单个/批量)
		filesRoutes.POST("/copy", handle.DefaultHandler)         // 复制文件 (单个/批量)
		filesRoutes.POST("/move", handle.DefaultHandler)         // 移动文件 (单个/批量)

		// 文件版本管理 (单个文件)
		versionGroup := filesRoutes.Group("/versions")
		{
			versionGroup.GET("", handle.DefaultHandler)         // 获取版本列表
			versionGroup.POST("", handle.DefaultHandler)        // 创建新版本
			versionGroup.DELETE("/:vid", handle.DefaultHandler) // 删除版本
		}
	}

	// 文件元数据管理路由
	metaRoutes := g.Group("/meta")
	{
		metaGroup := metaRoutes.Group("/:id")
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
