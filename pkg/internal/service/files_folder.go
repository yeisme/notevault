package service

import (
	"context"
	"crypto/md5"
	"fmt"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"

	"github.com/yeisme/notevault/pkg/internal/model"
	"github.com/yeisme/notevault/pkg/internal/types"
)

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

	// 执行删除操作（S3）
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

	// 同步到数据库：软删除该前缀下的记录（不阻断主流程）
	dbx := fs.dbClient.GetDB().WithContext(ctx)
	_ = dbx.Where("user = ? AND object_key LIKE ?", user, folderPrefix+"%").Delete(&model.Files{}).Error

	return &types.DeleteFolderResponse{
		FolderID:     folderID,
		Name:         folderName,
		Path:         folderPath,
		DeletedFiles: deletedFiles,
		Success:      true,
	}, nil
}
