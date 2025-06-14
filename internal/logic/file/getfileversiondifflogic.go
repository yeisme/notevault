package file

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/yeisme/notevault/internal/svc"
	"github.com/yeisme/notevault/internal/types"
	"github.com/yeisme/notevault/pkg/storage/repository/dao"
	"gorm.io/gorm"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetFileVersionDiffLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// (高级) 获取文件两个版本之间的差异信息 (主要针对文本文件)。
func NewGetFileVersionDiffLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFileVersionDiffLogic {
	return &GetFileVersionDiffLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetFileVersionDiffLogic) GetFileVersionDiff(req *types.FileVersionDiffRequest) (resp *types.FileVersionDiffResponse, err error) {
	// Initialize GORM Gen query builder
	query := dao.Use(l.svcCtx.DB)

	// Check if file exists and not deleted
	_, err = query.File.Where(
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

	// Get base version
	baseVersion, err := query.FileVersion.Where(
		query.FileVersion.FileID.Eq(req.FileID),
		query.FileVersion.VersionNumber.Eq(int32(req.BaseVersion)),
		query.FileVersion.DeletedAt.Eq(0),
	).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("base version %d not found", req.BaseVersion)
		}
		l.Error("failed to query base version", logx.Field("fileID", req.FileID), logx.Field("version", req.BaseVersion), logx.Field("error", err))
		return nil, fmt.Errorf("failed to query base version: %w", err)
	}

	// Get target version
	targetVersion, err := query.FileVersion.Where(
		query.FileVersion.FileID.Eq(req.FileID),
		query.FileVersion.VersionNumber.Eq(int32(req.TargetVersion)),
		query.FileVersion.DeletedAt.Eq(0),
	).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("target version %d not found", req.TargetVersion)
		}
		l.Error("failed to query target version", logx.Field("fileID", req.FileID), logx.Field("version", req.TargetVersion), logx.Field("error", err))
		return nil, fmt.Errorf("failed to query target version: %w", err)
	}

	// Check if both versions are text files (can only diff text files)
	if !l.isTextFile(baseVersion.ContentType) || !l.isTextFile(targetVersion.ContentType) {
		return &types.FileVersionDiffResponse{
			FileID:        req.FileID,
			BaseVersion:   req.BaseVersion,
			TargetVersion: req.TargetVersion,
			DiffContent:   "",
			Message:       "Cannot generate diff for non-text files",
		}, nil
	}

	// Get file content from OSS
	baseContent, err := l.getFileContent(baseVersion.Path)
	if err != nil {
		l.Error("failed to get base version content", logx.Field("path", baseVersion.Path), logx.Field("error", err))
		return nil, fmt.Errorf("failed to get base version content: %w", err)
	}

	targetContent, err := l.getFileContent(targetVersion.Path)
	if err != nil {
		l.Error("failed to get target version content", logx.Field("path", targetVersion.Path), logx.Field("error", err))
		return nil, fmt.Errorf("failed to get target version content: %w", err)
	}

	// Generate diff using go-diff library
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(baseContent, targetContent, false)
	diffText := dmp.DiffPrettyText(diffs)

	resp = &types.FileVersionDiffResponse{
		FileID:        req.FileID,
		BaseVersion:   req.BaseVersion,
		TargetVersion: req.TargetVersion,
		DiffContent:   diffText,
		Message:       "Diff generated successfully",
	}

	l.Infof("Generated diff for file %s between versions %d and %d", req.FileID, req.BaseVersion, req.TargetVersion)
	return resp, nil
}

// isTextFile checks if the content type is a text file
func (l *GetFileVersionDiffLogic) isTextFile(contentType string) bool {
	textTypes := []string{
		"text/",
		"application/json",
		"application/xml",
		"application/javascript",
		"application/x-yaml",
		"application/yaml",
	}

	for _, textType := range textTypes {
		if strings.HasPrefix(contentType, textType) {
			return true
		}
	}
	return false
}

// getFileContent retrieves file content from OSS
func (l *GetFileVersionDiffLogic) getFileContent(path string) (string, error) {
	object, err := l.svcCtx.OSS.GetObject(
		l.ctx,
		l.svcCtx.Config.Storage.Oss.BucketName,
		path,
		minio.GetObjectOptions{},
	)
	if err != nil {
		return "", fmt.Errorf("failed to get object from OSS: %w", err)
	}
	defer object.Close()

	// Read content
	content, err := io.ReadAll(object)
	if err != nil {
		return "", fmt.Errorf("failed to read object content: %w", err)
	}

	return string(content), nil
}
