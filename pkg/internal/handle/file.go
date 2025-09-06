package handle

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/yeisme/notevault/pkg/internal/service"
	"github.com/yeisme/notevault/pkg/internal/types"
	"github.com/yeisme/notevault/pkg/log"
)

const (
	// MaxFileSize 最大文件大小限制.
	MaxFileSize = 100 * 1024 * 1024 // 100MB
)

// UploadFile 处理上传文件请求：生成预签名 URL 或直接上传.
func UploadFile(c *gin.Context) {
	fileLog := log.Logger()

	var req types.UploadFileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fileLog.Warn().Err(err).Msg("invalid request")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		return
	}

	// 验证文件大小
	if req.FileSize > MaxFileSize {
		fileLog.Warn().Int64("file_size", req.FileSize).Msg("file size exceeds limit")
		c.JSON(http.StatusBadRequest, gin.H{"error": "file size exceeds 10MB limit"})

		return
	}

	user, err := checkUser(c)
	if user == "" || err != nil {
		fileLog.Warn().Err(err).Msg("missing or invalid user")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user"})

		return
	}

	fileLog.Info().
		Str("file_name", req.FileName).
		Str("file_type", req.FileType).
		Int64("file_size", req.FileSize).
		Str("user", user).
		Msg("processing upload request")

	svc := service.NewFileService(c.Request.Context())

	res, err := svc.PresignedPutURL(c.Request.Context(), user, &req)
	if err != nil {
		fileLog.Error().Err(err).Msg("failed to generate presigned URL")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	fileLog.Info().Str("object_key", res.ObjectKey).Msg("successfully generated presigned URL")
	c.JSON(http.StatusOK, res)
}
