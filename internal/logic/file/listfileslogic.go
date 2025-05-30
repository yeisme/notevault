package file

import (
	"context"
	"strings"

	"github.com/yeisme/notevault/internal/svc"
	"github.com/yeisme/notevault/internal/types"
	"github.com/yeisme/notevault/pkg/storage/repository/dao"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListFilesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 列出文件，支持分页、筛选和排序。
func NewListFilesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListFilesLogic {
	return &ListFilesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListFilesLogic) ListFiles(req *types.ListFilesRequest) (resp *types.ListFilesResponse, err error) {
	// 初始化响应
	resp = &types.ListFilesResponse{
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	// 验证分页参数
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 10
	}

	// 初始化GORM Gen查询构建器
	query := dao.Use(l.svcCtx.DB)

	// 构建基础查询条件 - 添加过滤已删除文件的条件
	fileQueryBuilder := query.File.WithContext(l.ctx).Where(query.File.DeletedAt.Eq(0))

	// 应用过滤条件
	if req.UserID != "" {
		fileQueryBuilder = fileQueryBuilder.Where(query.File.UserID.Eq(req.UserID))
	}

	if req.FileName != "" {
		fileQueryBuilder = fileQueryBuilder.Where(query.File.FileName.Like("%" + req.FileName + "%"))
	}

	if req.FileType != "" {
		fileQueryBuilder = fileQueryBuilder.Where(query.File.FileType.Eq(req.FileType))
	}

	// 时间范围过滤
	if req.CreatedAtStart > 0 {
		fileQueryBuilder = fileQueryBuilder.Where(query.File.CreatedAt.Gte(req.CreatedAtStart))
	}

	if req.CreatedAtEnd > 0 {
		fileQueryBuilder = fileQueryBuilder.Where(query.File.CreatedAt.Lte(req.CreatedAtEnd))
	}

	if req.UpdatedAtStart > 0 {
		fileQueryBuilder = fileQueryBuilder.Where(query.File.UpdatedAt.Gte(req.UpdatedAtStart))
	}

	if req.UpdatedAtEnd > 0 {
		fileQueryBuilder = fileQueryBuilder.Where(query.File.UpdatedAt.Lte(req.UpdatedAtEnd))
	}

	// 标签过滤 - 需要关联查询
	// TODO: 支持多个标签的查询
	if req.Tag != "" {
		// 先获取标签ID, 通过 tags 表查询
		tag, err := query.Tag.Where(query.Tag.Name.Eq(req.Tag)).First()
		if err == nil && tag != nil {
			// 关联文件标签表进行查询
			fileQueryBuilder = fileQueryBuilder.LeftJoin(query.FileTag, query.FileTag.FileID.EqCol(query.File.FileID))
			fileQueryBuilder = fileQueryBuilder.Where(query.FileTag.TagID.Eq(tag.TagID))
		} else {
			// 如果找不到该标签，则返回空结果
			return resp, nil
		}
	}

	// 计算总数
	total, err := fileQueryBuilder.Count()
	if err != nil {
		l.Error("count error", logx.Field("error", err))
		return nil, err
	}
	resp.TotalCount = total

	// 设置排序
	order := "DESC"
	if strings.ToLower(req.Order) == "asc" {
		order = "ASC"
	}

	switch strings.ToLower(req.SortBy) {
	case "name":
		if order == "ASC" {
			fileQueryBuilder = fileQueryBuilder.Order(query.File.FileName)
		} else {
			fileQueryBuilder = fileQueryBuilder.Order(query.File.FileName.Desc())
		}
	case "size":
		if order == "ASC" {
			fileQueryBuilder = fileQueryBuilder.Order(query.File.Size)
		} else {
			fileQueryBuilder = fileQueryBuilder.Order(query.File.Size.Desc())
		}
	case "type":
		if order == "ASC" {
			fileQueryBuilder = fileQueryBuilder.Order(query.File.FileType)
		} else {
			fileQueryBuilder = fileQueryBuilder.Order(query.File.FileType.Desc())
		}
	case "date":
		fallthrough
	default:
		if order == "ASC" {
			fileQueryBuilder = fileQueryBuilder.Order(query.File.UpdatedAt)
		} else {
			fileQueryBuilder = fileQueryBuilder.Order(query.File.UpdatedAt.Desc())
		}
	}

	// 应用分页
	offset := (req.Page - 1) * req.PageSize
	fileQueryBuilder = fileQueryBuilder.Offset(offset).Limit(req.PageSize)

	// 执行查询
	files, err := fileQueryBuilder.Find()
	if err != nil {
		l.Error("查询文件列表失败", logx.Field("error", err))
		return nil, err
	}

	// 转换为响应格式
	resp.Files = make([]types.FileMetadata, 0, len(files))
	for _, file := range files {
		// 查询文件关联的标签
		var tags []string
		fileTagRelations, err := query.FileTag.Where(query.FileTag.FileID.Eq(file.FileID)).Find()
		if err == nil && len(fileTagRelations) > 0 {
			for _, relation := range fileTagRelations {
				// 对每个关联查询标签名称
				tag, err := query.Tag.Where(query.Tag.TagID.Eq(relation.TagID)).First()
				if err == nil && tag != nil {
					tags = append(tags, tag.Name)
				}
			}
		}

		metadata := types.FileMetadata{
			FileID:      file.FileID,
			UserID:      file.UserID,
			FileName:    file.FileName,
			FileType:    file.FileType,
			ContentType: file.ContentType,
			Size:        file.Size,
			Path:        file.Path,
			CreatedAt:   file.CreatedAt,
			UpdatedAt:   file.UpdatedAt,
			Version:     int(file.CurrentVersion),
			Tags:        tags,
			Description: file.Description,
		}
		resp.Files = append(resp.Files, metadata)
	}

	l.Infof("User %s retrieved file list successfully", l.ctx.Value("user_id"))

	return resp, nil
}
