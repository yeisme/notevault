package file

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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

type UpdateFileMetadataLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 更新特定文件的元数据。这通常会创建一个新版本。
func NewUpdateFileMetadataLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateFileMetadataLogic {
	return &UpdateFileMetadataLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateFileMetadataLogic) UpdateFileMetadata(req *types.UpdateFileMetadataRequest) (resp *types.UpdateFileMetadataResponse, err error) {
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
		return nil, fmt.Errorf("no permission to update this file")
	}

	// Check if there are any changes
	hasChanges := false
	updates := make(map[string]any)

	// Check file name change
	if req.FileName != "" && req.FileName != currentFile.FileName {
		updates["file_name"] = req.FileName
		hasChanges = true
	}

	// Check description change
	if req.Description != currentFile.Description {
		updates["description"] = req.Description
		hasChanges = true
	}

	// If no metadata changes and no tag changes, return error
	tagsChanged := len(req.Tags) > 0
	if !hasChanges && !tagsChanged {
		return nil, fmt.Errorf("no changes detected")
	}

	now := time.Now()
	nowUnix := now.Unix()

	// Prepare commit message
	commitMessage := req.CommitMessage
	if commitMessage == "" {
		commitMessage = "Updated file metadata"
	}

	// Use transaction to ensure data consistency
	err = l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		txQuery := dao.Use(tx)

		// Update file metadata if there are changes
		if hasChanges {
			updates["updated_at"] = nowUnix
			_, err = txQuery.File.Where(txQuery.File.FileID.Eq(req.FileID)).Updates(updates)
			if err != nil {
				return fmt.Errorf("failed to update file metadata: %w", err)
			}
		}

		// Update tags if provided
		if tagsChanged {
			// Delete existing file-tag associations
			_, err = txQuery.FileTag.Where(txQuery.FileTag.FileID.Eq(req.FileID)).Delete()
			if err != nil {
				l.Error("failed to delete existing file tags", logx.Field("fileID", req.FileID), logx.Field("error", err))
				// Continue with the operation, don't fail
			}

			// Add new tags
			for _, tagName := range req.Tags {
				if tagName == "" {
					continue
				}

				// Check if tag exists, create if not
				var tag *model.Tag
				tag, err = txQuery.Tag.Where(txQuery.Tag.Name.Eq(tagName)).First()
				if err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						// Create new tag
						tagID := l.generateTagID()
						newTag := model.Tag{
							TagID: tagID,
							Name:  tagName,
						}
						err = txQuery.Tag.Create(&newTag)
						if err != nil {
							l.Error("failed to create tag", logx.Field("tagName", tagName), logx.Field("error", err))
							continue
						}
						tag = &newTag
					} else {
						l.Error("failed to query tag", logx.Field("tagName", tagName), logx.Field("error", err))
						continue
					}
				}

				// Create file-tag association
				fileTag := model.FileTag{
					FileID: req.FileID,
					TagID:  tag.TagID,
				}
				err = txQuery.FileTag.Create(&fileTag)
				if err != nil {
					l.Error("failed to create file-tag association", logx.Field("fileID", req.FileID), logx.Field("tagID", tag.TagID), logx.Field("error", err))
					continue
				}
			}
		}

		// Create a new version record if there were changes
		if hasChanges {
			newVersion := currentFile.CurrentVersion + 1

			// Update current version in file
			_, err = txQuery.File.Where(txQuery.File.FileID.Eq(req.FileID)).Update(txQuery.File.CurrentVersion, newVersion)
			if err != nil {
				return fmt.Errorf("failed to update file version: %w", err)
			}

			// Create new version record (metadata version)
			newVersionRecord := model.FileVersion{
				VersionID:     fmt.Sprintf("%s_%d", req.FileID, newVersion),
				FileID:        req.FileID,
				VersionNumber: newVersion,
				Size:          currentFile.Size,
				Path:          currentFile.Path,
				ContentType:   currentFile.ContentType,
				CreatedAt:     nowUnix,
				CommitMessage: commitMessage + " (metadata only)",
				Status:        1, // 1=active
			}

			err = txQuery.FileVersion.Create(&newVersionRecord)
			if err != nil {
				return fmt.Errorf("failed to create new version record: %w", err)
			}

			currentFile.CurrentVersion = newVersion
		}

		return nil
	})

	if err != nil {
		l.Error("update metadata transaction failed", logx.Field("fileID", req.FileID), logx.Field("error", err))
		return nil, err
	}

	// Get updated file metadata for response
	updatedFile, err := query.File.Where(
		query.File.FileID.Eq(req.FileID),
		query.File.DeletedAt.Eq(0),
	).First()
	if err != nil {
		l.Error("failed to get updated file", logx.Field("fileID", req.FileID), logx.Field("error", err))
		return nil, fmt.Errorf("failed to get updated file: %w", err)
	}

	// Get updated tags
	var tags []string
	err = query.Tag.Select(query.Tag.Name).
		LeftJoin(query.FileTag, query.FileTag.TagID.EqCol(query.Tag.TagID)).
		Where(query.FileTag.FileID.Eq(req.FileID)).
		Scan(&tags)
	if err != nil {
		l.Error("failed to query updated tags", logx.Field("fileID", req.FileID), logx.Field("error", err))
		tags = []string{} // Continue with empty tags
	}

	// Prepare response
	resp = &types.UpdateFileMetadataResponse{
		Metadata: types.FileMetadata{
			FileID:      updatedFile.FileID,
			UserID:      updatedFile.UserID,
			FileName:    updatedFile.FileName,
			FileType:    updatedFile.FileType,
			ContentType: updatedFile.ContentType,
			Size:        updatedFile.Size,
			Path:        updatedFile.Path,
			CreatedAt:   updatedFile.CreatedAt,
			UpdatedAt:   updatedFile.UpdatedAt,
			Version:     int(updatedFile.CurrentVersion),
			Status:      int16(updatedFile.Status),
			Description: updatedFile.Description,
			Tags:        tags,
		},
		Message: "File metadata updated successfully",
	}

	l.Infof("File metadata updated successfully: fileID=%s, user=%s", req.FileID, userId)
	return resp, nil
}

// generateTagID generates a unique tag ID
func (l *UpdateFileMetadataLogic) generateTagID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
