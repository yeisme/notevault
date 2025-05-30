package file

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/yeisme/notevault/internal/svc"
	"github.com/yeisme/notevault/internal/types"
	"github.com/yeisme/notevault/pkg/storage/repository/dao"
	"github.com/yeisme/notevault/pkg/storage/repository/model"
	"gorm.io/gorm"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteFileLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// Delete file by file ID.
func NewDeleteFileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteFileLogic {
	return &DeleteFileLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteFileLogic) DeleteFile(req *types.FileDeleteRequest) (resp *types.FileDeleteResponse, err error) {
	// Get user ID
	userId, ok := l.ctx.Value("userId").(string)
	if !ok || userId == "" {
		userId = "notevault" // Default user, should get from JWT token in practice
	}

	// Initialize GORM Gen query builder
	query := dao.Use(l.svcCtx.DB)

	// Find file
	var file *model.File
	file, err = query.File.Where(
		query.File.FileID.Eq(req.FileID),
		query.File.DeletedAt.Eq(0),
	).First()

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("file not found or already deleted")
		}
		l.Error("failed to query file", logx.Field("fileID", req.FileID), logx.Field("error", err))
		return nil, fmt.Errorf("failed to query file: %w", err)
	}

	// Check file ownership (if needed)
	if file.UserID != userId {
		return nil, fmt.Errorf("no permission to delete this file")
	}

	now := time.Now().Unix()

	// Use transaction to ensure data consistency
	err = l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		txQuery := dao.Use(tx)

		// 1. Soft delete file record (keep OSS file)
		_, err = txQuery.File.Where(txQuery.File.FileID.Eq(req.FileID)).Updates(map[string]any{
			"deleted_at": now,
			"status":     3, // 3=pending deletion
			"updated_at": now,
		})
		if err != nil {
			return fmt.Errorf("failed to delete file record: %w", err)
		}

		// 2. Soft delete all related file versions (keep OSS files)
		_, err = txQuery.FileVersion.Where(txQuery.FileVersion.FileID.Eq(req.FileID)).Updates(map[string]any{
			"deleted_at": now,
			"status":     2, // 2=replaced/deleted
		})
		if err != nil {
			l.Error("failed to delete file versions", logx.Field("fileID", req.FileID), logx.Field("error", err))
			// Version deletion failure does not affect main operation, just log
		}

		// 3. Delete file tag associations (physical deletion of association table records)
		_, err = txQuery.FileTag.Where(txQuery.FileTag.FileID.Eq(req.FileID)).Delete()
		if err != nil {
			l.Error("failed to delete file tag associations", logx.Field("fileID", req.FileID), logx.Field("error", err))
			// Tag association deletion failure does not affect main operation, just log
		}

		return nil
	})

	if err != nil {
		l.Error("file deletion transaction failed", logx.Field("fileID", req.FileID), logx.Field("error", err))
		return nil, err
	}

	// Construct response
	resp = &types.FileDeleteResponse{
		Message: fmt.Sprintf("File deleted successfully (OSS file preserved): user=%s, fileID=%s, fileName=%s", userId, req.FileID, file.FileName),
	}

	l.Infof("File deleted successfully (OSS preserved): user=%s, fileID=%s, fileName=%s", userId, req.FileID, file.FileName)

	return resp, nil
}
