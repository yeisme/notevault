package handle

import (
	"archive/zip"
	"bytes"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/yeisme/notevault/pkg/internal/service"
	"github.com/yeisme/notevault/pkg/internal/types"
	"github.com/yeisme/notevault/pkg/log"
)

// GetDownloadURL 处理获取文件访问 URL（单个/批量）.
//
//	@Summary		获取文件访问URL
//	@Description	为文件获取预签名的访问URL，支持单个或批量对象
//	@Tags			文件下载
//	@Accept			json
//	@Produce		json
//	@Param			objects	body		types.GetFilesURLRequest	true	"获取文件访问URL请求"
//	@Success		200		{object}	types.GetFilesURLResponse	"预签名访问URL响应"
//	@Failure		400		{object}	map[string]string			"请求参数错误"
//	@Failure		500		{object}	map[string]string			"服务器内部错误"
//	@Router			/api/v1/files/download/url [post]
func GetDownloadURL(c *gin.Context) {
	l := log.Logger()

	var req types.GetFilesURLRequest
	if err := c.ShouldBind(&req); err != nil {
		l.Warn().Err(err).Msg("invalid request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		return
	}

	// 至少需要一个对象
	if len(req.Objects) == 0 {
		l.Warn().Msg("no objects provided in request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "no objects provided"})

		return
	}

	svc := service.NewFileService(c.Request.Context())

	resp, err := svc.PresignedGetURLs(c.Request.Context(), &req)
	if err != nil {
		l.Error().Err(err).Msg("failed to generate presigned get urls")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusOK, resp)
}

// DownloadFiles 直传下载（单文件或批量打包）.
//
//	@Summary		下载文件（直传/打包）
//	@Description	当对象数量为1且未指定archive时，直接返回文件流；否则按zip打包返回，同时会在响应头附带部分元信息.
//	@Tags			文件下载
//	@Accept			json
//	@Produce		application/octet-stream
//	@Param			req	body		types.DownloadFilesRequest	true	"下载请求"
//	@Success		200	{file}		file						"文件流或zip包"
//	@Failure		400	{object}	map[string]string			"请求参数错误"
//	@Failure		500	{object}	map[string]string			"服务器内部错误"
//	@Router			/api/v1/files/download [post]
func DownloadFiles(c *gin.Context) {
	l := log.Logger()

	var req types.DownloadFilesRequest
	if err := c.ShouldBind(&req); err != nil {
		l.Warn().Err(err).Msg("invalid request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		return
	}

	if len(req.Objects) == 0 {
		l.Warn().Msg("no objects provided in request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "no objects provided"})

		return
	}

	user, err := checkUser(c)
	if user == "" || err != nil {
		l.Warn().Err(err).Msg("missing or invalid user")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user"})

		return
	}

	svc := service.NewFileService(c.Request.Context())

	if len(req.Objects) == 1 && !req.Archive {
		if err := serveSingleFile(c, svc, user, req.Objects[0]); err != nil {
			l.Error().Err(err).Msg("serve single file failed")
		}

		return
	}

	if err := serveZip(c, svc, user, req.Objects, req.ArchiveName); err != nil {
		l.Error().Err(err).Msg("serve zip failed")
	}
}

// escapeRFC5987 简单转义文件名中的引号与分号等.
func escapeRFC5987(s string) string {
	replacer := strings.NewReplacer("\\", "_", "\"", "_", ";", "_", "\n", "_", "\r", "_")
	return replacer.Replace(s)
}

// serveSingleFile 直接返回单个文件流.
func serveSingleFile(c *gin.Context, svc *service.FileService,
	user string, item types.DownloadObjectItem) error {
	obj, info, err := svc.OpenObject(c.Request.Context(), user, item.ObjectKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return err
	}

	defer func() { _ = obj.Close() }()

	fileName := resolveFileName(item.FileName, item.ObjectKey)
	contentType := determineContentType(fileName, info.ContentType)
	reader := io.Reader(obj)
	// 若仍未知类型，尝试读取前 512 字节进行嗅探
	if contentType == "application/octet-stream" {
		const sniffLen = 512

		buf := make([]byte, sniffLen)

		n, _ := io.ReadFull(obj, buf)
		if n > 0 {
			contentType = http.DetectContentType(buf[:n])
			reader = io.MultiReader(bytes.NewReader(buf[:n]), obj)
		} else {
			reader = obj
		}
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Length", strconv.FormatInt(info.Size, 10))
	c.Header("ETag", info.ETag)

	if info.LastModified != "" {
		c.Header("Last-Modified", info.LastModified)
	}

	c.Header("Content-Disposition", "attachment; filename=\""+escapeRFC5987(fileName)+"\"")

	_, copyErr := io.Copy(c.Writer, reader)

	return copyErr
}

// serveZip 将多个对象打包为 ZIP 并流式返回.
func serveZip(c *gin.Context, svc *service.FileService,
	user string, items []types.DownloadObjectItem, archiveName string) error { //nolint:ireturn
	c.Header("Content-Type", "application/zip")

	if strings.TrimSpace(archiveName) == "" {
		archiveName = suggestArchiveName(items)
	}

	if !strings.HasSuffix(strings.ToLower(archiveName), ".zip") {
		archiveName += ".zip"
	}

	c.Header("Content-Disposition", "attachment; filename=\""+escapeRFC5987(archiveName)+"\"")

	zw := zip.NewWriter(c.Writer)

	defer func() { _ = zw.Close() }()

	for _, it := range items {
		obj, info, err := svc.OpenObject(c.Request.Context(), user, it.ObjectKey)
		if err != nil {
			// 跳过错误的条目，继续打包其他文件
			continue
		}

		name := resolveFileName(it.FileName, it.ObjectKey)
		fh := &zip.FileHeader{Name: name, Method: zip.Deflate}

		if info.LastModified != "" {
			if t, parseErr := time.Parse(time.RFC3339, info.LastModified); parseErr == nil {
				fh.Modified = t
			}
		}

		w, err := zw.CreateHeader(fh)
		if err != nil {
			_ = obj.Close()
			continue
		}

		_, _ = io.Copy(w, obj)
		_ = obj.Close()
	}

	return nil
}

// resolveFileName 解析下载文件名.
func resolveFileName(given, key string) string {
	if given != "" {
		return strings.TrimPrefix(given, "/")
	}

	base := filepath.Base(key)
	if base == "" || base == "." || base == "/" {
		return "file"
	}

	return base
}

// determineContentType 根据已知信息推断 Content-Type.
func determineContentType(fileName, headerType string) string {
	if headerType != "" {
		return headerType
	}

	if ext := filepath.Ext(fileName); ext != "" {
		if t := mime.TypeByExtension(ext); t != "" {
			return t
		}
	}

	return "application/octet-stream"
}

// suggestArchiveName 基于第一个文件名或对象键生成打包文件名.
func suggestArchiveName(items []types.DownloadObjectItem) string {
	if len(items) == 1 {
		base := resolveFileName(items[0].FileName, items[0].ObjectKey)

		name := strings.TrimSuffix(base, filepath.Ext(base))
		if name == "" {
			return "download"
		}

		return name
	}

	// 多文件时，取公共前缀（若无公共前缀，则固定名）
	bases := make([]string, 0, len(items))

	for _, it := range items {
		bases = append(bases, resolveFileName(it.FileName, it.ObjectKey))
	}

	prefix := commonPrefix(bases)
	if prefix == "" {
		return "download"
	}
	// 避免前缀包含扩展名
	prefix = strings.TrimSuffix(prefix, filepath.Ext(prefix))

	prefix = strings.Trim(prefix, "-_ .")
	if prefix == "" {
		return "download"
	}

	return prefix
}

// commonPrefix 计算字符串切片的公共前缀（按字符）.
func commonPrefix(ss []string) string {
	if len(ss) == 0 {
		return ""
	}

	prefix := ss[0]
	for _, s := range ss[1:] {
		for !strings.HasPrefix(s, prefix) && prefix != "" {
			prefix = prefix[:len(prefix)-1]
		}

		if prefix == "" {
			return ""
		}
	}

	return prefix
}
