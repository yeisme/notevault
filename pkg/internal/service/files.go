package service

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"maps"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"

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

// PresignedPostURLsPolicy 生成预签名 POST URLs，用于客户端批量直接上传，使用策略控制.
func (fs *FileService) PresignedPostURLsPolicy(ctx context.Context, user string,
	req *types.UploadFilesRequestPolicy,
) (*types.UploadFilesResponsePolicy, error) {
	bucket, err := fs.defaultBucket()
	if err != nil {
		return nil, err
	}

	var results = make([]types.PresignedUploadItem, 0, len(req.Files))

	for _, file := range req.Files {
		// 构建对象键
		objectKey := buildObjectKey(user, &file)

		// 为每个文件创建新的策略对象，避免条件累积
		policy := minio.NewPostPolicy()
		_ = policy.SetBucket(bucket)
		_ = policy.SetKey(objectKey)
		_ = policy.SetExpires(time.Now().UTC().Add(DefaultPresignedOpTimeout))

		// 应用文件策略
		applyFilePolicies(policy, &file)

		// 生成预签名表单
		url, formData, err := fs.s3Client.PresignedPostPolicy(ctx, policy)
		if err != nil {
			return nil, fmt.Errorf("presign post policy for %s: %w", file.FileName, err)
		}

		results = append(results, types.PresignedUploadItem{
			ObjectKey: objectKey,
			PutURL:    url.String(),
			FormData:  formData,
			ExpiresIn: int(DefaultPresignedOpTimeout.Seconds()),
		})
	}

	return &types.UploadFilesResponsePolicy{
		Results: results,
	}, nil
}

// PresignedPutURLs 生成预签名 PUT URLs，用于客户端批量直接上传，不使用策略.
func (fs *FileService) PresignedPutURLs(ctx context.Context, user string,
	req *types.UploadFilesRequest,
) (*types.UploadFilesResponse, error) {
	bucket, err := fs.defaultBucket()
	if err != nil {
		return nil, err
	}

	var results = make([]types.PresignedPutItem, 0, len(req.Files))

	for _, file := range req.Files {
		// 构建对象键
		objectKey := buildObjectKey(user, &file)

		// 生成预签名 PUT URL
		url, err := fs.s3Client.PresignedPutObject(ctx, bucket, objectKey, DefaultPresignedOpTimeout)
		if err != nil {
			return nil, fmt.Errorf("presign put for %s: %w", file.FileName, err)
		}

		results = append(results, types.PresignedPutItem{
			ObjectKey: objectKey,
			PutURL:    url.String(),
			ExpiresIn: int(DefaultPresignedOpTimeout.Seconds()),
		})
	}

	return &types.UploadFilesResponse{
		Results: results,
	}, nil
}

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

// UploadSingleFile 上传单个小文件.
func (fs *FileService) UploadSingleFile(ctx context.Context, user string,
	fileName string, fileReader io.Reader, size int64, metadata *types.UploadFileMetadata) (*types.UploadFileResponse, error) {
	bucket, err := fs.defaultBucket()
	if err != nil {
		return nil, err
	}

	// 使用提供的文件名或原始文件名
	actualFileName := fileName
	if metadata != nil && metadata.FileName != "" {
		actualFileName = metadata.FileName
	}

	// 构建对象键
	objectKey := buildObjectKey(user, &types.UploadFileItem{FileName: actualFileName})

	// 计算 hash 和上传
	hash, uploadInfo, err := fs.uploadFile(ctx, bucket, objectKey, fileReader, size, metadata)
	if err != nil {
		return &types.UploadFileResponse{
			ObjectKey: objectKey,
			Success:   false,
			Error:     err.Error(),
		}, err
	}

	// 构建响应
	response := &types.UploadFileResponse{
		ObjectKey:    objectKey,
		Hash:         hash,
		Size:         size,
		ETag:         uploadInfo.ETag,
		LastModified: uploadInfo.LastModified.Format(time.RFC3339),
		VersionID:    uploadInfo.VersionID,
		Bucket:       uploadInfo.Bucket,
		Location:     uploadInfo.Location,
		FileName:     actualFileName,
		Success:      true,
	}

	// 添加元数据信息
	if metadata != nil {
		response.Tags = metadata.Tags
		response.Description = metadata.Description
		response.ContentType = metadata.ContentType
	}

	return response, nil
}

// UploadBatchFiles 批量上传小文件.
func (fs *FileService) UploadBatchFiles(ctx context.Context, user string,
	files map[string]io.Reader, sizes map[string]int64, metadata map[string]*types.UploadFileMetadata) (*types.UploadBatchFilesResponse, error) {
	results := make([]types.UploadFileResponse, 0, len(files))
	total := len(files)
	successful := 0
	failed := 0

	bucket, err := fs.defaultBucket()
	if err != nil {
		return nil, err
	}

	for fileName, fileReader := range files {
		size := sizes[fileName]
		meta := metadata[fileName]

		// 使用提供的文件名或原始文件名
		actualFileName := fileName
		if meta != nil && meta.FileName != "" {
			actualFileName = meta.FileName
		}

		objectKey := buildObjectKey(user, &types.UploadFileItem{FileName: actualFileName})

		hash, uploadInfo, err := fs.uploadFile(ctx, bucket, objectKey, fileReader, size, meta)
		if err != nil {
			results = append(results, types.UploadFileResponse{
				ObjectKey: objectKey,
				Success:   false,
				Error:     err.Error(),
			})
			failed++
		} else {
			response := types.UploadFileResponse{
				ObjectKey:    objectKey,
				Hash:         hash,
				Size:         size,
				ETag:         uploadInfo.ETag,
				LastModified: uploadInfo.LastModified.Format(time.RFC3339),
				VersionID:    uploadInfo.VersionID,
				Bucket:       uploadInfo.Bucket,
				Location:     uploadInfo.Location,
				FileName:     actualFileName,
				Success:      true,
			}

			// 添加元数据信息
			if meta != nil {
				response.Tags = meta.Tags
				response.Description = meta.Description
				response.ContentType = meta.ContentType
			}

			results = append(results, response)
			successful++
		}
	}

	return &types.UploadBatchFilesResponse{
		Results:    results,
		Total:      total,
		Successful: successful,
		Failed:     failed,
	}, nil
}

// uploadFile 内部方法：上传文件并计算 hash.
func (fs *FileService) uploadFile(ctx context.Context, bucket,
	objectKey string, fileReader io.Reader, size int64, metadata *types.UploadFileMetadata) (string, minio.UploadInfo, error) {
	// 创建一个 TeeReader 来同时计算 hash 和上传
	hasher := md5.New()
	teeReader := io.TeeReader(fileReader, hasher)

	// 准备上传选项
	opts := minio.PutObjectOptions{}

	// 设置内容类型
	if metadata != nil && metadata.ContentType != "" {
		opts.ContentType = metadata.ContentType
	}

	// 设置用户元数据（标签等）
	if metadata != nil && len(metadata.Tags) > 0 {
		userMeta := make(map[string]string)
		maps.Copy(userMeta, metadata.Tags)

		opts.UserMetadata = userMeta
	}

	// 上传文件
	uploadInfo, err := fs.s3Client.PutObject(ctx, bucket, objectKey, teeReader, size, opts)
	if err != nil {
		return "", minio.UploadInfo{}, fmt.Errorf("upload file to S3: %w", err)
	}

	// 获取 hash
	hash := fmt.Sprintf("%x", hasher.Sum(nil))

	return hash, uploadInfo, nil
}

// defaultBucket 获取默认 bucket.
func (fs *FileService) defaultBucket() (string, error) {
	cfg := fs.s3Client.GetConfig()
	if len(cfg.Buckets) == 0 {
		return "", fmt.Errorf("no bucket configured")
	}

	return cfg.Buckets[0], nil
}

// resolveGetExpiry 解析请求中指定的过期时间（秒），若未指定返回默认值.
func resolveGetExpiry(req *types.GetFilesURLRequest) time.Duration {
	if req != nil && req.ExpirySeconds > 0 {
		return time.Duration(req.ExpirySeconds) * time.Second
	}

	return DefaultPresignedOpTimeout
}

// buildGetReqParams 构造可选响应头参数.
func buildGetReqParams(item *types.GetFileURLItem) url.Values {
	if item == nil {
		return nil
	}

	var params url.Values

	set := func(k, v string) {
		if v == "" {
			return
		}

		if params == nil {
			params = url.Values{}
		}

		params.Set(k, v)
	}

	set("response-content-type", item.ResponseContentType)
	set("response-content-disposition", item.ResponseContentDisposition)
	set("response-cache-control", item.ResponseCacheControl)
	set("response-content-language", item.ResponseContentLanguage)
	set("response-content-encoding", item.ResponseContentEncoding)

	return params
}

// presignGet 为单个对象生成预签名下载 URL.
func (fs *FileService) presignGet(ctx context.Context, bucket string, item *types.GetFileURLItem, expiry time.Duration) (types.PresignedDownloadItem, error) {
	params := buildGetReqParams(item)

	urlObj, err := fs.s3Client.PresignedGetObject(ctx, bucket, item.ObjectKey, expiry, params)
	if err != nil {
		return types.PresignedDownloadItem{}, fmt.Errorf("presign get for %s: %w", item.ObjectKey, err)
	}

	return types.PresignedDownloadItem{
		ObjectKey: item.ObjectKey,
		GetURL:    urlObj.String(),
		ExpiresIn: int(expiry.Seconds()),
	}, nil
}

// buildObjectKey 构建对象存储路径.放在 service 层便于未来统一策略（如目录分桶、版本号等）.
func buildObjectKey(user string, req *types.UploadFileItem) string {
	fileName := req.FileName

	datePath := time.Now().UTC().Format("2006/01") // 只到月，避免目录过深

	return fmt.Sprintf("%s/%s/%s", user, datePath, fileName) // user/2023/10/uuid_filename
}

// applyFilePolicies 应用文件策略到 MinIO PostPolicy.
func applyFilePolicies(policy *minio.PostPolicy, file *types.UploadFileItem) {
	if file.ContentType != "" {
		_ = policy.SetContentType(file.ContentType)
	}

	if file.MaxSize > 0 || file.MinSize > 0 {
		_ = policy.SetContentLengthRange(file.MinSize, file.MaxSize)
	}

	if file.KeyStartsWith != "" {
		_ = policy.SetKeyStartsWith(file.KeyStartsWith)
	}

	if file.ContentDisposition != "" {
		_ = policy.SetContentDisposition(file.ContentDisposition)
	}

	if file.ContentEncoding != "" {
		_ = policy.SetContentEncoding(file.ContentEncoding)
	}

	if len(file.UserMetadata) > 0 {
		for key, value := range file.UserMetadata {
			_ = policy.SetUserMetadata(key, value)
		}
	}
}
