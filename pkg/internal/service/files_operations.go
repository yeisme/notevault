package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/minio/minio-go/v7"

	"github.com/yeisme/notevault/pkg/internal/types"
)

// DeleteFiles 删除文件（支持单个/批量）.
func (fs *FileService) DeleteFiles(ctx context.Context, user string, req *types.DeleteFilesRequest) (*types.DeleteFilesResponse, error) {
	results := make([]types.DeleteFileResult, 0, len(req.ObjectKeys))
	total := len(req.ObjectKeys)
	success := 0
	failed := 0

	bucket, err := fs.defaultBucket()
	if err != nil {
		return nil, err
	}

	for _, objectKey := range req.ObjectKeys {
		result := types.DeleteFileResult{
			ObjectKey: objectKey,
			Success:   false,
		}

		// 验证对象键是否属于当前用户
		if !strings.HasPrefix(objectKey, user+"/") {
			result.Error = "access denied: object does not belong to user"
			results = append(results, result)
			failed++

			continue
		}

		// 删除对象
		err := fs.s3Client.RemoveObject(ctx, bucket, objectKey, minio.RemoveObjectOptions{})
		if err != nil {
			result.Error = err.Error()
			results = append(results, result)
			failed++
		} else {
			result.Success = true
			results = append(results, result)
			success++
		}
	}

	return &types.DeleteFilesResponse{
		Results: results,
		Total:   total,
		Success: success,
		Failed:  failed,
	}, nil
}

// UpdateFilesMetadata 更新文件元数据（支持单个/批量）.
func (fs *FileService) UpdateFilesMetadata(ctx context.Context, user string,
	req *types.UpdateFilesMetadataRequest) (*types.UpdateFilesMetadataResponse, error) {
	results := make([]types.UpdateFileMetadataResult, 0, len(req.Items))
	total := len(req.Items)
	success := 0
	failed := 0

	bucket, err := fs.defaultBucket()
	if err != nil {
		return nil, err
	}

	for _, item := range req.Items {
		result := types.UpdateFileMetadataResult{
			ObjectKey: item.ObjectKey,
			Success:   false,
		}

		// 验证对象键是否属于当前用户
		if !strings.HasPrefix(item.ObjectKey, user+"/") {
			result.Error = "access denied: object does not belong to user"
			results = append(results, result)
			failed++

			continue
		}

		// 准备复制选项
		copyOpts := minio.CopyDestOptions{
			Bucket:          bucket,
			Object:          item.ObjectKey,
			ReplaceMetadata: true,
		}

		// 设置元数据
		if len(item.Tags) > 0 {
			copyOpts.UserMetadata = make(map[string]string)
			for k, v := range item.Tags {
				copyOpts.UserMetadata[k] = v
			}
		}

		if item.ContentType != "" {
			copyOpts.ContentType = item.ContentType
		}

		// 执行复制操作来更新元数据
		srcOpts := minio.CopySrcOptions{
			Bucket: bucket,
			Object: item.ObjectKey,
		}

		_, err = fs.s3Client.CopyObject(ctx, copyOpts, srcOpts)
		if err != nil {
			result.Error = err.Error()
			results = append(results, result)
			failed++
		} else {
			result.Success = true
			results = append(results, result)
			success++
		}
	}

	return &types.UpdateFilesMetadataResponse{
		Results: results,
		Total:   total,
		Success: success,
		Failed:  failed,
	}, nil
}

// CopyFiles 复制文件（支持单个/批量）.
func (fs *FileService) CopyFiles(ctx context.Context, user string, req *types.CopyFilesRequest) (*types.CopyFilesResponse, error) {
	results := make([]types.CopyFileResult, 0, len(req.Items))
	total := len(req.Items)
	success := 0
	failed := 0

	bucket, err := fs.defaultBucket()
	if err != nil {
		return nil, err
	}

	for _, item := range req.Items {
		result := types.CopyFileResult{
			SourceKey:      item.SourceKey,
			DestinationKey: item.DestinationKey,
			Success:        false,
		}

		// 验证源对象键是否属于当前用户
		if !strings.HasPrefix(item.SourceKey, user+"/") {
			result.Error = "access denied: source object does not belong to user"
			results = append(results, result)
			failed++

			continue
		}

		// 验证目标对象键是否属于当前用户
		if !strings.HasPrefix(item.DestinationKey, user+"/") {
			result.Error = "access denied: destination object does not belong to user"
			results = append(results, result)
			failed++

			continue
		}

		// 执行复制操作
		srcOpts := minio.CopySrcOptions{
			Bucket: bucket,
			Object: item.SourceKey,
		}

		dstOpts := minio.CopyDestOptions{
			Bucket: bucket,
			Object: item.DestinationKey,
		}

		_, err := fs.s3Client.CopyObject(ctx, dstOpts, srcOpts)
		if err != nil {
			result.Error = err.Error()
			results = append(results, result)
			failed++
		} else {
			result.Success = true
			results = append(results, result)
			success++
		}
	}

	return &types.CopyFilesResponse{
		Results: results,
		Total:   total,
		Success: success,
		Failed:  failed,
	}, nil
}

// MoveFiles 移动文件（支持单个/批量）.
func (fs *FileService) MoveFiles(ctx context.Context, user string, req *types.MoveFilesRequest) (*types.MoveFilesResponse, error) {
	results := make([]types.MoveFileResult, 0, len(req.Items))
	total := len(req.Items)
	success := 0
	failed := 0

	bucket, err := fs.defaultBucket()
	if err != nil {
		return nil, err
	}

	for _, item := range req.Items {
		result := types.MoveFileResult{
			SourceKey:      item.SourceKey,
			DestinationKey: item.DestinationKey,
			Success:        false,
		}

		// 验证源对象键是否属于当前用户
		if !strings.HasPrefix(item.SourceKey, user+"/") {
			result.Error = "access denied: source object does not belong to user"
			results = append(results, result)
			failed++

			continue
		}

		// 验证目标对象键是否属于当前用户
		if !strings.HasPrefix(item.DestinationKey, user+"/") {
			result.Error = "access denied: destination object does not belong to user"
			results = append(results, result)
			failed++

			continue
		}

		// 执行复制操作
		srcOpts := minio.CopySrcOptions{
			Bucket: bucket,
			Object: item.SourceKey,
		}

		dstOpts := minio.CopyDestOptions{
			Bucket: bucket,
			Object: item.DestinationKey,
		}

		_, err := fs.s3Client.CopyObject(ctx, dstOpts, srcOpts)
		if err != nil {
			result.Error = err.Error()
			results = append(results, result)
			failed++

			continue
		}

		// 删除源对象
		err = fs.s3Client.RemoveObject(ctx, bucket, item.SourceKey, minio.RemoveObjectOptions{})
		if err != nil {
			result.Error = fmt.Sprintf("copy succeeded but failed to remove source: %v", err)
			results = append(results, result)
			failed++
		} else {
			result.Success = true
			results = append(results, result)
			success++
		}
	}

	return &types.MoveFilesResponse{
		Results: results,
		Total:   total,
		Success: success,
		Failed:  failed,
	}, nil
}
