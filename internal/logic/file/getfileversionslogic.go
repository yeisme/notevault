package file

import (
	"context"
	"errors"
	"fmt"

	"github.com/yeisme/notevault/internal/svc"
	"github.com/yeisme/notevault/internal/types"
	"github.com/yeisme/notevault/pkg/storage/repository/dao"
	"gorm.io/gorm"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetFileVersionsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取文件的版本历史。
func NewGetFileVersionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFileVersionsLogic {
	return &GetFileVersionsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetFileVersionsLogic) GetFileVersions(req *types.GetFileVersionsRequest) (resp *types.GetFileVersionsResponse, err error) {
	// Initialize GORM Gen query builder
	query := dao.Use(l.svcCtx.DB)

	// First check if file exists and not deleted
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

	// Query all versions for this file, sorted by version number descending
	fileVersions, err := query.FileVersion.Where(
		query.FileVersion.FileID.Eq(req.FileID),
		query.FileVersion.DeletedAt.Eq(0),
	).Order(query.FileVersion.VersionNumber.Desc()).Find()

	if err != nil {
		l.Error("failed to query file versions", logx.Field("fileID", req.FileID), logx.Field("error", err))
		return nil, fmt.Errorf("failed to query file versions: %w", err)
	}

	// Convert to response format
	versions := make([]types.FileVersionInfo, 0, len(fileVersions))
	for _, version := range fileVersions {
		versionInfo := types.FileVersionInfo{
			Version:       int(version.VersionNumber),
			Size:          version.Size,
			CreatedAt:     version.CreatedAt,
			ContentType:   version.ContentType,
			CommitMessage: version.CommitMessage,
		}
		versions = append(versions, versionInfo)
	}

	resp = &types.GetFileVersionsResponse{
		FileID:   req.FileID,
		Versions: versions,
	}

	l.Infof("Retrieved %d versions for file %s", len(versions), req.FileID)
	return resp, nil
}
