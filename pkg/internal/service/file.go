package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/yeisme/notevault/pkg/configs"
	ctxPkg "github.com/yeisme/notevault/pkg/context"
	"github.com/yeisme/notevault/pkg/internal/storage/db"
	"github.com/yeisme/notevault/pkg/internal/storage/mq"
	"github.com/yeisme/notevault/pkg/internal/storage/s3"
	"github.com/yeisme/notevault/pkg/internal/types"
	nlog "github.com/yeisme/notevault/pkg/log"
)

const (
	// DefaultPresignedOpTimeout 预签名操作超时时间.
	DefaultPresignedOpTimeout = 15 * time.Minute
	// ExtLimit 文件扩展名长度限制，超过该长度将被截断.
	ExtLimit = 20
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

// PresignedPutURL 生成预签名 PUT URL，用于客户端直接上传.
func (fs *FileService) PresignedPutURL(ctx context.Context, user string, req *types.UploadFileRequest) (*types.PresignedUploadResult, error) {
	// 验证文件大小
	const maxFileSize = 10 * 1024 * 1024 // 10MB
	if req.FileSize > maxFileSize {
		return nil, fmt.Errorf("file size %d exceeds limit %d", req.FileSize, maxFileSize)
	}

	cfg := configs.GetConfig().S3
	if len(cfg.Buckets) == 0 {
		return nil, fmt.Errorf("no bucket configured")
	}

	bucket := cfg.Buckets[0]

	// 构建对象键
	objectKey := buildObjectKey(user, req)

	// 生成预签名 URL，过期时间为 DefaultPresignedOpTimeout
	u, err := fs.s3Client.PresignedPutObject(ctx, bucket, objectKey, DefaultPresignedOpTimeout)
	if err != nil {
		return nil, fmt.Errorf("presign put object: %w", err)
	}

	return &types.PresignedUploadResult{
		ObjectKey: objectKey,
		PutURL:    u.String(),
		ExpiresIn: int(DefaultPresignedOpTimeout.Seconds()),
	}, nil
}

// buildObjectKey 构建对象存储路径.放在 service 层便于未来统一策略（如目录分桶、版本号等）.
func buildObjectKey(user string, req *types.UploadFileRequest) string {
	fileType := req.FileType
	fileName := sanitizeFileName(req.FileName)

	id := uuid.New().String()
	datePath := time.Now().UTC().Format("2006/01") // 只到月，避免目录过深

	return fmt.Sprintf("%s/%s/%s_%s.%s", user, datePath, id, fileName, fileType) // user/2023/10/uuid_filename.ext
}

// sanitizeFileName 清理文件名，移除或替换特殊字符以确保对象键安全.
func sanitizeFileName(fileName string) string {
	// 简单清理：移除或替换常见特殊字符
	// 这里可以扩展为更复杂的清理逻辑
	replacer := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_")
	return replacer.Replace(fileName)
}
