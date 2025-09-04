package handle

import (
	"net/http"

	"github.com/gin-gonic/gin"

	pkgctx "github.com/yeisme/notevault/pkg/context"
	"github.com/yeisme/notevault/pkg/log"
)

var fileLog = log.Logger().With().Str("component", "file_handlers").Logger()

// FileHandlers 是一个可注入的处理器集合，上层可以用自己的实现替换这些方法。
type FileHandlers struct{}

// Upload 处理文件上传请求。
func (h *FileHandlers) Upload() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从 gin.Context 获取 request-scoped context
		reqCtx := c.Request.Context()

		// 从上下文中提取 storage.Manager
		mgr := pkgctx.GetManager(reqCtx)
		fileLog.Debug().Msgf("Extracted storage manager from context: %+v", mgr)

		if mgr == nil {
			// 如果没有注入 Manager，返回 500 并给出明确提示
			c.JSON(http.StatusInternalServerError, gin.H{"error": "storage manager not available in context"})
			return
		}

		// 这里是演示如何继续使用 mgr（例如获取 S3 客户端或 DB 客户端）
		// s3Client := mgr.GetS3Client()
		// ... 使用 s3Client 执行上传逻辑

		// 占位响应，具体实现请在上层替换此处理器
		c.JSON(http.StatusNotImplemented, gin.H{"error": "upload not implemented (manager available)"})
	}
}

// Download 处理文件下载请求。
func (h *FileHandlers) Download() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
	}
}

// Delete 处理删除文件请求。
func (h *FileHandlers) Delete() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
	}
}

// List 返回文件列表。
func (h *FileHandlers) List() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
	}
}
