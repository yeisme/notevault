package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/yeisme/notevault/pkg/configs"
	ctxPkg "github.com/yeisme/notevault/pkg/context"
	"github.com/yeisme/notevault/pkg/internal/model"
	"github.com/yeisme/notevault/pkg/internal/storage/db"
	"github.com/yeisme/notevault/pkg/internal/storage/mq"
	"github.com/yeisme/notevault/pkg/internal/storage/s3"
	"github.com/yeisme/notevault/pkg/internal/types"
	nlog "github.com/yeisme/notevault/pkg/log"
	"github.com/yeisme/notevault/pkg/queue"
)

// FileService 负责文件相关业务逻辑（存储、元数据处理等），不处理 HTTP 细节.
type FileService struct {
	s3Client *s3.Client
	dbClient *db.Client
	mqClient *mq.Client
}

// NewFileService 从 context 获取依赖实例.
func NewFileService(c context.Context) *FileService {
	s3c := ctxPkg.GetS3Client(c)
	dbc := ctxPkg.GetDBClient(c)
	mqc := ctxPkg.GetMQClient(c)

	// 为了安全起见，应该直接 panic 而不是返回 nil，依赖此服务就不需要再检查
	if s3c == nil || s3c.Client == nil || dbc == nil || dbc.DB == nil || mqc == nil {
		nlog.Logger().Fatal().Msg("storage clients not initialized")
	}

	return &FileService{
		s3Client: s3c,
		dbClient: dbc,
		mqClient: mqc,
	}
}

// ListFilesByMonth lists user's files for a given UTC year-month (YYYY, MM).
// It scans objects with prefix "user/YYYY/MM/" and returns their basic info.
func (fs *FileService) ListFilesByMonth(ctx context.Context, user string, year int, month time.Month) ([]types.ObjectInfo, error) { //nolint:ireturn
	if user == "" {
		return nil, fmt.Errorf("user is required")
	}

	bucket, err := fs.defaultBucket()
	if err != nil {
		return nil, err
	}

	// Prefix pattern matches buildObjectKey's datePath = YYYY/MM
	prefix := fmt.Sprintf("%s/%04d/%02d/", user, year, int(month))

	// List objects under the prefix
	opts := minio.ListObjectsOptions{Prefix: prefix, Recursive: true}
	ch := fs.s3Client.ListObjects(ctx, bucket, opts)

	files := make([]types.ObjectInfo, 0, DefaultSliceCapacity)

	for obj := range ch {
		if obj.Err != nil {
			return nil, fmt.Errorf("list objects: %v", obj.Err)
		}
		// skip "folders"
		if strings.HasSuffix(obj.Key, "/") {
			continue
		}

		files = append(files, types.ObjectInfo{
			ObjectKey: obj.Key,
			Size:      obj.Size,
			ETag:      strings.Trim(obj.ETag, "\""),
			// ContentType not available in ListObjects; leave empty to be filled by Stat when needed
			LastModified: obj.LastModified.UTC().Format(time.RFC3339),
			VersionID:    obj.VersionID,
			StorageClass: obj.StorageClass,
			Bucket:       bucket,
			UserMetadata: nil,
		})
	}

	return files, nil
}

// ListFilesThisMonth lists files for the current UTC year-month.
func (fs *FileService) ListFilesThisMonth(ctx context.Context, user string, now time.Time) ([]types.ObjectInfo, error) { //nolint:ireturn
	y, m, _ := now.UTC().Date()
	return fs.ListFilesByMonth(ctx, user, y, m)
}

// SearchFiles 基于数据库进行条件查询，并返回对象展示信息.
func (fs *FileService) SearchFiles(ctx context.Context, user string, req *types.SearchFilesRequest) (types.SearchFilesResponse, error) { //nolint:ireturn
	if user == "" {
		return types.SearchFilesResponse{}, fmt.Errorf("user is required")
	}

	if req == nil {
		return types.SearchFilesResponse{}, fmt.Errorf("request is nil")
	}

	dbx := fs.dbClient.GetDB().WithContext(ctx).Model(&model.Files{})

	// 基础过滤：按用户
	dbx = dbx.Where("user = ?", user)

	// 前缀过滤（对象键）
	if req.Prefix != "" {
		like := req.Prefix + "%"
		dbx = dbx.Where("object_key LIKE ?", like)
	}

	if req.Category != "" {
		dbx = dbx.Where("category = ?", req.Category)
	}

	if req.ContentType != "" {
		dbx = dbx.Where("content_type LIKE ?", req.ContentType+"%")
	}

	if req.MinSize > 0 {
		dbx = dbx.Where("size >= ?", req.MinSize)
	}

	if req.MaxSize > 0 {
		dbx = dbx.Where("size <= ?", req.MaxSize)
	}

	if !req.Start.IsZero() {
		dbx = dbx.Where("last_modified >= ?", req.Start)
	}

	if !req.End.IsZero() {
		dbx = dbx.Where("last_modified <= ?", req.End)
	}

	// 关键字匹配：文件名、描述、标签 JSON
	if strings.TrimSpace(req.Keyword) != "" {
		kw := "%" + strings.TrimSpace(req.Keyword) + "%"
		dbx = dbx.Where("file_name LIKE ? OR description LIKE ? OR tags_json LIKE ?", kw, kw, kw)
	}

	// 统计总数
	var total int64
	if err := dbx.Count(&total).Error; err != nil {
		return types.SearchFilesResponse{}, fmt.Errorf("count: %w", err)
	}

	// 排序
	sortBy := "last_modified"
	if req.SortBy == "size" {
		sortBy = "size"
	}

	order := "DESC"
	if strings.EqualFold(req.SortOrder, "asc") {
		order = "ASC"
	}

	dbx = dbx.Order(sortBy + " " + order)

	// 分页
	page := req.Page
	size := req.PageSize

	if page <= 0 {
		page = 1
	}

	if size <= 0 || size > 200 {
		size = 50
	}

	offset := (page - 1) * size
	dbx = dbx.Offset(offset).Limit(size)

	// 查询记录
	var rows []model.Files
	if err := dbx.Find(&rows).Error; err != nil {
		return types.SearchFilesResponse{}, fmt.Errorf("query: %w", err)
	}

	// 映射为 ObjectInfo
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
			// Tags、描述等在必要时由客户端再取 meta
		})
	}

	return types.SearchFilesResponse{Total: int(total), Page: page, Size: size, Files: files}, nil
}

// SyncObjectsToDB 同步：扫描对象存储并将对象元数据落库.
func (fs *FileService) SyncObjectsToDB(ctx context.Context, user string) error {
	if user == "" {
		return fmt.Errorf("user is required")
	}

	bucket, err := fs.defaultBucket()
	if err != nil {
		return err
	}

	prefix := user + "/"
	ch := fs.s3Client.ListObjects(ctx, bucket,
		minio.ListObjectsOptions{Prefix: prefix, Recursive: true})

	dbx := fs.dbClient.GetDB().WithContext(ctx)
	now := time.Now().UTC()

	for obj := range ch {
		if obj.Err != nil {
			return fmt.Errorf("list objects: %v", obj.Err)
		}

		if strings.HasSuffix(obj.Key, "/") {
			continue
		}

		// upsert by (user, object_key)
		rec := model.Files{
			User:         user,
			ObjectKey:    obj.Key,
			FileName:     lastPathComponent(obj.Key),
			Size:         obj.Size,
			ETag:         strings.Trim(obj.ETag, "\""),
			ContentType:  "", // TODO 通过 Stat 可补充，这里简化
			Category:     "", // 通过 LLM 分类
			Description:  "", // 通过 LLM 生成摘要
			TagsJSON:     "", // 通过 LLM 提取标签
			Bucket:       bucket,
			VersionID:    obj.VersionID,
			StorageClass: obj.StorageClass,
			LastModified: obj.LastModified.UTC(),
			UpdatedAt:    now,
		}

		// 使用 PostgreSQL/SQLite 的 UPSERT 语法；GORM 提供了统一的 OnConflict
		if err := dbx.Clauses(onConflictUserKeyUpdate()).Create(&rec).Error; err != nil {
			nlog.Logger().Warn().Err(err).Str("key", obj.Key).Msg("upsert file failed")
			// 不中断整体同步
		} else {
			// 元数据成功落库后，发布对象已存储事件（nv.object.stored）
			fs.publishObjectStored(
				bucket,
				obj.Key,
				obj.VersionID,
				strings.Trim(obj.ETag, "\""),
				obj.Size,
				rec.ContentType,
				rec.FileName,
				"sync",
			)
		}
	}

	return nil
}

// SyncObjectsToDBByDate 按日期范围（年/月/日）同步对象到数据库.
// 传参约定：
//   - year>0, month==0, day==0 => 同步全年（前缀 user/YYYY/）
//   - year>0, month>0, day==0  => 同步某月（前缀 user/YYYY/MM/）
//   - year>0, month>0, day>0   => 同步某天（前缀 user/YYYY/MM/ 且按 LastModified.Day 过滤）
//   - 其他组合等价于全量（前缀 user/）.
func (fs *FileService) SyncObjectsToDBByDate(ctx context.Context, user string, year, month, day int) error {
	if user == "" {
		return fmt.Errorf("user is required")
	}

	bucket, err := fs.defaultBucket()
	if err != nil {
		return err
	}

	// 构建前缀
	prefix := user + "/"
	if year > 0 && month <= 0 {
		prefix = fmt.Sprintf("%s/%04d/", user, year)
	}

	if year > 0 && month > 0 {
		prefix = fmt.Sprintf("%s/%04d/%02d/", user, year, month)
	}

	ch := fs.s3Client.ListObjects(ctx, bucket,
		minio.ListObjectsOptions{Prefix: prefix, Recursive: true})

	dbx := fs.dbClient.GetDB().WithContext(ctx)
	now := time.Now().UTC()

	for obj := range ch {
		if obj.Err != nil {
			return fmt.Errorf("list objects: %v", obj.Err)
		}

		if strings.HasSuffix(obj.Key, "/") {
			continue
		}

		// 如果指定了 day，并且 month/year 也指定了，则按 LastModified 的天过滤
		if year > 0 && month > 0 && day > 0 {
			ts := obj.LastModified.UTC()

			y, m, d := ts.Date()
			if y != year || int(m) != month || d != day {
				continue
			}
		}

		rec := model.Files{
			User:         user,
			ObjectKey:    obj.Key,
			FileName:     lastPathComponent(obj.Key),
			Size:         obj.Size,
			ETag:         strings.Trim(obj.ETag, "\""),
			ContentType:  "",
			Category:     "",
			Description:  "",
			TagsJSON:     "",
			Bucket:       bucket,
			VersionID:    obj.VersionID,
			StorageClass: obj.StorageClass,
			LastModified: obj.LastModified.UTC(),
			UpdatedAt:    now,
		}

		if err := dbx.Clauses(onConflictUserKeyUpdate()).Create(&rec).Error; err != nil {
			nlog.Logger().Warn().Err(err).Str("key", obj.Key).Msg("upsert file failed")
		} else {
			// 元数据成功落库后，发布对象已存储事件（nv.object.stored）
			fs.publishObjectStored(
				bucket,
				obj.Key,
				obj.VersionID,
				strings.Trim(obj.ETag, "\""),
				obj.Size,
				rec.ContentType,
				rec.FileName,
				"notevault-sync",
			)
		}
	}

	return nil
}

// lastPathComponent 返回 key 的最后一段文件名.
func lastPathComponent(key string) string {
	if key == "" {
		return ""
	}

	idx := strings.LastIndex(key, "/")
	if idx < 0 || idx+1 >= len(key) {
		return key
	}

	return key[idx+1:]
}

// onConflictUserKeyUpdate 生成 GORM OnConflict 子句以支持 upsert.
// 针对唯一索引 idx_user_key(user, object_key)，更新可变字段.
func onConflictUserKeyUpdate() clause.Expression {
	// 仅更新"系统来源"的字段；保留 DB 中由 LLM/人工写入的富元数据（category/description/tags_json）.
	// 对 content_type 做保护：只有当新值非空时才覆盖，避免被空串擦掉原值.
	return clause.OnConflict{
		Columns: []clause.Column{{Name: "user"}, {Name: "object_key"}},
		DoUpdates: clause.Assignments(map[string]any{
			"file_name": gorm.Expr("EXCLUDED.file_name"),
			"size":      gorm.Expr("EXCLUDED.size"),
			"etag":      gorm.Expr("EXCLUDED.etag"),
			// PostgreSQL/SQLite 兼容：只在新值非空时覆盖
			"content_type": gorm.Expr("COALESCE(NULLIF(EXCLUDED.content_type, ''), content_type)"),
			// LLM 富元数据：不在"对象→DB"同步中覆盖
			// "category":     ...,
			// "description":  ...,
			// "tags_json":    ...,
			"bucket":        gorm.Expr("EXCLUDED.bucket"),
			"version_id":    gorm.Expr("EXCLUDED.version_id"),
			"storage_class": gorm.Expr("EXCLUDED.storage_class"),
			"last_modified": gorm.Expr("EXCLUDED.last_modified"),
			"updated_at":    gorm.Expr("EXCLUDED.updated_at"),
		}),
	}
}

// publishObjectStored 发布 nv.object.stored 事件，供同步流程在 upsert 成功后调用.
// 注意：发布失败仅记录告警，不影响主流程.
func (fs *FileService) publishObjectStored(
	bucket, objectKey, versionID, etag string,
	size int64,
	contentType, fileName, source string,
) {
	cfg := configs.GetConfig().Events
	if !cfg.Enabled || !cfg.Object.Stored {
		return
	}

	payload := queue.ObjectStoredPayload{
		Object: queue.ObjectRef{
			Bucket:      bucket,
			ObjectKey:   objectKey,
			VersionID:   versionID,
			ETag:        etag,
			Size:        size,
			ContentType: contentType,
		},
		Source:   source,
		FileName: fileName,
	}

	if err := queue.PublishObjectStored(fs.mqClient, payload); err != nil {
		nlog.Logger().Warn().Err(err).Str("key", objectKey).Msg("publish object stored message failed")
	}

	// 结构化日志：对象写入完成
	nlog.Logger().Info().
		Str("key", objectKey).
		Str("ver", versionID).
		Str("etag", etag).
		Str("src", source).
		Str("file", fileName).
		Str("event", "nv.object.stored").
		Msg("object stored")
}

// publishObjectUpdated 发布 nv.object.updated（通常是新版本创建或内容变更后）.
func (fs *FileService) publishObjectUpdated(
	bucket, objectKey, versionID, etag string,
	size int64,
	contentType, fileName, source string,
) {
	cfg := configs.GetConfig().Events
	if !cfg.Enabled || !cfg.Object.Updated {
		return
	}

	payload := queue.ObjectUpdatedPayload{
		Object: queue.ObjectRef{
			Bucket:      bucket,
			ObjectKey:   objectKey,
			VersionID:   versionID,
			ETag:        etag,
			Size:        size,
			ContentType: contentType,
		},
		PrevVersionID: "", // 可在调用处填充
	}

	if err := queue.PublishObjectUpdated(fs.mqClient, payload); err != nil {
		nlog.Logger().Warn().Err(err).Str("key", objectKey).Msg("publish object updated message failed")
	}

	nlog.Logger().Info().Str("key", objectKey).
		Str("ver", versionID).Str("etag", etag).
		Str("src", source).Str("file", fileName).
		Str("event", "nv.object.updated").
		Msg("object updated")
}

// publishObjectDeleted 发布 nv.object.deleted 事件.
func (fs *FileService) publishObjectDeleted(bucket, objectKey, versionID string, deletedAll bool) {
	cfg := configs.GetConfig().Events
	if !cfg.Enabled || !cfg.Object.Deleted {
		return
	}

	payload := queue.ObjectDeletedPayload{
		Object: queue.ObjectRef{
			Bucket:    bucket,
			ObjectKey: objectKey,
			VersionID: versionID,
		},
		DeletedAll: deletedAll,
	}
	if err := queue.PublishObjectDeleted(fs.mqClient, payload); err != nil {
		nlog.Logger().Warn().Err(err).Str("key", objectKey).Msg("publish object deleted message failed")
	}

	// 结构化日志：对象删除
	nlog.Logger().Info().
		Str("key", objectKey).
		Str("ver", versionID).
		Bool("all", deletedAll).
		Str("event", "nv.object.deleted").
		Msg("object deleted")
}

// publishObjectVersioned 发布 nv.object.versioned 事件.
func (fs *FileService) publishObjectVersioned(bucket, objectKey, versionID, baseVersionID string) {
	cfg := configs.GetConfig().Events
	if !cfg.Enabled || !cfg.Object.Versioned {
		return
	}

	payload := queue.ObjectVersionedPayload{
		Object: queue.ObjectRef{
			Bucket:    bucket,
			ObjectKey: objectKey,
			VersionID: versionID,
		},
		BaseVersionID: baseVersionID,
	}
	if err := queue.PublishObjectVersioned(fs.mqClient, payload); err != nil {
		nlog.Logger().Warn().Err(err).Str("key", objectKey).Msg("publish object versioned message failed")
	}

	// 结构化日志：对象产生新版本
	nlog.Logger().Info().
		Str("key", objectKey).
		Str("ver", versionID).
		Str("base", baseVersionID).
		Str("event", "nv.object.versioned").
		Msg("object versioned")
}

// publishObjectRestored 发布 nv.object.restored 事件.
func (fs *FileService) publishObjectRestored(bucket, objectKey, fromVersion, restoredAs string) {
	cfg := configs.GetConfig().Events
	if !cfg.Enabled || !cfg.Object.Restored {
		return
	}

	payload := queue.ObjectRestoredPayload{
		Object: queue.ObjectRef{
			Bucket:    bucket,
			ObjectKey: objectKey,
			VersionID: restoredAs,
		},
		FromVersion: fromVersion,
		RestoredAs:  restoredAs,
	}
	if err := queue.PublishObjectRestored(fs.mqClient, payload); err != nil {
		nlog.Logger().Warn().Err(err).Str("key", objectKey).Msg("publish object restored message failed")
	}

	// 结构化日志：对象从历史版本恢复
	nlog.Logger().Info().
		Str("key", objectKey).
		Str("from", fromVersion).
		Str("as", restoredAs).
		Str("event", "nv.object.restored").
		Msg("object restored")
}

// publishObjectMoved 发布 nv.object.moved 事件.
func (fs *FileService) publishObjectMoved(bucket, srcKey, dstKey, reason string) {
	cfg := configs.GetConfig().Events
	if !cfg.Enabled || !cfg.Object.Moved {
		return
	}

	payload := queue.ObjectMovedPayload{
		Source:      queue.ObjectRef{Bucket: bucket, ObjectKey: srcKey},
		Destination: queue.ObjectRef{Bucket: bucket, ObjectKey: dstKey},
		Reason:      reason,
	}
	if err := queue.PublishObjectMoved(fs.mqClient, payload); err != nil {
		nlog.Logger().Warn().Err(err).Str("src", srcKey).Str("dst", dstKey).Msg("publish object moved message failed")
	}

	// 结构化日志：对象移动/重命名
	nlog.Logger().Info().
		Str("src", srcKey).
		Str("dst", dstKey).
		Str("reason", reason).
		Str("event", "nv.object.moved").
		Msg("object moved")
}

// publishObjectAccessed 发布 nv.object.accessed 事件.
func (fs *FileService) publishObjectAccessed(bucket, objectKey, method, via, userAgent, remote string) {
	cfg := configs.GetConfig().Events
	if !cfg.Enabled || !cfg.Object.Accessed {
		return
	}

	payload := queue.ObjectAccessedPayload{
		Object:     queue.ObjectRef{Bucket: bucket, ObjectKey: objectKey},
		Method:     method,
		Via:        via,
		UserAgent:  userAgent,
		RemoteAddr: remote,
	}
	if err := queue.PublishObjectAccessed(fs.mqClient, payload); err != nil {
		nlog.Logger().Warn().Err(err).Str("key", objectKey).Msg("publish object accessed message failed")
	}

	// 结构化日志：对象访问
	nlog.Logger().Info().
		Str("key", objectKey).
		Str("method", method).
		Str("via", via).
		Str("ua", userAgent).
		Str("remote", remote).
		Str("event", "nv.object.accessed").
		Msg("object accessed")
}
