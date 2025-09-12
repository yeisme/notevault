package handle

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/yeisme/notevault/pkg/internal/service"
	"github.com/yeisme/notevault/pkg/internal/types"
	"github.com/yeisme/notevault/pkg/log"
)

// CreateFolder 处理创建文件夹请求.
//
//	@Summary		创建文件夹
//	@Description	创建新的文件夹，支持指定父路径和描述
//	@Tags			文件夹管理
//	@Accept			json
//	@Produce		json
//	@Param			folder	body		types.CreateFolderRequest	true	"创建文件夹请求"
//	@Success		201		{object}	types.CreateFolderResponse	"文件夹创建响应"
//	@Failure		400		{object}	map[string]string			"请求参数错误"
//	@Failure		500		{object}	map[string]string			"服务器内部错误"
//	@Router			/api/v1/files/folder [post]
func CreateFolder(c *gin.Context) {
	l := log.Logger()

	var req types.CreateFolderRequest
	if err := c.ShouldBind(&req); err != nil {
		l.Warn().Err(err).Msg("invalid create folder request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request parameters"})

		return
	}

	// 检查用户身份
	user, err := checkUser(c)
	if user == "" || err != nil {
		l.Warn().Err(err).Msg("missing or invalid user")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user"})

		return
	}

	svc := service.NewFileService(c.Request.Context())

	resp, err := svc.CreateFolder(c.Request.Context(), user, &req)
	if err != nil {
		l.Error().Err(err).Msg("failed to create folder")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusCreated, resp)
}

// RenameFolder 处理重命名文件夹请求.
//
//	@Summary		重命名文件夹
//	@Description	重命名指定的文件夹
//	@Tags			文件夹管理
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"文件夹ID"
//	@Param			folder	body		types.RenameFolderRequest	true	"重命名文件夹请求"
//	@Success		200		{object}	types.RenameFolderResponse	"文件夹重命名响应"
//	@Failure		400		{object}	map[string]string			"请求参数错误"
//	@Failure		404		{object}	map[string]string			"文件夹不存在"
//	@Failure		500		{object}	map[string]string			"服务器内部错误"
//	@Router			/api/v1/files/folder/{id} [put]
func RenameFolder(c *gin.Context) {
	handleFolderWithID(c, "rename", func(svc *service.FileService, ctx context.Context, user, folderID string, req any) (any, error) {
		renameReq, ok := req.(*types.RenameFolderRequest)
		if !ok {
			return nil, fmt.Errorf("invalid request type")
		}

		return svc.RenameFolder(ctx, user, folderID, renameReq)
	})
}

// DeleteFolder 处理删除文件夹请求.
//
//	@Summary		删除文件夹
//	@Description	删除指定的文件夹，支持递归删除
//	@Tags			文件夹管理
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"文件夹ID"
//	@Param			folder	body		types.DeleteFolderRequest	false	"删除文件夹请求"
//	@Success		200		{object}	types.DeleteFolderResponse	"文件夹删除响应"
//	@Failure		400		{object}	map[string]string			"请求参数错误"
//	@Failure		404		{object}	map[string]string			"文件夹不存在"
//	@Failure		500		{object}	map[string]string			"服务器内部错误"
//	@Router			/api/v1/files/folder/{id} [delete]
func DeleteFolder(c *gin.Context) {
	handleFolderWithID(c, "delete", func(svc *service.FileService, ctx context.Context, user, folderID string, req any) (any, error) {
		deleteReq, ok := req.(*types.DeleteFolderRequest)
		if !ok {
			return nil, fmt.Errorf("invalid request type")
		}

		return svc.DeleteFolder(ctx, user, folderID, deleteReq)
	})
}

// handleFolderWithID 处理需要文件夹ID的操作.
func handleFolderWithID(c *gin.Context, operation string, serviceFunc func(*service.FileService, context.Context, string, string, any) (any, error)) {
	l := log.Logger()

	folderID := c.Param("id")
	if folderID == "" {
		l.Warn().Msg("missing folder ID")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing folder ID"})

		return
	}

	// 检查用户身份
	user, err := checkUser(c)
	if user == "" || err != nil {
		l.Warn().Err(err).Msg("missing or invalid user")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user"})

		return
	}

	svc := service.NewFileService(c.Request.Context())

	var (
		req     any
		bindErr error
	)

	switch operation {
	case "rename":
		var renameReq types.RenameFolderRequest

		bindErr = c.ShouldBind(&renameReq)
		req = &renameReq
	case "delete":
		var deleteReq types.DeleteFolderRequest

		bindErr = c.ShouldBind(&deleteReq)
		req = &deleteReq
	}

	if bindErr != nil {
		l.Warn().Err(bindErr).Msgf("invalid %s folder request", operation)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request parameters"})

		return
	}

	resp, err := serviceFunc(svc, c.Request.Context(), user, folderID, req)
	if err != nil {
		l.Error().Err(err).Msgf("failed to %s folder", operation)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusOK, resp)
}
