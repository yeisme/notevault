package handle

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/yeisme/notevault/pkg/internal/service"
	"github.com/yeisme/notevault/pkg/internal/types"
	"github.com/yeisme/notevault/pkg/log"
)

// UploadFileURLPolicy 处理上传文件请求：生成预签名 URL 或直接上传 POST 带策略.
//
//	@Summary		生成预签名POST上传URL
//	@Description	为文件上传生成预签名的POST URL，使用策略控制文件类型、大小等限制
//	@Tags			文件上传
//	@Accept			json
//	@Produce		json
//	@Param			files	body		types.UploadFilesRequestPolicy	true	"带策略的文件上传请求"
//	@Success		200		{object}	types.UploadFilesResponsePolicy	"预签名URL和表单数据响应"
//	@Failure		400		{object}	map[string]string				"请求参数错误"
//	@Failure		500		{object}	map[string]string				"服务器内部错误"
//	@Router			/api/v1/files/upload/urls/policy [post]
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
		return svc.PresignedPostURLsPolicy(ctx, user, r.(*types.UploadFilesRequestPolicy)) //nolint:errcheck
	}, "presigned URL")
}

// UploadFileURL 处理文件上传请求：生成预签名 URL PUT 不带策略.
//
//	@Summary		生成预签名PUT上传URL
//	@Description	为文件上传生成预签名的PUT URL，不使用策略控制，客户端可直接PUT上传文件
//	@Tags			文件上传
//	@Accept			json
//	@Produce		json
//	@Param			files	body		types.UploadFilesRequest	true	"文件上传请求"
//	@Success		200		{object}	types.UploadFilesResponse	"预签名URL响应"
//	@Failure		400		{object}	map[string]string			"请求参数错误"
//	@Failure		500		{object}	map[string]string			"服务器内部错误"
//	@Router			/api/v1/files/upload/urls [post]
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
		return svc.PresignedPutURLs(ctx, user, r.(*types.UploadFilesRequest)) //nolint:errcheck
	}, "presigned PUT URL")
}

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

// UploadSingleFile 处理单个小文件上传.
//
//	@Summary		上传单个小文件
//	@Description	直接上传单个小文件，支持自定义文件名、标签、描述等元数据
//	@Tags			文件上传
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			file			formData	file						true	"上传的文件"
//	@Param			file_name		formData	string						false	"自定义文件名"
//	@Param			tags			formData	string						false	"标签(JSON格式)"
//	@Param			description		formData	string						false	"文件描述"
//	@Param			content_type	formData	string						false	"内容类型"
//	@Param			category		formData	string						false	"文件分类"
//	@Param			folder			formData	string						false	"文件夹路径"
//	@Param			is_public		formData	bool						false	"是否公开"
//	@Param			expiry_days		formData	int							false	"过期天数"
//	@Success		200				{object}	types.UploadFileResponse	"文件上传响应"
//	@Failure		400				{object}	map[string]string			"请求参数错误"
//	@Failure		500				{object}	map[string]string			"服务器内部错误"
//	@Router			/api/v1/files/upload/single [post]
func UploadSingleFile(c *gin.Context) {
	l := log.Logger()

	// 获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		l.Warn().Err(err).Msg("failed to get uploaded file")
		c.JSON(http.StatusBadRequest, gin.H{"error": "no file provided"})

		return
	}

	// 检查用户身份
	user, err := checkUser(c)
	if user == "" || err != nil {
		l.Warn().Err(err).Msg("missing or invalid user")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user"})

		return
	}

	// 解析元数据参数
	metadata := &types.UploadFileMetadata{}
	if fileName := c.PostForm("file_name"); fileName != "" {
		metadata.FileName = fileName
	}

	if tagsStr := c.PostForm("tags"); tagsStr != "" {
		// 简单解析JSON格式的标签
		metadata.Tags = parseTagsFromString(tagsStr)
	}

	if desc := c.PostForm("description"); desc != "" {
		metadata.Description = desc
	}

	if contentType := c.PostForm("content_type"); contentType != "" {
		metadata.ContentType = contentType
	}

	if category := c.PostForm("category"); category != "" {
		metadata.Category = category
	}

	if folder := c.PostForm("folder"); folder != "" {
		metadata.Folder = folder
	}

	if isPublicStr := c.PostForm("is_public"); isPublicStr != "" {
		metadata.IsPublic = isPublicStr == "true"
	}

	if expiryStr := c.PostForm("expiry_days"); expiryStr != "" {
		if expiry, parseErr := strconv.Atoi(expiryStr); parseErr == nil {
			metadata.ExpiryDays = expiry
		}
	}

	// 打开文件
	src, err := file.Open()
	if err != nil {
		l.Error().Err(err).Msg("failed to open uploaded file")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process file"})

		return
	}
	defer src.Close()

	// 上传文件
	svc := service.NewFileService(c.Request.Context())

	resp, err := svc.UploadSingleFile(c.Request.Context(), user, file.Filename, src, file.Size, metadata)
	if err != nil {
		l.Error().Err(err).Msg("failed to upload file")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusOK, resp)
}

// parseTagsFromString 解析标签字符串，支持 JSON 格式和 key:value 格式.
func parseTagsFromString(tagsStr string) map[string]string {
	const (
		keyValuePairParts = 2 // 键值对分割后的期望部分数量
		separatorLimit    = 2 // SplitN 的限制参数
	)

	tags := make(map[string]string)

	// 尝试解析 JSON 格式
	if strings.HasPrefix(strings.TrimSpace(tagsStr), "{") {
		if err := json.Unmarshal([]byte(tagsStr), &tags); err == nil {
			return tags
		}
	}

	// 解析 key:value,key:value 格式
	pairs := strings.SplitSeq(tagsStr, ",")
	for pair := range pairs {
		kv := strings.SplitN(strings.TrimSpace(pair), ":", separatorLimit)
		if len(kv) == keyValuePairParts {
			tags[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}

	return tags
}

// UploadBatchFiles 处理批量小文件上传.
//
//	@Summary		批量上传小文件
//	@Description	直接上传多个小文件，支持为每个文件指定元数据
//	@Tags			文件上传
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			files		formData	[]file							true	"上传的文件数组"
//	@Param			metadata	formData	string							false	"元数据JSON数组"
//	@Success		200			{object}	types.UploadBatchFilesResponse	"批量文件上传响应"
//	@Failure		400			{object}	map[string]string				"请求参数错误"
//	@Failure		500			{object}	map[string]string				"服务器内部错误"
//	@Router			/api/v1/files/upload/batch [post]
func UploadBatchFiles(c *gin.Context) {
	l := log.Logger()

	// 获取表单
	form, err := c.MultipartForm()
	if err != nil {
		l.Warn().Err(err).Msg("failed to parse multipart form")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid form data"})

		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		l.Warn().Msg("no files provided in batch upload")
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

	// 解析元数据
	metadataMap := make(map[string]*types.UploadFileMetadata)
	if metadataStr := c.PostForm("metadata"); metadataStr != "" {
		var metadataList []types.UploadFileMetadata
		if unmarshalErr := json.Unmarshal([]byte(metadataStr), &metadataList); unmarshalErr == nil {
			for i := range metadataList {
				if i < len(files) {
					metadataMap[files[i].Filename] = &metadataList[i]
				}
			}
		}
	}

	// 准备文件数据
	fileReaders := make(map[string]io.Reader)
	fileSizes := make(map[string]int64)

	for _, file := range files {
		src, fileOpenError := file.Open()
		if fileOpenError != nil {
			l.Error().Err(fileOpenError).Str("filename", file.Filename).Msg("failed to open file")
			continue
		}
		defer src.Close()

		fileReaders[file.Filename] = src
		fileSizes[file.Filename] = file.Size
	}

	if len(fileReaders) == 0 {
		l.Warn().Msg("no valid files to upload")
		c.JSON(http.StatusBadRequest, gin.H{"error": "no valid files"})

		return
	}

	// 上传文件
	svc := service.NewFileService(c.Request.Context())

	resp, err := svc.UploadBatchFiles(c.Request.Context(), user, fileReaders, fileSizes, metadataMap)
	if err != nil {
		l.Error().Err(err).Msg("failed to upload batch files")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusOK, resp)
}
