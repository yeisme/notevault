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

// DeleteFiles 删除文件（单个/批量）。
//
//	@Summary		删除文件(单个/批量)
//	@Description	根据对象键列表删除文件，支持批量删除。
//	@Tags			文件操作
//	@Accept			json
//	@Produce		json
//	@Param			req	body		types.DeleteFilesRequest	true	"删除请求"
//	@Success		200	{object}	types.DeleteFilesResponse	"删除结果"
//	@Failure		400	{object}	map[string]string			"请求参数错误"
//	@Failure		500	{object}	map[string]string			"服务器内部错误"
//	@Router			/api/v1/files [delete]
func DeleteFiles(c *gin.Context) {
	var req types.DeleteFilesRequest
	handleFilesOperation(c, "delete files", &req,
		func() error {
			if len(req.ObjectKeys) == 0 {
				return fmt.Errorf("no object_keys provided")
			}

			return nil
		},
		func(svc *service.FileService, ctx context.Context, user string) (any, error) {
			return svc.DeleteFiles(ctx, user, &req)
		},
	)
}

// CopyFiles 复制文件（单个/批量）。
//
//	@Summary		复制文件(单个/批量)
//	@Description	将源对象复制到目标对象，支持批量复制。
//	@Tags			文件操作
//	@Accept			json
//	@Produce		json
//	@Param			req	body		types.CopyFilesRequest	true	"复制请求"
//	@Success		200	{object}	types.CopyFilesResponse	"复制结果"
//	@Failure		400	{object}	map[string]string		"请求参数错误"
//	@Failure		500	{object}	map[string]string		"服务器内部错误"
//	@Router			/api/v1/files/copy [post]
func CopyFiles(c *gin.Context) {
	var req types.CopyFilesRequest
	handleFilesOperation(c, "copy files", &req,
		func() error {
			if len(req.Items) == 0 {
				return fmt.Errorf("no items provided")
			}

			return nil
		},
		func(svc *service.FileService, ctx context.Context, user string) (any, error) {
			return svc.CopyFiles(ctx, user, &req)
		},
	)
}

// MoveFiles 移动文件（单个/批量）。
//
//	@Summary		移动文件(单个/批量)
//	@Description	将源对象移动到目标对象，支持批量移动。
//	@Tags			文件操作
//	@Accept			json
//	@Produce		json
//	@Param			req	body		types.MoveFilesRequest	true	"移动请求"
//	@Success		200	{object}	types.MoveFilesResponse	"移动结果"
//	@Failure		400	{object}	map[string]string		"请求参数错误"
//	@Failure		500	{object}	map[string]string		"服务器内部错误"
//	@Router			/api/v1/files/move [post]
func MoveFiles(c *gin.Context) {
	var req types.MoveFilesRequest
	handleFilesOperation(c, "move files", &req,
		func() error {
			if len(req.Items) == 0 {
				return fmt.Errorf("no items provided")
			}

			return nil
		},
		func(svc *service.FileService, ctx context.Context, user string) (any, error) {
			return svc.MoveFiles(ctx, user, &req)
		},
	)
}

// handleFilesOperation 封装公共流程：绑定/校验/用户校验/调用 service/统一返回。
func handleFilesOperation(c *gin.Context, operation string, req any,
	validate func() error,
	serviceCall func(*service.FileService, context.Context, string) (any, error),
) {
	l := log.Logger()

	if err := c.ShouldBind(req); err != nil {
		l.Warn().Err(err).Str("op", operation).Msg("invalid request")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		return
	}

	if err := validate(); err != nil {
		l.Warn().Err(err).Str("op", operation).Msg("invalid payload")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		return
	}

	user, err := checkUser(c)
	if user == "" || err != nil {
		l.Warn().Err(err).Str("op", operation).Msg("missing or invalid user")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user"})

		return
	}

	ctx := c.Request.Context()
	svc := service.NewFileService(ctx)

	resp, err := serviceCall(svc, ctx, user)
	if err != nil {
		l.Error().Err(err).Str("op", operation).Msg("service call failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusOK, resp)
}
