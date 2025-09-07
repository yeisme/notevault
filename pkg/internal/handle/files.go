package handle

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/yeisme/notevault/pkg/internal/service"
	"github.com/yeisme/notevault/pkg/internal/types"
	"github.com/yeisme/notevault/pkg/log"
)

// UploadFileURLPolicy 处理上传文件请求：生成预签名 URL 或直接上传 POST 带策略.
func UploadFileURLPolicy(c *gin.Context) {
	var req types.UploadFilesRequestPolicy
	if err := c.ShouldBind(&req); err != nil {
		l := log.Logger()
		l.Warn().Err(err).Msg("invalid request")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		return
	}

	handleUpload(c, &req, func(ctx context.Context, user string, r any) (any, error) {
		svc := service.NewFileService(ctx)
		return svc.PresignedPostURLsPolicy(ctx, user, r.(*types.UploadFilesRequestPolicy)) //nolint
	}, "presigned URL")
}

// UploadFileURL 处理文件上传请求：生成预签名 URL PUT 不带策略.
func UploadFileURL(c *gin.Context) {
	var req types.UploadFilesRequest
	if err := c.ShouldBind(&req); err != nil {
		l := log.Logger()
		l.Warn().Err(err).Msg("invalid request")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		return
	}

	handleUpload(c, &req, func(ctx context.Context, user string, r any) (any, error) {
		svc := service.NewFileService(ctx)
		return svc.PresignedPutURLs(ctx, user, r.(*types.UploadFilesRequest)) //nolint
	}, "presigned PUT URL")
}

// handleUpload 通用上传请求处理函数，封装共同的逻辑.
func handleUpload(c *gin.Context, req any, serviceFunc func(context.Context, string, any) (any, error), logMsg string) {
	l := log.Logger()

	// 绑定请求，对 req 进行类型断言，
	var files []types.UploadFileItem
	switch r := req.(type) {
	case *types.UploadFilesRequestPolicy:
		files = r.Files
	case *types.UploadFilesRequest:
		files = r.Files
	default:
		l.Error().Msg("invalid request type")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid request type"})

		return
	}

	if len(files) == 0 {
		l.Warn().Msg("no files provided in request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "no files provided"})

		return
	}

	// 检查用户身份
	user, err := checkUser(c)
	if user == "" || err != nil {
		l.Warn().Err(err).Msg("missing or invalid user")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user"})

		return
	}

	l.Debug().
		Int("file_count", len(files)).
		Str("user", user).
		Msg("processing batch upload request")

	// serviceFunc 通过 svc.PresignedPutURLs 或 svc.PresignedPostURLsPolicy 调用
	res, err := serviceFunc(c.Request.Context(), user, req)
	if err != nil {
		l.Error().Err(err).Msg("failed to generate " + logMsg)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	// 统计结果数量
	var resultCount int

	switch r := res.(type) {
	case *types.UploadFilesResponsePolicy:
		resultCount = len(r.Results)
	case *types.UploadFilesResponse:
		resultCount = len(r.Results)
	default:
		resultCount = 0
	}

	l.Info().Int("result_count", resultCount).Msg("successfully generated " + logMsg + "s")
	c.JSON(http.StatusOK, res)
}

// GetDownloadURL 处理获取文件访问 URL（单个/批量）.
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
