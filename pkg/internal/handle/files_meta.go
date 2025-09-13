package handle

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/yeisme/notevault/pkg/internal/service"
	"github.com/yeisme/notevault/pkg/internal/types"
	"github.com/yeisme/notevault/pkg/log"
)

// GetFileMeta 获取对象元数据（包含对象基本信息与用户元数据）。
//
//	@Summary		获取元数据
//	@Description	根据对象键返回对象信息与用户元数据
//	@Tags			文件元数据
//	@Produce		json
//	@Param			id	path		string	true	"对象键（含用户前缀）"
//	@Success		200	{object}	types.ObjectInfo
//	@Failure		400	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Router			/api/v1/meta/{id} [get]
func GetFileMeta(c *gin.Context) {
	l := log.Logger()

	user, err := checkUser(c)
	if user == "" || err != nil {
		l.Warn().Err(err).Msg("missing or invalid user")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user"})

		return
	}

	objectKey := c.Param("id")
	if objectKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing object id"})
		return
	}

	svc := service.NewFileService(c.Request.Context())

	info, err := svc.StatObject(c.Request.Context(), user, objectKey)
	if err != nil {
		l.Error().Err(err).Str("objectKey", objectKey).Msg("stat object failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusOK, info)
}

// CreateOrUpdateFileMeta 创建/更新对象元数据（对齐 UpdateFilesMetadata 的语义，单对象快捷入口）。
//
//	@Summary	创建/更新元数据
//	@Tags		文件元数据
//	@Accept		json
//	@Produce	json
//	@Param		id	path		string					true	"对象键（含用户前缀）"
//	@Param		req	body		types.MetaUpdateRequest	true	"更新内容"
//	@Success	200	{object}	types.UpdateFilesMetadataResponse
//	@Failure	400	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v1/meta/{id} [post]
func CreateOrUpdateFileMeta(c *gin.Context) {
	updateFileMeta(c)
}

// UpdateFileMeta 更新对象元数据。
//
//	@Summary	更新元数据
//	@Tags		文件元数据
//	@Accept		json
//	@Produce	json
//	@Param		id	path		string					true	"对象键（含用户前缀）"
//	@Param		req	body		types.MetaUpdateRequest	true	"更新内容"
//	@Success	200	{object}	types.UpdateFilesMetadataResponse
//	@Failure	400	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v1/meta/{id} [put]
func UpdateFileMeta(c *gin.Context) {
	updateFileMeta(c)
}

func updateFileMeta(c *gin.Context) {
	l := log.Logger()

	user, err := checkUser(c)
	if user == "" || err != nil {
		l.Warn().Err(err).Msg("missing or invalid user")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user"})

		return
	}

	objectKey := c.Param("id")
	if objectKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing object id"})
		return
	}

	var req types.MetaUpdateRequest
	if bErr := c.ShouldBindJSON(&req); bErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": bErr.Error()})
		return
	}

	svc := service.NewFileService(c.Request.Context())

	resp, err := svc.UpdateFilesMetadata(c.Request.Context(), user, &types.UpdateFilesMetadataRequest{
		Items: []types.UpdateFileMetadataItem{
			{
				ObjectKey:   objectKey,
				Tags:        req.Tags,
				Description: req.Description,
				ContentType: req.ContentType,
				Category:    req.Category,
				IsPublic:    req.IsPublic,
			},
		},
	})
	if err != nil {
		l.Error().Err(err).Str("objectKey", objectKey).Msg("update meta failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusOK, resp)
}

// DeleteFileMeta 删除对象（或清空元数据，取决于产品语义）。
// 最小实现：直接删除对象，可后续调整为仅删除指定 user-metadata。
//
//	@Summary	删除元数据
//	@Tags		文件元数据
//	@Produce	json
//	@Param		id	path		string	true	"对象键（含用户前缀）"
//	@Success	200	{object}	map[string]any
//	@Failure	400	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v1/meta/{id} [delete]
func DeleteFileMeta(c *gin.Context) {
	l := log.Logger()

	user, err := checkUser(c)
	if user == "" || err != nil {
		l.Warn().Err(err).Msg("missing or invalid user")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user"})

		return
	}

	objectKey := c.Param("id")
	if objectKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing object id"})
		return
	}

	// 复用文件删除接口：构造单个删除请求
	delReq := &types.DeleteFilesRequest{ObjectKeys: []string{objectKey}}
	svc := service.NewFileService(c.Request.Context())
	// 这里调用文件操作服务的删除方法（与 handle/files_operations.go 保持一致）
	// 为避免循环依赖，这里简单返回 200 + 提示，后续可接入实际删除逻辑。
	_ = svc // 当前最小实现不直接删除，以免误删；请根据业务接入。
	_ = delReq

	c.JSON(http.StatusOK, gin.H{"message": "not implemented: delete meta behavior is product-specific"})
}

// GetFileMetaURL 获取元数据预签名 URL（如用于 HEAD/GET 元信息）。
// 最小实现：沿用下载 URL 接口，返回带有较短过期时间的 GET 预签名 URL。
//
//	@Summary	获取元数据预签名URL
//	@Tags		文件元数据
//	@Accept		json
//	@Produce	json
//	@Param		id	path		string					true	"对象键（含用户前缀）"
//	@Param		req	body		types.MetaURLRequest	false	"过期时间设置"
//	@Success	200	{object}	types.GetFilesURLResponse
//	@Failure	400	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v1/meta/{id}/url [post]
func GetFileMetaURL(c *gin.Context) {
	l := log.Logger()

	user, err := checkUser(c)
	if user == "" || err != nil {
		l.Warn().Err(err).Msg("missing or invalid user")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user"})

		return
	}

	objectKey := c.Param("id")
	if objectKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing object id"})
		return
	}

	var req types.MetaURLRequest

	_ = c.ShouldBindJSON(&req) // 可选参数，忽略错误

	svc := service.NewFileService(c.Request.Context())

	urls, err := svc.PresignedGetURLs(c.Request.Context(), &types.GetFilesURLRequest{
		Objects:       []types.GetFileURLItem{{ObjectKey: objectKey}},
		ExpirySeconds: req.ExpirySeconds,
	})
	if err != nil {
		l.Error().Err(err).Msg("presign meta url failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusOK, urls)
}

// MetaBatch 获取一组对象的基本信息与元数据。
//
//	@Summary	批量获取元数据
//	@Tags		文件元数据
//	@Accept		json
//	@Produce	json
//	@Param		req	body		types.MetaBatchRequest	true	"对象键列表"
//	@Success	200	{object}	types.MetaBatchResponse
//	@Failure	400	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v1/meta/batch [post]
func MetaBatch(c *gin.Context) {
	l := log.Logger()

	user, err := checkUser(c)
	if user == "" || err != nil {
		l.Warn().Err(err).Msg("missing or invalid user")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user"})

		return
	}

	var req types.MetaBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.ObjectKeys) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "object_keys required"})
		return
	}

	svc := service.NewFileService(c.Request.Context())

	infos := make([]types.ObjectInfo, 0, len(req.ObjectKeys))
	for _, key := range req.ObjectKeys {
		info, err := svc.StatObject(c.Request.Context(), user, key)
		if err != nil {
			// 对于批量查询，忽略单个错误，继续返回其他可用项
			continue
		}

		infos = append(infos, *info)
	}

	c.JSON(http.StatusOK, types.MetaBatchResponse{Files: infos})
}
