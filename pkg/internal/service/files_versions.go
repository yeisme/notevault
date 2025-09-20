package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"

	"github.com/yeisme/notevault/pkg/internal/types"
)

const defaultVersionsCap = 8

// ListFileVersions 根据 scope 列出对象版本：
//   - scope = "all"：返回所有版本（需要后端启用版本化）
//   - scope = "current"（默认）：仅返回当前可见版本
func (fs *FileService) ListFileVersions(ctx context.Context, user, objectKey, scope string) (*types.ListFileVersionsResponse, error) { //nolint:ireturn
	if user == "" || !strings.HasPrefix(objectKey, user+"/") {
		return nil, fmt.Errorf("access denied: object does not belong to user")
	}

	bucket, err := fs.defaultBucket()
	if err != nil {
		return nil, err
	}

	// 标准化 scope
	scope = strings.ToLower(strings.TrimSpace(scope))

	// 当前版本：直接 StatObject
	if scope == "current" || scope == "" {
		info, err := fs.s3Client.StatObject(ctx, bucket, objectKey, minio.StatObjectOptions{})
		if err != nil {
			return nil, fmt.Errorf("stat object %s: %w", objectKey, err)
		}

		versions := []types.FileVersionInfo{
			{
				ObjectKey:    objectKey,
				VersionID:    info.VersionID,
				IsLatest:     true,
				Size:         info.Size,
				ETag:         strings.Trim(info.ETag, "\""),
				ContentType:  info.ContentType,
				LastModified: info.LastModified.UTC().Format(time.RFC3339),
				StorageClass: info.StorageClass,
				Bucket:       bucket,
				UserMetadata: info.UserMetadata,
			},
		}

		return &types.ListFileVersionsResponse{FileID: objectKey, Versions: versions, Total: len(versions)}, nil
	}

	// 全部版本：ListObjects(WithVersions=true) + StatObject(VersionID)
	opts := minio.ListObjectsOptions{
		Prefix:       objectKey,
		Recursive:    true,
		WithVersions: true,
	}

	ch := fs.s3Client.ListObjects(ctx, bucket, opts)

	versions := make([]types.FileVersionInfo, 0, defaultVersionsCap)

	for obj := range ch {
		if obj.Err != nil {
			return nil, fmt.Errorf("list object versions for %s: %v", objectKey, obj.Err)
		}

		// 仅关注完全匹配该对象键的版本，且跳过删除标记
		if obj.Key != objectKey || obj.IsDeleteMarker {
			continue
		}

		// 通过 StatObject(指定 VersionID) 获取更完整的元数据
		st, stErr := fs.s3Client.StatObject(ctx, bucket, objectKey, minio.StatObjectOptions{VersionID: obj.VersionID})
		if stErr != nil {
			// 若获取失败，退化为使用 ListObjects 返回的信息（缺少 content-type、user-metadata 等）
			versions = append(versions, types.FileVersionInfo{
				ObjectKey:    objectKey,
				VersionID:    obj.VersionID,
				IsLatest:     obj.IsLatest,
				Size:         obj.Size,
				ETag:         strings.Trim(obj.ETag, "\""),
				ContentType:  "",
				LastModified: obj.LastModified.UTC().Format(time.RFC3339),
				StorageClass: obj.StorageClass,
				Bucket:       bucket,
				UserMetadata: nil,
			})

			continue
		}

		versions = append(versions, types.FileVersionInfo{
			ObjectKey:    objectKey,
			VersionID:    st.VersionID,
			IsLatest:     obj.IsLatest,
			Size:         st.Size,
			ETag:         strings.Trim(st.ETag, "\""),
			ContentType:  st.ContentType,
			LastModified: st.LastModified.UTC().Format(time.RFC3339),
			StorageClass: st.StorageClass,
			Bucket:       bucket,
			UserMetadata: st.UserMetadata,
		})
	}

	// 如果未获取到任何版本，作为兜底再尝试 Stat 当前可见版本
	if len(versions) == 0 {
		info, err := fs.s3Client.StatObject(ctx, bucket, objectKey, minio.StatObjectOptions{})
		if err == nil {
			versions = append(versions, types.FileVersionInfo{
				ObjectKey:    objectKey,
				VersionID:    info.VersionID,
				IsLatest:     true,
				Size:         info.Size,
				ETag:         strings.Trim(info.ETag, "\""),
				ContentType:  info.ContentType,
				LastModified: info.LastModified.UTC().Format(time.RFC3339),
				StorageClass: info.StorageClass,
				Bucket:       bucket,
				UserMetadata: info.UserMetadata,
			})
		}
	}

	return &types.ListFileVersionsResponse{FileID: objectKey, Versions: versions, Total: len(versions)}, nil
}

// CreateFileVersion 基于现有对象创建一个新版本（通过拷贝到自身来触发新版本）.
func (fs *FileService) CreateFileVersion(ctx context.Context, user string, req *types.CreateFileVersionRequest) (*types.CreateFileVersionResponse, error) {
	if user == "" || !strings.HasPrefix(req.ObjectKey, user+"/") {
		return nil, fmt.Errorf("access denied: object does not belong to user")
	}

	bucket, err := fs.defaultBucket()
	if err != nil {
		return nil, err
	}

	// 源选项：可选指定基线版本
	src := minio.CopySrcOptions{Bucket: bucket, Object: req.ObjectKey}
	if req.BaseVersion != "" {
		src.VersionID = req.BaseVersion
	}

	// 目标选项：复制到相同 key，以触发新版本
	dst := minio.CopyDestOptions{Bucket: bucket, Object: req.ObjectKey}
	if req.ContentType != "" || len(req.UserMeta) > 0 {
		dst.ReplaceMetadata = true

		dst.ContentType = req.ContentType
		if len(req.UserMeta) > 0 {
			dst.UserMetadata = req.UserMeta
		}
	}

	ui, err := fs.s3Client.CopyObject(ctx, dst, src)
	if err != nil {
		return nil, fmt.Errorf("create new version by copy: %w", err)
	}

	return &types.CreateFileVersionResponse{
		ObjectKey: req.ObjectKey,
		VersionID: ui.VersionID,
		Success:   true,
	}, nil
}

// DeleteFileVersion 删除指定版本.
func (fs *FileService) DeleteFileVersion(ctx context.Context, user, objectKey, versionID string) (*types.DeleteFileVersionResponse, error) {
	if user == "" || !strings.HasPrefix(objectKey, user+"/") {
		return nil, fmt.Errorf("access denied: object does not belong to user")
	}

	bucket, err := fs.defaultBucket()
	if err != nil {
		return nil, err
	}

	err = fs.s3Client.RemoveObject(ctx, bucket, objectKey, minio.RemoveObjectOptions{VersionID: versionID})
	if err != nil {
		return &types.DeleteFileVersionResponse{ObjectKey: objectKey, VersionID: versionID, Success: false, Error: err.Error()}, nil
	}

	return &types.DeleteFileVersionResponse{ObjectKey: objectKey, VersionID: versionID, Success: true}, nil
}

// RestoreFileVersion 基于指定版本恢复为最新版本（复制该版本到同一 key）.
func (fs *FileService) RestoreFileVersion(ctx context.Context, user, objectKey, versionID string) (*types.RestoreFileVersionResponse, error) {
	if user == "" || !strings.HasPrefix(objectKey, user+"/") {
		return nil, fmt.Errorf("access denied: object does not belong to user")
	}

	bucket, err := fs.defaultBucket()
	if err != nil {
		return nil, err
	}

	// 从指定版本拷贝到自身，得到一个新的最新版本
	src := minio.CopySrcOptions{Bucket: bucket, Object: objectKey, VersionID: versionID}
	dst := minio.CopyDestOptions{Bucket: bucket, Object: objectKey}

	ui, err := fs.s3Client.CopyObject(ctx, dst, src)
	if err != nil {
		return &types.RestoreFileVersionResponse{ObjectKey: objectKey, FromVersion: versionID, Success: false, Error: err.Error()}, nil
	}

	return &types.RestoreFileVersionResponse{ObjectKey: objectKey, FromVersion: versionID, RestoredAs: ui.VersionID, Success: true}, nil
}
