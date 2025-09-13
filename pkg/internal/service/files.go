package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"

	ctxPkg "github.com/yeisme/notevault/pkg/context"
	"github.com/yeisme/notevault/pkg/internal/storage/db"
	"github.com/yeisme/notevault/pkg/internal/storage/mq"
	"github.com/yeisme/notevault/pkg/internal/storage/s3"
	"github.com/yeisme/notevault/pkg/internal/types"
	nlog "github.com/yeisme/notevault/pkg/log"
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
