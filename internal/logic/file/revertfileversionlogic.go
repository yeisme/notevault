package file

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/yeisme/notevault/internal/svc"
	"github.com/yeisme/notevault/internal/types"
	"github.com/yeisme/notevault/pkg/storage/repository/dao"
	"github.com/yeisme/notevault/pkg/storage/repository/model"
	"gorm.io/gorm"

	"github.com/zeromicro/go-zero/core/logx"
)

type RevertFileVersionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 将文件恢复到特定版本。
func NewRevertFileVersionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RevertFileVersionLogic {
	return &RevertFileVersionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RevertFileVersionLogic) RevertFileVersion(req *types.RevertFileVersionRequest) (resp *types.RevertFileVersionResponse, err error) {
	// Get user ID
	userId, ok := l.ctx.Value("userId").(string)
	if !ok || userId == "" {
		userId = "notevault" // Default user, should get from JWT token in practice
	}

	// Initialize GORM Gen query builder
	query := dao.Use(l.svcCtx.DB)

	// Get current file information
	currentFile, err := query.File.Where(
		query.File.FileID.Eq(req.FileID),
		query.File.DeletedAt.Eq(0),
	).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("file not found")
		}
		l.Error("failed to query file", logx.Field("fileID", req.FileID), logx.Field("error", err))
		return nil, fmt.Errorf("failed to query file: %w", err)
	}

	// Check file ownership
	if currentFile.UserID != userId {
		return nil, fmt.Errorf("no permission to revert this file")
	}

	// Get the target version to revert to
	targetVersion, err := query.FileVersion.Where(
		query.FileVersion.FileID.Eq(req.FileID),
		query.FileVersion.VersionNumber.Eq(int32(req.Version)),
		query.FileVersion.DeletedAt.Eq(0),
	).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("target version %d not found", req.Version)
		}
		l.Error("failed to query target version", logx.Field("fileID", req.FileID), logx.Field("version", req.Version), logx.Field("error", err))
		return nil, fmt.Errorf("failed to query target version: %w", err)
	}

	// If current version is the same as target version, no need to revert
	if currentFile.CurrentVersion == int32(req.Version) {
		return nil, fmt.Errorf("file is already at version %d", req.Version)
	}

	now := time.Now()
	nowUnix := now.Unix()
	yearMonth := now.Format("200601")

	// Copy the target version file to a new location
	newPath := fmt.Sprintf("%s/%s/%s_v%d", userId, yearMonth, req.FileID, currentFile.CurrentVersion+1)
	err = l.copyFile(targetVersion.Path, newPath)
	if err != nil {
		l.Error("failed to copy file", logx.Field("from", targetVersion.Path), logx.Field("to", newPath), logx.Field("error", err))
		return nil, fmt.Errorf("failed to copy file: %w", err)
	}

	// Prepare commit message
	commitMessage := req.CommitMessage
	if commitMessage == "" {
		commitMessage = fmt.Sprintf("Reverted to version %d", req.Version)
	}

	// Use transaction to ensure data consistency
	err = l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		txQuery := dao.Use(tx)

		// Update file record
		newVersion := currentFile.CurrentVersion + 1
		_, err = txQuery.File.Where(txQuery.File.FileID.Eq(req.FileID)).Updates(map[string]any{
			"current_version": newVersion,
			"path":            newPath,
			"size":            targetVersion.Size,
			"content_type":    targetVersion.ContentType,
			"updated_at":      nowUnix,
		})
		if err != nil {
			return fmt.Errorf("failed to update file record: %w", err)
		}

		// Create new version record
		newVersionRecord := model.FileVersion{
			VersionID:     fmt.Sprintf("%s_%d", req.FileID, newVersion),
			FileID:        req.FileID,
			VersionNumber: newVersion,
			Size:          targetVersion.Size,
			Path:          newPath,
			ContentType:   targetVersion.ContentType,
			CreatedAt:     nowUnix,
			CommitMessage: commitMessage,
			Status:        1, // 1=active
		}

		err = txQuery.FileVersion.Create(&newVersionRecord)
		if err != nil {
			return fmt.Errorf("failed to create new version record: %w", err)
		}

		return nil
	})

	if err != nil {
		// Clean up the copied file if transaction fails
		l.cleanupFile(newPath)
		l.Error("revert transaction failed", logx.Field("fileID", req.FileID), logx.Field("error", err))
		return nil, err
	}

	// Prepare response
	resp = &types.RevertFileVersionResponse{
		Metadata: types.FileMetadata{
			FileID:        req.FileID,
			UserID:        userId,
			FileName:      currentFile.FileName,
			FileType:      currentFile.FileType,
			ContentType:   targetVersion.ContentType,
			Size:          targetVersion.Size,
			Path:          newPath,
			CreatedAt:     currentFile.CreatedAt,
			UpdatedAt:     nowUnix,
			Version:       int(currentFile.CurrentVersion + 1),
			Status:        int16(currentFile.Status),
			Description:   currentFile.Description,
			CommitMessage: commitMessage,
		},
		Message: fmt.Sprintf("File reverted from version %d to version %d", currentFile.CurrentVersion, req.Version),
	}

	l.Infof("File reverted successfully: fileID=%s, from version %d to version %d", req.FileID, currentFile.CurrentVersion, req.Version)
	return resp, nil
}

// copyFile copies a file from source path to destination path in OSS
func (l *RevertFileVersionLogic) copyFile(srcPath, destPath string) error {
	// Get source object
	srcObject, err := l.svcCtx.OSS.GetObject(
		l.ctx,
		l.svcCtx.Config.Storage.Oss.BucketName,
		srcPath,
		minio.GetObjectOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to get source object: %w", err)
	}
	defer srcObject.Close()

	// Get object info for content type
	srcInfo, err := l.svcCtx.OSS.StatObject(
		l.ctx,
		l.svcCtx.Config.Storage.Oss.BucketName,
		srcPath,
		minio.StatObjectOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to get source object info: %w", err)
	}

	// Copy to destination
	_, err = l.svcCtx.OSS.PutObject(
		l.ctx,
		l.svcCtx.Config.Storage.Oss.BucketName,
		destPath,
		srcObject,
		srcInfo.Size,
		minio.PutObjectOptions{ContentType: srcInfo.ContentType},
	)
	if err != nil {
		return fmt.Errorf("failed to put destination object: %w", err)
	}

	return nil
}

// cleanupFile removes a file from OSS
func (l *RevertFileVersionLogic) cleanupFile(path string) {
	err := l.svcCtx.OSS.RemoveObject(
		l.ctx,
		l.svcCtx.Config.Storage.Oss.BucketName,
		path,
		minio.RemoveObjectOptions{},
	)
	if err != nil {
		l.Error("failed to cleanup file", logx.Field("path", path), logx.Field("error", err))
	}
}
