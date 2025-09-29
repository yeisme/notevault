package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"

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

		// 发布访问事件（通过预签名链接）
		fs.publishObjectAccessed(bucket, item.ObjectKey, "read", "presigned", "", "")
	}

	return &types.GetFilesURLResponse{Results: results}, nil
}

// StatObject 查询对象信息（包含大小、类型、ETag、最后修改时间等）.
func (fs *FileService) StatObject(ctx context.Context, user, objectKey string) (*types.ObjectInfo, error) {
	if user == "" || !strings.HasPrefix(objectKey, user+"/") {
		return nil, fmt.Errorf("access denied: object does not belong to user")
	}

	bucket, err := fs.defaultBucket()
	if err != nil {
		return nil, err
	}

	info, err := fs.s3Client.StatObject(ctx, bucket, objectKey, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("stat object %s: %w", objectKey, err)
	}

	obj := &types.ObjectInfo{
		ObjectKey:    objectKey,
		Size:         info.Size,
		ETag:         strings.Trim(info.ETag, "\""),
		ContentType:  info.ContentType,
		LastModified: info.LastModified.UTC().Format(time.RFC3339),
		VersionID:    info.VersionID,
		StorageClass: info.StorageClass,
		Bucket:       bucket,
		UserMetadata: info.UserMetadata,
	}

	// 发布访问事件（元信息读取视为一次访问）
	fs.publishObjectAccessed(bucket, objectKey, "head", "server", "", "")

	return obj, nil
}

// OpenObject 打开对象获取可读流与其信息.
func (fs *FileService) OpenObject(ctx context.Context, user, objectKey string) (*minio.Object, *types.ObjectInfo, error) { //nolint:ireturn
	if user == "" || !strings.HasPrefix(objectKey, user+"/") {
		return nil, nil, fmt.Errorf("access denied: object does not belong to user")
	}

	bucket, err := fs.defaultBucket()
	if err != nil {
		return nil, nil, err
	}

	obj, err := fs.s3Client.GetObject(ctx, bucket, objectKey, minio.GetObjectOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("get object %s: %w", objectKey, err)
	}

	// 通过 StatObject 获取对象信息
	info, err := fs.s3Client.StatObject(ctx, bucket, objectKey, minio.StatObjectOptions{})
	if err != nil {
		// 关闭 obj 以避免泄露
		_ = obj.Close()
		return nil, nil, fmt.Errorf("stat object %s: %w", objectKey, err)
	}

	meta := &types.ObjectInfo{
		ObjectKey:    objectKey,
		Size:         info.Size,
		ETag:         strings.Trim(info.ETag, "\""),
		ContentType:  info.ContentType,
		LastModified: info.LastModified.UTC().Format(time.RFC3339),
		VersionID:    info.VersionID,
		StorageClass: info.StorageClass,
		Bucket:       bucket,
		UserMetadata: info.UserMetadata,
	}

	// 发布访问事件（服务端读流）
	fs.publishObjectAccessed(bucket, objectKey, "read", "server", "", "")

	return obj, meta, nil
}
