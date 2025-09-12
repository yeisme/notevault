package service

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
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
	response := fs.buildUploadResponse(objectKey, hash, actualFileName, size, uploadInfo, metadata)

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

// CreateFolder 创建文件夹.
func (fs *FileService) CreateFolder(ctx context.Context, user string, req *types.CreateFolderRequest) (*types.CreateFolderResponse, error) {
	bucket, err := fs.defaultBucket()
	if err != nil {
		return nil, err
	}

	// 确定目标用户：优先使用请求中的用户字段，否则使用当前用户
	targetUser := user
	if req.User != "" {
		targetUser = req.User
	}

	// 构建完整的文件夹路径
	fullPath := req.Name
	if req.Path != "" {
		fullPath = req.Path + "/" + req.Name
	}

	// 在 S3 中创建文件夹标记（空对象）
	folderKey := fmt.Sprintf("%s/%s/", targetUser, fullPath)

	_, err = fs.s3Client.PutObject(ctx, bucket, folderKey, nil, 0, minio.PutObjectOptions{})
	if err != nil {
		return &types.CreateFolderResponse{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	// 生成文件夹ID（使用用户和路径的组合hash）
	folderID := fmt.Sprintf("%x", md5.Sum([]byte(targetUser+"/"+fullPath)))

	return &types.CreateFolderResponse{
		FolderID:  folderID,
		Name:      req.Name,
		Path:      req.Path,
		FullPath:  fullPath,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Success:   true,
	}, nil
}

// RenameFolder 重命名文件夹.
func (fs *FileService) RenameFolder(ctx context.Context, user string, folderID string, req *types.RenameFolderRequest) (*types.RenameFolderResponse, error) {
	bucket, err := fs.defaultBucket()
	if err != nil {
		return nil, err
	}

	// 查找文件夹的当前路径
	// 注意：在实际项目中，这应该从数据库中查询，这里通过扫描S3对象来模拟
	oldPath, oldName, err := findFolderPath(ctx, fs.s3Client, bucket, user, folderID)
	if err != nil {
		return &types.RenameFolderResponse{
			FolderID: folderID,
			Success:  false,
			Error:    fmt.Sprintf("failed to find folder: %v", err),
		}, err
	}

	// 构建新的路径
	var newPath string
	if oldPath == "" {
		// 根级文件夹
		newPath = req.NewName
	} else if strings.Contains(oldPath, "/") {
		// 如果有父路径，替换最后一部分
		parts := strings.Split(oldPath, "/")
		parts[len(parts)-1] = req.NewName
		newPath = strings.Join(parts, "/")
	} else {
		// 单级文件夹
		newPath = req.NewName
	}

	// 如果新旧名称相同，返回成功
	if oldName == req.NewName {
		return &types.RenameFolderResponse{
			FolderID:  folderID,
			OldName:   oldName,
			NewName:   req.NewName,
			Path:      oldPath,
			UpdatedAt: time.Now().UTC().Format(time.RFC3339),
			Success:   true,
		}, nil
	}

	// 执行重命名操作
	err = renameFolderObjects(ctx, fs.s3Client, bucket, user, oldPath, newPath)
	if err != nil {
		return &types.RenameFolderResponse{
			FolderID: folderID,
			OldName:  oldName,
			NewName:  req.NewName,
			Path:     oldPath,
			Success:  false,
			Error:    err.Error(),
		}, err
	}

	return &types.RenameFolderResponse{
		FolderID:  folderID,
		OldName:   oldName,
		NewName:   req.NewName,
		Path:      newPath,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Success:   true,
	}, nil
}

// DeleteFolder 删除文件夹.
func (fs *FileService) DeleteFolder(ctx context.Context, user string, folderID string, req *types.DeleteFolderRequest) (*types.DeleteFolderResponse, error) {
	bucket, err := fs.defaultBucket()
	if err != nil {
		return nil, err
	}

	// 查找文件夹的路径和名称
	folderPath, folderName, err := findFolderPath(ctx, fs.s3Client, bucket, user, folderID)
	if err != nil {
		return &types.DeleteFolderResponse{
			FolderID: folderID,
			Success:  false,
			Error:    fmt.Sprintf("failed to find folder: %v", err),
		}, err
	}

	// 构建完整的文件夹前缀
	var folderPrefix string
	if folderPath != "" {
		folderPrefix = user + "/" + folderPath + "/" + folderName + "/"
	} else {
		folderPrefix = user + "/" + folderName + "/"
	}

	// 执行删除操作
	deletedFiles, err := deleteFolderObjects(ctx, fs.s3Client, bucket, folderPrefix, req.Recursive)
	if err != nil {
		return &types.DeleteFolderResponse{
			FolderID: folderID,
			Name:     folderName,
			Path:     folderPath,
			Success:  false,
			Error:    err.Error(),
		}, err
	}

	return &types.DeleteFolderResponse{
		FolderID:     folderID,
		Name:         folderName,
		Path:         folderPath,
		DeletedFiles: deletedFiles,
		Success:      true,
	}, nil
}
