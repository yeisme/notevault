package file

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/h2non/filetype"
	"github.com/minio/minio-go/v7"
	"github.com/yeisme/notevault/internal/svc"
	"github.com/yeisme/notevault/internal/types"
	"github.com/yeisme/notevault/pkg/storage/repository/dao"
	"github.com/yeisme/notevault/pkg/storage/repository/model"
	"gorm.io/gorm"

	"github.com/zeromicro/go-zero/core/logx"
)

type UploadFileLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

// Upload a new file. The actual file is sent as multipart/form-data.
func NewUploadFileLogic(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *UploadFileLogic {

	return &UploadFileLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

// UploadFile uploads file and metadata
func (l *UploadFileLogic) UploadFile(req *types.FileUploadRequest) (resp *types.FileUploadResponse, err error) {

	// TODO: Decoder jwt token to get userId
	// Get user ID from context
	userId, ok := l.ctx.Value("userId").(string)
	if !ok || userId == "" {
		userId = "notevault"
	}

	// Check if this is a new version upload
	isNewVersion := req.FileID != ""
	var existingFile *model.File
	var newVersionNumber int32 = 1

	if isNewVersion {
		// Initialize the query using gorm gen
		query := dao.Use(l.svcCtx.DB)

		// Verify the existing file belongs to the user
		existingFile, err = query.File.Where(
			query.File.FileID.Eq(req.FileID),
			query.File.UserID.Eq(userId),
			query.File.DeletedAt.Eq(0),
		).First()

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("File not found or you don't have permission to upload new version")
			}
			return nil, fmt.Errorf("Failed to query existing file: %w", err)
		}

		newVersionNumber = existingFile.CurrentVersion + 1
		l.Infof("Uploading new version %d for file %s", newVersionNumber, req.FileID)
	}

	// frontend also check file size, but we need to check it again here
	// TODO: use multipart instead of FromFile
	file, fileHeader, err := l.r.FormFile("file")
	if err != nil {
		return nil, fmt.Errorf("Failed to retrieve uploaded file: %w", err)
	}
	defer file.Close()

	// Check file size
	maxUploadSize := 16 * 1024 * 1024 // 16MB

	if fileHeader.Size > int64(maxUploadSize) {
		return nil, fmt.Errorf("File size exceeds limit (maximum %d MB)", maxUploadSize/(1024*1024))
	}

	// File name processing
	fileName := req.FileName
	if fileName == "" {
		fileName = fileHeader.Filename
	}

	// File type processing
	contentType := fileHeader.Header.Get("Content-Type")

	fileType := req.FileType
	if fileType == "" {
		fileHeaderBytes := make([]byte, 261)
		if _, err := file.Read(fileHeaderBytes); err != nil {
			return nil, fmt.Errorf("Failed to read file header: %w", err)
		}
		// Reset file pointer for subsequent operations
		if _, err := file.Seek(0, 0); err != nil {
			return nil, fmt.Errorf("Failed to reset file pointer: %w", err)
		}
		kind, err := filetype.Match(fileHeaderBytes)
		if err != nil {
			return nil, fmt.Errorf("Failed to determine file type: %w", err)
		}
		if kind == filetype.Unknown {
			// If file type cannot be recognized, check file extension
			extension := strings.ToLower(filepath.Ext(fileName))

			// Set correct MIME types for common text file types
			textExtensions := map[string]string{
				".md":   "text/markdown",
				".txt":  "text/plain",
				".csv":  "text/csv",
				".json": "application/json",
				".xml":  "application/xml",
				".html": "text/html",
				".css":  "text/css",
				".js":   "application/javascript",
				".yml":  "application/x-yaml",
				".yaml": "application/x-yaml",
				".toml": "application/toml",
				".ini":  "text/plain",
				".conf": "text/plain",
				".log":  "text/plain",
				".sql":  "application/sql",
			}

			if mimeType, ok := textExtensions[extension]; ok {
				fileType = mimeType
			} else if contentType != "" {
				fileType = contentType
			} else {
				fileType = "application/octet-stream"
			}
		} else {
			fileType = kind.MIME.Value
		}
	}

	// Calculate file hash (sha256) before uploading
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, fmt.Errorf("failed to calculate file hash: %w", err)
	}
	fileID := fmt.Sprintf("%x", hash.Sum(nil))

	// Reset file pointer to beginning for upload
	if _, err := file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to reset file for upload: %w", err)
	}

	// Get the timestamp and yearMonth format
	now := time.Now()
	yearMonth := now.Format("200601")
	now_time := now.Unix()

	// Build final storage path with fileID
	storePath := fmt.Sprintf("%s/%s/%s", userId, yearMonth, fileID)

	// Upload file directly to final location
	_, err = l.svcCtx.OSS.PutObject(
		l.ctx,
		l.svcCtx.Config.Storage.Oss.BucketName,
		storePath,
		file,
		fileHeader.Size,
		minio.PutObjectOptions{ContentType: contentType},
	)
	if err != nil {
		l.Error("Failed to upload file to OSS", logx.Field("error", err))
		return nil, fmt.Errorf("Failed to upload file to storage: %w", err)
	}

	// Initialize the query using gorm gen
	query := dao.Use(l.svcCtx.DB)

	// Check if file with same hash already exists in database using gorm gen
	// Check if files with the same hash already exist (including files from same user and other users)
	var existingFileByHash *model.File
	existingFileByHash, err = query.File.Where(
		query.File.FileID.Eq(fileID),
		query.File.DeletedAt.Eq(0), // Only check non-deleted files
	).First()

	if err == nil && existingFileByHash != nil {
		// File already exists, handle differently based on owner
		if existingFileByHash.UserID == userId {
			// Duplicate file from same user
			l.Info("User attempting to upload duplicate file",
				logx.Field("userId", userId),
				logx.Field("fileID", fileID),
				logx.Field("fileName", fileName),
				logx.Field("existingFileName", existingFileByHash.FileName))

			// Clean up the just uploaded duplicate file
			l.cleanupFile(storePath)

			return nil, fmt.Errorf("you have already uploaded a file with identical content (file name: %s, upload time: %s)",
				existingFileByHash.FileName,
				time.Unix(existingFileByHash.CreatedAt, 0).Format("2006-01-02 15:04:05"))
		} else {
			// Same content file already uploaded by different user
			l.Info("File with same content exists from different user",
				logx.Field("currentUserId", userId),
				logx.Field("existingUserId", existingFileByHash.UserID),
				logx.Field("fileID", fileID),
				logx.Field("fileName", fileName))

			// In this case, we can choose to:
			// 1. Allow upload (different users can have files with same content)
			// 2. Reject upload with notification
			// Here we choose to allow upload but log for monitoring
			l.Info("Allowing upload of duplicate content for different user")
			// 继续执行上传流程，不返回错误
		}
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		// 数据库查询错误（非记录不存在错误）
		l.Error("Failed to check for existing file",
			logx.Field("fileID", fileID),
			logx.Field("userId", userId),
			logx.Field("fileName", fileName),
			logx.Field("error", err))

		// 清理已上传的文件
		l.cleanupFile(storePath)

		return nil, fmt.Errorf("检查文件重复性时发生数据库错误: %w", err)
	}

	// 如果到这里说明：
	// 1. 文件不存在（err == gorm.ErrRecordNotFound）
	// 2. 或者文件存在但属于其他用户且我们允许重复上传
	// 继续执行后续流程

	// Process file tags, splitting by comma
	// TODO: I hope tags auto generate, use some LLM or NLP to generate tags
	var tags []string
	if req.Tags != "" {
		// Split and clean tags
		rawTags := strings.SplitSeq(req.Tags, ",")
		for tag := range rawTags {
			trimmedTag := strings.TrimSpace(tag)
			if trimmedTag != "" {
				tags = append(tags, trimmedTag)
			}
		}
	}

	// Create metadata
	metadata := &types.FileMetadata{
		FileID:      fileID,
		UserID:      userId,
		FileName:    fileName,
		FileType:    fileType,
		ContentType: contentType,
		Size:        fileHeader.Size,
		Path:        storePath,
		CreatedAt:   now_time,
		UpdatedAt:   now_time,
		Version:     1,
		Tags:        tags,
		Description: req.Description,
	}
	l.Infof("File metadata created successfully: %+v", metadata)

	// Save metadata to database using the model directly
	fileModel := model.File{
		FileID:         metadata.FileID,
		UserID:         metadata.UserID,
		FileName:       metadata.FileName,
		FileType:       metadata.FileType,
		ContentType:    metadata.ContentType,
		Size:           metadata.Size,
		Path:           metadata.Path,
		CreatedAt:      metadata.CreatedAt,
		UpdatedAt:      metadata.UpdatedAt,
		CurrentVersion: int32(metadata.Version),
		Description:    metadata.Description,
	}

	// Save file_version to database
	fileVersion := model.FileVersion{
		VersionID:     fmt.Sprintf("%s_%d", fileID, fileModel.CurrentVersion),
		FileID:        fileID,
		VersionNumber: fileModel.CurrentVersion,
		Size:          fileHeader.Size,
		Path:          storePath,
		ContentType:   contentType,
		CreatedAt:     now_time,
		CommitMessage: "Initial upload",
	}

	// Use gorm gen to create the file
	if err := query.File.Create(&fileModel); err != nil {
		l.Error("Failed to save file metadata", logx.Field("error", err))
		// File uploaded but metadata save failed, should delete file from OSS
		l.cleanupFile(storePath)
		return nil, fmt.Errorf("Failed to save file information: %w", err)
	}

	if err := query.FileVersion.Create(&fileVersion); err != nil {
		// 数据库可能已经存在相同文件
		l.Error("Failed to save file version", logx.Field("error", err))
		// 如果版本保存失败，尝试删除之前创建的文件记录
		_, deleteErr := query.File.Where(query.File.FileID.Eq(fileID)).Delete(&fileModel)
		if deleteErr != nil {
			l.Error("Failed to clean up file record after version creation failed",
				logx.Field("fileID", fileID), logx.Field("error", deleteErr))
			// 继续处理，不要因为清理失败而中断
		}
		l.cleanupFile(storePath)
		return nil, fmt.Errorf("Failed to save file version information: %w", err)
	}

	// Save file tags to database
	if len(tags) > 0 {
		// 使用事务来保证所有的标签操作一致性
		err := l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
			// 使用事务的查询构建器
			txQuery := dao.Use(tx)

			// 为每个标签创建记录（如果不存在）并关联到文件
			for _, tagName := range tags {
				// 查找或创建标签
				var tagModel *model.Tag
				tagModel, err := txQuery.Tag.Where(txQuery.Tag.Name.Eq(tagName)).First()

				if err != nil {
					// 标签不存在，创建新标签
					if errors.Is(err, gorm.ErrRecordNotFound) {
						tagID := fmt.Sprintf("tag_%x", sha256.Sum256([]byte(tagName)))[:36]
						tagModel = &model.Tag{
							TagID: tagID,
							Name:  tagName,
						}
						if err := txQuery.Tag.Create(tagModel); err != nil {
							return fmt.Errorf("failed to create tag: %w", err)
						}
					} else {
						return fmt.Errorf("failed to query tag: %w", err)
					}
				}

				// 创建文件和标签的关联
				fileTagRelation := model.FileTag{
					FileID: fileID,
					TagID:  tagModel.TagID,
				}

				if err := txQuery.FileTag.Create(&fileTagRelation); err != nil {
					return fmt.Errorf("failed to create file-tag relation: %w", err)
				}
			}

			return nil
		})

		if err != nil {
			l.Error("Failed to save file tags", logx.Field("error", err))
			// 记录错误但不影响文件上传的整体结果
			// 注意：这里我们选择继续而不是失败整个上传过程
			l.Logger.Error("Failed to save tags but file upload is successful", logx.Field("error", err))
		}
	}

	// Create response
	resp = &types.FileUploadResponse{
		FileID:      fileID,
		FileName:    fileName,
		ContentType: contentType,
		Size:        fileHeader.Size,
		Message:     "File upload successful",
		Version:     1,
	}

	logx.Infof("File uploaded successfully: %s", fileName)

	return resp, nil
}

// Clean up the uploaded file (called when an error occurs)
func (l *UploadFileLogic) cleanupFile(path string) {
	err := l.svcCtx.OSS.RemoveObject(
		l.ctx,
		l.svcCtx.Config.Storage.Oss.BucketName,
		path,
		minio.RemoveObjectOptions{},
	)
	if err != nil {
		l.Error("Failed to clean up OSS file", logx.Field("path", path), logx.Field("error", err))
	}
}
