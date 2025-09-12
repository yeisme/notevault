package service

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"maps"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"

	"github.com/yeisme/notevault/pkg/internal/storage/s3"
	"github.com/yeisme/notevault/pkg/internal/types"
	nlog "github.com/yeisme/notevault/pkg/log"
)

const (
	// DefaultSliceCapacity 默认slice预分配容量.
	DefaultSliceCapacity = 100
	// DefaultPresignedOpTimeout 默认预签名操作超时时间.
	DefaultPresignedOpTimeout = 15 * time.Minute
)

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

// defaultBucket 获取默认 bucket.
func (fs *FileService) defaultBucket() (string, error) {
	cfg := fs.s3Client.GetConfig()
	if len(cfg.Buckets) == 0 {
		return "", fmt.Errorf("no bucket configured")
	}

	return cfg.Buckets[0], nil
}

// buildUploadResponse 构建上传响应，处理元数据设置.
func (fs *FileService) buildUploadResponse(objectKey, hash, actualFileName string,
	size int64, uploadInfo minio.UploadInfo, meta *types.UploadFileMetadata) types.UploadFileResponse {
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

	// 如果提供了自定义的last_modified，使用它而不是服务器时间
	if meta != nil && meta.LastModified != "" {
		if parsedTime, err := time.Parse(time.RFC3339, meta.LastModified); err == nil {
			response.LastModified = parsedTime.Format(time.RFC3339)
		}
	}

	// 添加元数据信息
	if meta != nil {
		response.Tags = meta.Tags
		response.Description = meta.Description
		response.ContentType = meta.ContentType
	}

	return response
}

// uploadFile 上传文件.
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

	// 设置自定义最后修改时间
	if metadata != nil && metadata.LastModified != "" {
		if parsedTime, err := time.Parse(time.RFC3339, metadata.LastModified); err == nil {
			// 注意：MinIO可能不支持直接设置LastModified，这里我们通过UserMetadata传递
			if opts.UserMetadata == nil {
				opts.UserMetadata = make(map[string]string)
			}

			opts.UserMetadata["last-modified"] = parsedTime.Format(time.RFC3339)
		}
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

// findFolderPath 根据folderID查找文件夹路径和名称.
// 注意：这是一个简化的实现，实际项目中应该从数据库查询.
func findFolderPath(ctx context.Context, s3Client *s3.Client, bucket, user, folderID string) (path, name string, err error) {
	// 扫描用户的所有文件夹对象，查找匹配的folderID
	prefix := user + "/"

	// 列出所有对象
	opts := minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	}

	objectCh := s3Client.ListObjects(ctx, bucket, opts)

	for object := range objectCh {
		if object.Err != nil {
			return "", "", fmt.Errorf("list objects error: %v", object.Err)
		}

		// 检查是否是文件夹标记对象（以/结尾）
		if strings.HasSuffix(object.Key, "/") {
			// 移除用户前缀和末尾的/
			folderPath := strings.TrimPrefix(strings.TrimSuffix(object.Key, "/"), user+"/")

			// 计算folderID
			expectedID := fmt.Sprintf("%x", md5.Sum([]byte(user+"/"+folderPath)))

			if expectedID == folderID {
				// 找到匹配的文件夹
				if strings.Contains(folderPath, "/") {
					parts := strings.Split(folderPath, "/")
					return strings.Join(parts[:len(parts)-1], "/"), parts[len(parts)-1], nil
				} else {
					return "", folderPath, nil
				}
			}
		}
	}

	return "", "", fmt.Errorf("folder not found")
}

// renameFolderObjects 重命名文件夹及其内容.
func renameFolderObjects(ctx context.Context, s3Client *s3.Client, bucket, user, oldPath, newPath string) error {
	oldPrefix := user + "/" + oldPath + "/"
	newPrefix := user + "/" + newPath + "/"

	// 如果是根级文件夹，需要特殊处理
	if oldPath == "" {
		oldPrefix = user + "/"
	}

	if newPath == "" {
		newPrefix = user + "/"
	}

	// 收集需要重命名的对象
	objectsToRename := make([]string, 0, DefaultSliceCapacity)

	opts := minio.ListObjectsOptions{
		Prefix:    oldPrefix,
		Recursive: true,
	}

	objectCh := s3Client.ListObjects(ctx, bucket, opts)

	for object := range objectCh {
		if object.Err != nil {
			return fmt.Errorf("list objects error: %v", object.Err)
		}

		objectsToRename = append(objectsToRename, object.Key)
	}

	// 重命名每个对象
	for _, oldKey := range objectsToRename {
		// 生成新键
		newKey := strings.Replace(oldKey, oldPrefix, newPrefix, 1)

		// 复制对象到新位置
		src := minio.CopySrcOptions{
			Bucket: bucket,
			Object: oldKey,
		}
		dst := minio.CopyDestOptions{
			Bucket: bucket,
			Object: newKey,
		}

		_, err := s3Client.CopyObject(ctx, dst, src)
		if err != nil {
			return fmt.Errorf("copy object %s to %s: %v", oldKey, newKey, err)
		}

		// 删除旧对象
		err = s3Client.RemoveObject(ctx, bucket, oldKey, minio.RemoveObjectOptions{})
		if err != nil {
			nlog.Logger().Warn().Err(err).Str("object", oldKey).Msg("failed to remove old object after copy")
			// 不返回错误，继续处理其他对象
		}
	}

	return nil
}

// deleteFolderObjects 删除文件夹及其内容.
func deleteFolderObjects(ctx context.Context, s3Client *s3.Client, bucket, folderPrefix string, recursive bool) (int, error) {
	var (
		deletedCount    int
		objectsToDelete = make([]minio.ObjectInfo, 0, DefaultSliceCapacity)
	)

	// 收集需要删除的对象
	opts := minio.ListObjectsOptions{
		Prefix:    folderPrefix,
		Recursive: recursive,
	}

	objectCh := s3Client.ListObjects(ctx, bucket, opts)

	for object := range objectCh {
		if object.Err != nil {
			return deletedCount, fmt.Errorf("list objects error: %v", object.Err)
		}

		objectsToDelete = append(objectsToDelete, object)
	}

	// 检查是否是空文件夹（如果不是递归删除）
	if !recursive && len(objectsToDelete) > 1 {
		// 如果有多个对象（除了文件夹标记本身），说明文件夹不为空
		return 0, fmt.Errorf("folder is not empty, use recursive=true to delete all contents")
	}

	// 删除对象
	for _, object := range objectsToDelete {
		err := s3Client.RemoveObject(ctx, bucket, object.Key, minio.RemoveObjectOptions{})
		if err != nil {
			nlog.Logger().Warn().Err(err).Str("object", object.Key).Msg("failed to delete object")
			// 继续删除其他对象，不中断整个操作
			continue
		}

		deletedCount++
	}

	return deletedCount, nil
}
