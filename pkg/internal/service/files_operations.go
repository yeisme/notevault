package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"

	"github.com/yeisme/notevault/pkg/internal/model"
	"github.com/yeisme/notevault/pkg/internal/types"
)

// DeleteFiles 删除文件（支持单个/批量）.
func (fs *FileService) DeleteFiles(ctx context.Context, user string, req *types.DeleteFilesRequest) (*types.DeleteFilesResponse, error) {
	results := make([]types.DeleteFileResult, 0, len(req.ObjectKeys))
	total := len(req.ObjectKeys)
	success := 0
	failed := 0
	// 收集成功删除（或移动到回收站）的对象键，用于后续分享失效
	toInvalidateShares := make([]string, 0, len(req.ObjectKeys))

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

		// 删除对象（S3）
		err := fs.s3Client.RemoveObject(ctx, bucket, objectKey, minio.RemoveObjectOptions{})
		if err != nil {
			result.Error = err.Error()
			results = append(results, result)
			failed++
		} else {
			result.Success = true
			results = append(results, result)
			success++

			// 同步到数据库：软删除对应元数据（若存在）.
			_ = fs.dbSoftDeleteFile(ctx, user, objectKey)

			// 发布对象删除事件
			fs.publishObjectDeleted(bucket, objectKey, "", false)

			// 记录待失效分享的对象键
			toInvalidateShares = append(toInvalidateShares, objectKey)
		}
	}

	// 统一处理：对包含被删除对象的分享立即失效（软删分享记录 + 清理缓存）
	if len(toInvalidateShares) > 0 {
		// 尽力而为，失败不影响文件删除结果
		_ = NewShareService(ctx).InvalidateSharesByObjectKeys(ctx, user, toInvalidateShares)
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
		r := types.UpdateFileMetadataResult{ObjectKey: item.ObjectKey}

		if !fs.userOwnsKey(user, item.ObjectKey) {
			r.Error = "access denied: object does not belong to user"
			results = append(results, r)
			failed++

			continue
		}

		if err := fs.s3UpdateMetadata(ctx, bucket, item); err != nil {
			r.Error = err.Error()
			results = append(results, r)
			failed++

			continue
		}

		// S3 更新成功即算本项成功；DB 同步尽力而为，不影响成功统计.
		r.Success = true
		results = append(results, r)
		success++

		_ = fs.dbUpsertMetadata(ctx, user, bucket, item)
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

		ui, err := fs.s3Client.CopyObject(ctx, dstOpts, srcOpts)
		if err != nil {
			result.Error = err.Error()
			results = append(results, result)
			failed++
		} else {
			result.Success = true
			results = append(results, result)
			success++

			// 发布对象移动事件
			fs.publishObjectMoved(bucket, item.SourceKey, item.DestinationKey, "move")

			// 复制生成的新对象可视为一次写入/更新
			fs.publishObjectStored(bucket, item.DestinationKey, ui.VersionID, "", 0, "", lastPathComponent(item.DestinationKey), "move")
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

// userOwnsKey 校验对象键是否归属于用户命名空间.
func (fs *FileService) userOwnsKey(user, key string) bool {
	return strings.HasPrefix(key, user+"/")
}

// s3UpdateMetadata 通过拷贝覆盖的方式更新对象的元数据.
func (fs *FileService) s3UpdateMetadata(ctx context.Context, bucket string, item types.UpdateFileMetadataItem) error {
	copyOpts := minio.CopyDestOptions{
		Bucket:          bucket,
		Object:          item.ObjectKey,
		ReplaceMetadata: true,
	}

	if len(item.Tags) > 0 {
		copyOpts.UserMetadata = make(map[string]string, len(item.Tags))
		for k, v := range item.Tags {
			copyOpts.UserMetadata[k] = v
		}
	}

	if item.ContentType != "" {
		copyOpts.ContentType = item.ContentType
	}

	srcOpts := minio.CopySrcOptions{
		Bucket: bucket,
		Object: item.ObjectKey,
	}

	_, err := fs.s3Client.CopyObject(ctx, copyOpts, srcOpts)

	return err
}

// dbUpsertMetadata 将用户传入的元数据写入数据库（仅覆盖显式提供的字段）.
func (fs *FileService) dbUpsertMetadata(ctx context.Context, user, bucket string, item types.UpdateFileMetadataItem) error {
	dbx := fs.dbClient.GetDB().WithContext(ctx)

	var rec model.Files

	err := dbx.Where("user = ? AND object_key = ?", user, item.ObjectKey).First(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) { //nolint
		rec = model.Files{
			User:      user,
			ObjectKey: item.ObjectKey,
			FileName:  lastPathComponent(item.ObjectKey),
			Bucket:    bucket,
		}

		if item.ContentType != "" {
			rec.ContentType = item.ContentType
		}

		if item.Category != "" {
			rec.Category = item.Category
		}

		if item.Description != "" {
			rec.Description = item.Description
		}

		if len(item.Tags) > 0 {
			if b, mErr := sonic.Marshal(item.Tags); mErr == nil {
				rec.TagsJSON = string(b)
			}
		}

		return dbx.Create(&rec).Error
	}

	if err != nil {
		// 其他 DB 错误
		return err
	}

	updates := map[string]any{}
	if item.ContentType != "" {
		updates["content_type"] = item.ContentType
	}

	if item.Category != "" {
		updates["category"] = item.Category
	}

	if item.Description != "" {
		updates["description"] = item.Description
	}

	if len(item.Tags) > 0 {
		if b, mErr := sonic.Marshal(item.Tags); mErr == nil {
			updates["tags_json"] = string(b)
		}
	}

	if len(updates) == 0 {
		return nil
	}

	return dbx.Model(&model.Files{}).
		Where("user = ? AND object_key = ?", user, item.ObjectKey).
		Updates(updates).Error
}

// dbSoftDeleteFile 软删除数据库中的文件记录（若存在）.
func (fs *FileService) dbSoftDeleteFile(ctx context.Context, user, objectKey string) error {
	dbx := fs.dbClient.GetDB().WithContext(ctx)
	return dbx.Where("user = ? AND object_key = ?", user, objectKey).Delete(&model.Files{}).Error
}
