package service

import (
	"context"
	"fmt"
	"time"

	"github.com/yeisme/notevault/pkg/internal/model"
	"github.com/yeisme/notevault/pkg/internal/types"
)

// TrashService 提供回收站相关能力（最小实现，基于 DB 软删除标记）.
type TrashService struct{ *FileService }

func NewTrashService(c context.Context) *TrashService { return &TrashService{NewFileService(c)} }

// List 列出用户回收站中的文件（DeletedAt 非空）.
func (t *TrashService) List(ctx context.Context, user string, page, size int) (types.TrashListResponse, error) { //nolint:ireturn
	if user == "" {
		return types.TrashListResponse{}, fmt.Errorf("user is required")
	}

	if page <= 0 {
		page = 1
	}

	if size <= 0 || size > 200 {
		size = 50
	}

	dbx := t.dbClient.GetDB().WithContext(ctx).Model(&model.Files{}).Unscoped().Where("user = ? AND deleted_at IS NOT NULL", user)

	var total int64
	if err := dbx.Count(&total).Error; err != nil {
		return types.TrashListResponse{}, err
	}

	var rows []model.Files
	if err := dbx.Order("deleted_at DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error; err != nil {
		return types.TrashListResponse{}, err
	}

	files := make([]types.ObjectInfo, 0, len(rows))
	for _, r := range rows {
		files = append(files, types.ObjectInfo{
			ObjectKey:    r.ObjectKey,
			Size:         r.Size,
			ETag:         r.ETag,
			ContentType:  r.ContentType,
			LastModified: r.LastModified.UTC().Format(time.RFC3339),
			VersionID:    r.VersionID,
			StorageClass: r.StorageClass,
			Bucket:       r.Bucket,
		})
	}

	return types.TrashListResponse{Total: int(total), Page: page, Size: size, Files: files}, nil
}

// Restore 恢复指定对象键（DB: 取消软删除）.
func (t *TrashService) Restore(ctx context.Context, user string, keys []string) (int, error) {
	if user == "" || len(keys) == 0 {
		return 0, fmt.Errorf("user/keys required")
	}

	dbx := t.dbClient.GetDB().WithContext(ctx).Model(&model.Files{}).Unscoped()
	// 将 DeletedAt 置空
	tx := dbx.Where("user = ? AND object_key IN ?", user, keys).Update("deleted_at", nil)

	return int(tx.RowsAffected), tx.Error
}

// DeletePermanently 永久删除（DB 硬删记录；S3 物理删除可选在此或后台执行）.
func (t *TrashService) DeletePermanently(ctx context.Context, user string, keys []string) (int, error) {
	if user == "" || len(keys) == 0 {
		return 0, fmt.Errorf("user/keys required")
	}

	dbx := t.dbClient.GetDB().WithContext(ctx)
	tx := dbx.Where("user = ? AND object_key IN ?", user, keys).Delete(&model.Files{})
	// 尽力而为：失效相关分享与缓存
	_ = NewShareService(ctx).InvalidateSharesByObjectKeys(ctx, user, keys)

	return int(tx.RowsAffected), tx.Error
}

// Empty 清空回收站（硬删所有被软删除的记录）.
func (t *TrashService) Empty(ctx context.Context, user string) (int, error) {
	if user == "" {
		return 0, fmt.Errorf("user required")
	}

	dbx := t.dbClient.GetDB().WithContext(ctx)
	// 先查询 keys 以便失效分享
	var rows []model.Files
	if err := dbx.Unscoped().Where("user = ? AND deleted_at IS NOT NULL", user).Find(&rows).Error; err != nil {
		return 0, err
	}

	keys := make([]string, 0, len(rows))
	for _, r := range rows {
		keys = append(keys, r.ObjectKey)
	}

	tx := dbx.Where("user = ? AND object_key IN ?", user, keys).Delete(&model.Files{})
	if len(keys) > 0 {
		_ = NewShareService(ctx).InvalidateSharesByObjectKeys(ctx, user, keys)
	}

	return int(tx.RowsAffected), tx.Error
}

// AutoClean 删除删除时间早于 before 的回收站记录.
func (t *TrashService) AutoClean(ctx context.Context, user string, before time.Time) (int, error) {
	if user == "" {
		return 0, fmt.Errorf("user required")
	}

	if before.IsZero() {
		return 0, fmt.Errorf("before required")
	}

	dbx := t.dbClient.GetDB().WithContext(ctx)

	var rows []model.Files
	if err := dbx.Unscoped().Where("user = ? AND deleted_at < ?", user, before).Find(&rows).Error; err != nil {
		return 0, err
	}

	keys := make([]string, 0, len(rows))
	for _, r := range rows {
		keys = append(keys, r.ObjectKey)
	}

	tx := dbx.Where("user = ? AND object_key IN ?", user, keys).Delete(&model.Files{})
	if len(keys) > 0 {
		_ = NewShareService(ctx).InvalidateSharesByObjectKeys(ctx, user, keys)
	}

	return int(tx.RowsAffected), tx.Error
}
