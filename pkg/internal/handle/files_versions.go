package handle

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/yeisme/notevault/pkg/internal/service"
	"github.com/yeisme/notevault/pkg/internal/types"
	"github.com/yeisme/notevault/pkg/log"
)

// ListFileVersions 获取指定文件的版本列表.
//
//	@Summary		获取文件版本列表
//	@Description	根据对象键查询其版本列表（当前实现返回最新可见版本信息，后端如启用版本化可扩展）
//	@Tags			文件版本
//	@Produce		json
//	@Param			fileId	path		string	true	"对象键（完整 object_key）"
//	@Param			scope	query		string	false	"版本范围：current(默认) 或 all"
//	@Success		200		{object}	types.ListFileVersionsResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Router			/api/v1/files/versions/{fileId} [get]
func ListFileVersions(c *gin.Context) {
	l := log.Logger()

	fileID := c.Param("fileId")
	if fileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing fileId"})
		return
	}

	user, err := checkUser(c)
	if user == "" || err != nil {
		l.Warn().Err(err).Msg("missing or invalid user")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user"})

		return
	}

	scope := c.Query("scope") // "current" or "all" (default: current) ?scope=all

	svc := service.NewFileService(c.Request.Context())

	resp, err := svc.ListFileVersions(c.Request.Context(), user, fileID, scope)
	if err != nil {
		l.Error().Err(err).Msg("list versions failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusOK, resp)
}

// CreateFileVersion 创建新版本.
//
//	@Summary		创建文件新版本
//	@Description	将指定对象拷贝到自身生成新版本；可选基于某版本并覆盖元数据
//	@Tags			文件版本
//	@Accept			json
//	@Produce		json
//	@Param			fileId	path		string							true	"对象键（完整 object_key）"
//	@Param			req		body		types.CreateFileVersionRequest	true	"创建版本请求"
//	@Success		200		{object}	types.CreateFileVersionResponse
//	@Failure		400		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Router			/api/v1/files/versions/{fileId} [post]
func CreateFileVersion(c *gin.Context) {
	l := log.Logger()

	fileID := c.Param("fileId")
	if fileID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing fileId"})
		return
	}

	var req types.CreateFileVersionRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// 如果 body 未提供 object_key，则使用路径参数
	if req.ObjectKey == "" {
		req.ObjectKey = fileID
	}

	user, err := checkUser(c)
	if user == "" || err != nil {
		l.Warn().Err(err).Msg("missing or invalid user")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user"})

		return
	}

	svc := service.NewFileService(c.Request.Context())

	resp, err := svc.CreateFileVersion(c.Request.Context(), user, &req)
	if err != nil {
		l.Error().Err(err).Msg("create version failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusOK, resp)
}

// DeleteFileVersion 删除指定版本.
//
//	@Summary	删除指定文件版本
//	@Tags		文件版本
//	@Produce	json
//	@Param		fileId		path		string	true	"对象键（完整 object_key）"
//	@Param		versionId	path		string	true	"版本ID"
//	@Success	200			{object}	types.DeleteFileVersionResponse
//	@Failure	400			{object}	map[string]string
//	@Failure	500			{object}	map[string]string
//	@Router		/api/v1/files/versions/{fileId}/{versionId} [delete]
func DeleteFileVersion(c *gin.Context) {
	handleVersionOperation(c, "delete version", func(ctx context.Context, svc *service.FileService, user, fileID, versionID string) (any, error) {
		return svc.DeleteFileVersion(ctx, user, fileID, versionID)
	})
}

// RestoreFileVersion 恢复指定版本为最新版本.
//
//	@Summary	恢复到指定文件版本
//	@Tags		文件版本
//	@Produce	json
//	@Param		fileId		path		string	true	"对象键（完整 object_key）"
//	@Param		versionId	path		string	true	"版本ID"
//	@Success	200			{object}	types.RestoreFileVersionResponse
//	@Failure	400			{object}	map[string]string
//	@Failure	500			{object}	map[string]string
//	@Router		/api/v1/files/versions/{fileId}/{versionId}/restore [post]
func RestoreFileVersion(c *gin.Context) {
	handleVersionOperation(c, "restore version", func(ctx context.Context, svc *service.FileService, user, fileID, versionID string) (any, error) {
		return svc.RestoreFileVersion(ctx, user, fileID, versionID)
	})
}

// handleVersionOperation 抽取公共处理逻辑，减少 Delete/Restore 的重复代码.
func handleVersionOperation(
	c *gin.Context,
	opName string,
	fn func(ctx context.Context, svc *service.FileService, user, fileID, versionID string) (any, error),
) {
	l := log.Logger()

	fileID := c.Param("fileId")

	versionID := c.Param("versionId")
	if fileID == "" || versionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing fileId or versionId"})
		return
	}

	user, err := checkUser(c)
	if user == "" || err != nil {
		l.Warn().Err(err).Msg("missing or invalid user")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user"})

		return
	}

	svc := service.NewFileService(c.Request.Context())

	resp, err := fn(c.Request.Context(), svc, user, fileID, versionID)
	if err != nil {
		l.Error().Err(err).Msg(opName + " failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusOK, resp)
}
