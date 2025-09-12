package router

import (
	"github.com/gin-gonic/gin"

	"github.com/yeisme/notevault/pkg/internal/handle"
)

// RegisterFilesRoutes 注册文件操作相关路由.
func RegisterFilesRoutes(g *gin.RouterGroup) {
	// 文件操作路由 - 应用文件操作专用中间件
	filesRoutes := g.Group("/files")

	{
		// ===== 文件上传相关路由 =====
		uploadGroup := filesRoutes.Group("/upload")
		{
			// 生成预签名URL进行上传
			uploadGroup.POST("/urls", handle.UploadFileURL)              // PUT方式，无策略
			uploadGroup.POST("/urls/policy", handle.UploadFileURLPolicy) // POST方式，带策略

			// 直接上传小文件（适用于小文件场景）
			uploadGroup.POST("/single", handle.UploadSingleFile) // 单个小文件
			uploadGroup.POST("/batch", handle.UploadBatchFiles)  // 批量小文件
		}

		// ===== 文件查询相关路由 =====
		filesRoutes.GET("", handle.DefaultHandler)         // 获取文件列表
		filesRoutes.POST("/search", handle.DefaultHandler) // 高级搜索

		// ===== 文件夹管理路由 =====
		folderGroup := filesRoutes.Group("/folder")
		{
			folderGroup.POST("", handle.CreateFolder)       // 创建文件夹
			folderGroup.PUT("/:id", handle.RenameFolder)    // 重命名文件夹
			folderGroup.DELETE("/:id", handle.DeleteFolder) // 删除文件夹
		}

		// ===== 文件操作路由（支持单个和批量） =====
		// 注意：通过请求体中的ID列表来支持批量操作
		filesRoutes.DELETE("", handle.DefaultHandler)    // 删除文件(单个/批量)
		filesRoutes.PUT("", handle.DefaultHandler)       // 更新文件元数据(单个/批量)
		filesRoutes.POST("/copy", handle.DefaultHandler) // 复制文件(单个/批量)
		filesRoutes.POST("/move", handle.DefaultHandler) // 移动文件(单个/批量)

		// ===== 文件下载路由 =====
		downloadGroup := filesRoutes.Group("/download")
		{
			downloadGroup.POST("", handle.DefaultHandler)     // 下载文件(单个/批量)
			downloadGroup.POST("/url", handle.GetDownloadURL) // 获取下载URL(单个/批量)
		}

		// ===== 文件版本管理路由 =====
		versionGroup := filesRoutes.Group("/versions")
		{
			versionGroup.GET("/:fileId", handle.DefaultHandler)                     // 获取版本列表
			versionGroup.POST("/:fileId", handle.DefaultHandler)                    // 创建新版本
			versionGroup.DELETE("/:fileId/:versionId", handle.DefaultHandler)       // 删除指定版本
			versionGroup.POST("/:fileId/:versionId/restore", handle.DefaultHandler) // 恢复到指定版本
		}
	}

	// ===== 文件元数据管理路由 =====
	metaRoutes := g.Group("/meta")

	{
		metaGroup := metaRoutes.Group("/:id")
		{
			metaGroup.GET("", handle.DefaultHandler)      // 获取元数据
			metaGroup.POST("", handle.DefaultHandler)     // 创建/更新元数据
			metaGroup.PUT("", handle.DefaultHandler)      // 更新元数据
			metaGroup.DELETE("", handle.DefaultHandler)   // 删除元数据
			metaGroup.POST("/url", handle.DefaultHandler) // 获取元数据预签名URL
		}

		// 批量元数据操作
		metaRoutes.POST("/batch", handle.DefaultHandler) // 批量获取元数据
	}
}
