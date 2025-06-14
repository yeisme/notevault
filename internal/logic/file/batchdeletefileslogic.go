package file

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/yeisme/notevault/internal/svc"
	"github.com/yeisme/notevault/internal/types"
	"github.com/yeisme/notevault/pkg/storage/repository/dao"
	"gorm.io/gorm"

	"github.com/zeromicro/go-zero/core/logx"
)

type BatchDeleteFilesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 批量删除文件。
func NewBatchDeleteFilesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchDeleteFilesLogic {
	return &BatchDeleteFilesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *BatchDeleteFilesLogic) BatchDeleteFiles(req *types.BatchDeleteFilesRequest) (resp *types.BatchDeleteFilesResponse, err error) {
	// Get user ID
	userId, ok := l.ctx.Value("userId").(string)
	if !ok || userId == "" {
		userId = "notevault" // Default user, should get from JWT token in practice
	}

	// Initialize response
	resp = &types.BatchDeleteFilesResponse{
		Succeeded: make([]string, 0),
		Failed:    make([]string, 0),
	}

	if len(req.FileIDs) == 0 {
		resp.Message = "No files to move to trash"
		return resp, nil
	}

	// Initialize GORM Gen query builder
	query := dao.Use(l.svcCtx.DB)

	successCount := 0
	failureCount := 0

	// Process each file ID
	for _, fileID := range req.FileIDs {
		err := l.deleteFile(fileID, userId, query)
		if err != nil {
			resp.Failed = append(resp.Failed, fileID)
			failureCount++
			l.Error("failed to move file to trash", logx.Field("fileID", fileID), logx.Field("error", err))
		} else {
			resp.Succeeded = append(resp.Succeeded, fileID)
			successCount++
		}
	}

	resp.Message = fmt.Sprintf("Batch move to trash completed: %d succeeded, %d failed", successCount, failureCount)
	l.Infof("Batch move to trash completed for user %s: %d succeeded, %d failed", userId, successCount, failureCount)

	return resp, nil
}

// deleteFile performs the trash operation for a single file
func (l *BatchDeleteFilesLogic) deleteFile(fileID, userId string, query *dao.Query) error {
	// Find file
	file, err := query.File.Where(
		query.File.FileID.Eq(fileID),
		query.File.DeletedAt.Eq(0),
	).First()

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("file not found or already moved to trash")
		}
		return fmt.Errorf("failed to query file: %w", err)
	}

	// Check file ownership
	if file.UserID != userId {
		return fmt.Errorf("no permission to move this file to trash")
	}

	now := time.Now().Unix()

	// Use transaction to ensure data consistency
	return l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		txQuery := dao.Use(tx)

		// 1. Soft delete file record (keep OSS file)
		_, err = txQuery.File.Where(txQuery.File.FileID.Eq(fileID)).Updates(map[string]any{
			"deleted_at": now,
			"status":     3, // 3=pending deletion (user trash)
			"trashed_at": now,
			"updated_at": now,
		})
		if err != nil {
			return fmt.Errorf("failed to move file record to trash: %w", err)
		}

		// 2. Soft delete all related file versions (keep OSS files)
		_, err = txQuery.FileVersion.Where(txQuery.FileVersion.FileID.Eq(fileID)).Updates(map[string]any{
			"deleted_at": now,
			"status":     2, // 2=replaced/deleted
		})
		if err != nil {
			l.Error("failed to delete file versions", logx.Field("fileID", fileID), logx.Field("error", err))
			// Version deletion failure does not affect main operation, just log
		}

		// 3. Delete file tag associations (physical deletion of association table records)
		_, err = txQuery.FileTag.Where(txQuery.FileTag.FileID.Eq(fileID)).Delete()
		if err != nil {
			l.Error("failed to delete file tag associations", logx.Field("fileID", fileID), logx.Field("error", err))
			// Tag association deletion failure does not affect main operation, just log
		}

		return nil
	})
}
