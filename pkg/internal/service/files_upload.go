package service

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"

	"github.com/yeisme/notevault/pkg/internal/types"
)

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
	response := fs.buildUploadResponse(objectKey, hash, actualFileName, size, uploadInfo, metadata)

	// 发布对象写入/更新事件
	ctype := ""
	if metadata != nil {
		ctype = metadata.ContentType
	}

	fs.publishObjectStored(uploadInfo.Bucket, objectKey, uploadInfo.VersionID, uploadInfo.ETag, size, ctype, actualFileName, "upload-single")
	fs.publishObjectUpdated(uploadInfo.Bucket, objectKey, uploadInfo.VersionID, uploadInfo.ETag, size, ctype, actualFileName, "upload-single")

	return &response, nil
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
			response := fs.buildUploadResponse(objectKey, hash, actualFileName, size, uploadInfo, meta)
			// 发布对象写入/更新事件
			ctype := ""
			if meta != nil {
				ctype = meta.ContentType
			}

			fs.publishObjectStored(uploadInfo.Bucket, objectKey, uploadInfo.VersionID, uploadInfo.ETag, size, ctype, actualFileName, "upload-batch")
			fs.publishObjectUpdated(uploadInfo.Bucket, objectKey, uploadInfo.VersionID, uploadInfo.ETag, size, ctype, actualFileName, "upload-batch")

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
