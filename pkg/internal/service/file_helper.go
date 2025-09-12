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

	"github.com/yeisme/notevault/pkg/internal/types"
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
