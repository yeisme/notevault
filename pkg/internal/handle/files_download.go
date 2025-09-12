package handle

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/yeisme/notevault/pkg/internal/service"
	"github.com/yeisme/notevault/pkg/internal/types"
	"github.com/yeisme/notevault/pkg/log"
)

// GetDownloadURL 处理获取文件访问 URL（单个/批量）.
//
//	@Summary		获取文件访问URL
//	@Description	为文件获取预签名的访问URL，支持单个或批量对象
//	@Tags			文件下载
//	@Accept			json
//	@Produce		json
//	@Param			objects	body		types.GetFilesURLRequest	true	"获取文件访问URL请求"
//	@Success		200		{object}	types.GetFilesURLResponse	"预签名访问URL响应"
//	@Failure		400		{object}	map[string]string			"请求参数错误"
//	@Failure		500		{object}	map[string]string			"服务器内部错误"
//	@Router			/api/v1/files/download/url [post]
func GetDownloadURL(c *gin.Context) {
	l := log.Logger()

	var req types.GetFilesURLRequest
	if err := c.ShouldBind(&req); err != nil {
		l.Warn().Err(err).Msg("invalid request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		return
	}

	// 至少需要一个对象
	if len(req.Objects) == 0 {
		l.Warn().Msg("no objects provided in request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "no objects provided"})

		return
	}

	svc := service.NewFileService(c.Request.Context())

	resp, err := svc.PresignedGetURLs(c.Request.Context(), &req)
	if err != nil {
		l.Error().Err(err).Msg("failed to generate presigned get urls")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusOK, resp)
}
