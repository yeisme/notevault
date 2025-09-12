package service

import (
	"context"

	"github.com/yeisme/notevault/pkg/internal/types"
)

// PresignedGetURLs 生成对象的预签名 GET 访问 URL（支持单个/批量）.
func (fs *FileService) PresignedGetURLs(ctx context.Context, req *types.GetFilesURLRequest) (*types.GetFilesURLResponse, error) {
	bucket, err := fs.defaultBucket()
	if err != nil {
		return nil, err
	}

	expiry := resolveGetExpiry(req)
	results := make([]types.PresignedDownloadItem, 0, len(req.Objects))

	for i := range req.Objects {
		item := &req.Objects[i]

		d, err := fs.presignGet(ctx, bucket, item, expiry)
		if err != nil {
			return nil, err
		}

		results = append(results, d)
	}

	return &types.GetFilesURLResponse{Results: results}, nil
}
