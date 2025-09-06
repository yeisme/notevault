package handle

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/yeisme/notevault/pkg/internal/service"
	"github.com/yeisme/notevault/pkg/internal/types"
	"github.com/yeisme/notevault/pkg/log"
)

// UploadFileURLPolicy 处理上传文件请求：生成预签名 URL 或直接上传.
func UploadFileURLPolicy(c *gin.Context) {
	fileLog := log.Logger()

	var req types.UploadFilesRequestPolicy
	if err := c.ShouldBind(&req); err != nil {
		fileLog.Warn().Err(err).Msg("invalid request")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		return
	}

	if len(req.Files) == 0 {
		fileLog.Warn().Msg("no files provided in request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "no files provided"})

		return
	}

	user, err := checkUser(c)
	if user == "" || err != nil {
		fileLog.Warn().Err(err).Msg("missing or invalid user")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user"})

		return
	}

	fileLog.Debug().
		Int("file_count", len(req.Files)).
		Str("user", user).
		Msg("processing batch upload request")

	svc := service.NewFileService(c.Request.Context())

	res, err := svc.PresignedPostURLsPolicy(c.Request.Context(), user, &req)
	if err != nil {
		fileLog.Error().Err(err).Msg("failed to generate presigned URL")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	fileLog.Info().Int("result_count", len(res.Results)).Msg("successfully generated presigned URLs")
	c.JSON(http.StatusOK, res)
}

// UploadFileURL 处理单个文件上传请求：生成预签名 URL.
func UploadFileURL(c *gin.Context) {

}
