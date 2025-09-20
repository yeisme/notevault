package service

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/yeisme/notevault/pkg/internal/model"
	"github.com/yeisme/notevault/pkg/internal/types"
)

// StatsService 提供统计计算（基于 DB 的 Files 表）。
type StatsService struct{ *FileService }

func NewStatsService(c context.Context) *StatsService { return &StatsService{NewFileService(c)} }

const (
	hoursPerDay      = 24
	defaultTrendDays = 14
	maxTrendDays     = 60
	oneMB            = 1 << 20
	tenMB            = 10 << 20
	hundredMB        = 100 << 20
)

// 通用聚合结果行。
type aggRow struct {
	Key string `gorm:"column:k"`
	Cnt int64  `gorm:"column:cnt"`
	Sum int64  `gorm:"column:sum"`
}

// FilesSummary 统计当前用户活跃/回收总量与大小。
func (s *StatsService) FilesSummary(ctx context.Context, user string) (types.StatsFilesSummary, error) {
	if user == "" {
		return types.StatsFilesSummary{}, fmt.Errorf("user required")
	}

	dbx := s.dbClient.GetDB().WithContext(ctx)

	// 使用一次聚合查询计算活跃/回收站的数量与大小，避免重复 SQL 与多次往返
	var agg struct {
		ActiveCount  int64 `gorm:"column:active_count"`
		TrashedCount int64 `gorm:"column:trashed_count"`
		ActiveSize   int64 `gorm:"column:active_size"`
		TrashedSize  int64 `gorm:"column:trashed_size"`
	}

	// SQLite/MySQL 兼容处理：使用 COALESCE 避免 NULL
	selectExpr :=
		"COALESCE(SUM(CASE WHEN deleted_at IS NULL THEN 1 ELSE 0 END),0) AS active_count, " +
			"COALESCE(SUM(CASE WHEN deleted_at IS NOT NULL THEN 1 ELSE 0 END),0) AS trashed_count, " +
			"COALESCE(SUM(CASE WHEN deleted_at IS NULL THEN size ELSE 0 END),0) AS active_size, " +
			"COALESCE(SUM(CASE WHEN deleted_at IS NOT NULL THEN size ELSE 0 END),0) AS trashed_size"

	if err := dbx.Model(&model.Files{}).
		Unscoped(). // 包含软删除数据
		Select(selectExpr).
		Where("user = ?", user).
		Scan(&agg).Error; err != nil {
		return types.StatsFilesSummary{}, err
	}

	return types.StatsFilesSummary{
		TotalFiles:   int(agg.ActiveCount + agg.TrashedCount),
		ActiveFiles:  int(agg.ActiveCount),
		TrashedFiles: int(agg.TrashedCount),
		TotalSize:    agg.ActiveSize + agg.TrashedSize,
		ActiveSize:   agg.ActiveSize,
		TrashedSize:  agg.TrashedSize,
	}, nil
}

// FilesByType 按 content_type 一级类型（如 image、video、application）聚合。
func (s *StatsService) FilesByType(ctx context.Context, user string) ([]types.StatsTypeItem, error) { //nolint:ireturn
	if user == "" {
		return nil, fmt.Errorf("user required")
	}

	dbx := s.dbClient.GetDB().WithContext(ctx)

	rows := []struct {
		CT  string
		Cnt int64
		Sum int64
	}{}
	// SQLite/MySQL 兼容处理：取 content_type 的前缀（到 '/' 之前），为空归类 unknown
	// 使用 GORM 原生表达式写法，尽可能兼容
	err := dbx.Model(&model.Files{}).
		Select("CASE WHEN content_type LIKE '%/%' THEN "+
			"SUBSTR(content_type,1,INSTR(content_type,'/')-1) "+
			"ELSE COALESCE(NULLIF(content_type,''),'unknown') END as ct, "+
			"COUNT(*) as cnt, COALESCE(SUM(size),0) as sum").
		Where("user = ?", user).
		Group("ct").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	out := make([]types.StatsTypeItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, types.StatsTypeItem{Type: r.CT, Count: int(r.Cnt), Size: r.Sum})
	}

	return out, nil
}

// FilesBySizeBuckets 按大小分桶（可根据需要调整分桶）。
func (s *StatsService) FilesBySizeBuckets(ctx context.Context, user string) ([]types.StatsSizeBucket, error) { //nolint:ireturn
	if user == "" {
		return nil, fmt.Errorf("user required")
	}

	dbx := s.dbClient.GetDB().WithContext(ctx)

	buckets := []types.StatsSizeBucket{
		{Name: "0-1MB", Min: 0, Max: oneMB},
		{Name: "1-10MB", Min: oneMB, Max: tenMB},
		{Name: "10-100MB", Min: tenMB, Max: hundredMB},
		{Name: ">=100MB", Min: hundredMB, Max: -1},
	}

	for i := range buckets {
		q := dbx.Model(&model.Files{}).Where("user = ? AND size >= ?", user, buckets[i].Min)
		if buckets[i].Max > 0 {
			q = q.Where("size < ?", buckets[i].Max)
		}

		var (
			cnt int64
			sum struct{ Sum int64 }
		)

		if err := q.Count(&cnt).Error; err != nil {
			return nil, err
		}

		if err := q.Select("COALESCE(SUM(size),0) as sum").Scan(&sum).Error; err != nil {
			return nil, err
		}

		buckets[i].Count = int(cnt)
		buckets[i].Size = sum.Sum
	}

	return buckets, nil
}

// FilesTrend 按天统计数量与大小趋势（最近 N 天）。
func (s *StatsService) FilesTrend(ctx context.Context, user string, days int) ([]types.StatsTrendPoint, error) { //nolint:ireturn
	if user == "" {
		return nil, fmt.Errorf("user required")
	}

	if days <= 0 || days > maxTrendDays {
		days = 30
	}

	dbx := s.dbClient.GetDB().WithContext(ctx)

	start := time.Now().UTC().AddDate(0, 0, -days+1).Truncate(hoursPerDay * time.Hour)
	rows := []struct {
		D   string
		Cnt int64
		Sum int64
	}{}
	// 兼容 SQLite/MySQL：按 DATE(last_modified) 分组
	// SQL: DATE(last_modified) as d, COUNT(*) as cnt, COALESCE(SUM(size),0) as sum WHERE user = ? AND last_modified >= ? GROUP BY DATE(last_modified)
	// ORDER BY d
	// 注意 SQLite 的 DATE() 返回格式为 "YYYY-MM-DD"
	// MySQL 也是类似格式
	// 最终结果按天补齐
	// 注意 last_modified 可能有未来时间的脏数据，暂不处理
	if err := dbx.Model(&model.Files{}).
		Select("DATE(last_modified) as d, COUNT(*) as cnt, COALESCE(SUM(size),0) as sum").
		Where("user = ? AND last_modified >= ?", user, start).
		Group("DATE(last_modified)").
		Order("d").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	// 补齐日期
	data := make(map[string]struct {
		C int64
		S int64
	})
	for _, r := range rows {
		data[r.D] = struct{ C, S int64 }{r.Cnt, r.Sum}
	}

	out := make([]types.StatsTrendPoint, 0, days)
	for i := 0; i < days; i++ {
		d := start.AddDate(0, 0, i).Format("2006-01-02")
		if v, ok := data[d]; ok {
			out = append(out, types.StatsTrendPoint{Date: d, Count: int(v.C), Size: v.S})
		} else {
			out = append(out, types.StatsTrendPoint{Date: d})
		}
	}

	return out, nil
}

// StorageSummary 汇总活跃/回收的对象与大小。
func (s *StatsService) StorageSummary(ctx context.Context, user string) (types.StorageSummary, error) {
	if user == "" {
		return types.StorageSummary{}, fmt.Errorf("user required")
	}

	fs, err := s.FilesSummary(ctx, user)
	if err != nil {
		return types.StorageSummary{}, err
	}

	return types.StorageSummary{ActiveObjects: fs.ActiveFiles, ActiveSize: fs.ActiveSize, TrashObjects: fs.TrashedFiles, TrashSize: fs.TrashedSize}, nil
}

// StorageByBucket 活跃对象按桶聚合。
func (s *StatsService) StorageByBucket(ctx context.Context, user string) ([]types.StorageByBucketItem, error) { //nolint:ireturn
	if user == "" {
		return nil, fmt.Errorf("user required")
	}

	rows, err := s.queryAgg(
		ctx,
		"bucket as k, COUNT(*) as cnt, COALESCE(SUM(size),0) as sum",
		"user = ? AND bucket <> ''",
		"bucket",
		user,
	)
	if err != nil {
		return nil, err
	}

	out := make([]types.StorageByBucketItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, types.StorageByBucketItem{Bucket: r.Key, Count: int(r.Cnt), Size: r.Sum})
	}

	return out, nil
}

// UploadsDaily 最近 N 日的每日上传统计（按 last_modified 来近似）。
func (s *StatsService) UploadsDaily(ctx context.Context, user string, days int) ([]types.StatsTrendPoint, error) {
	// 直接复用 FilesTrend
	return s.FilesTrend(ctx, user, days)
}

// UploadsByUser 留出扩展：当系统支持多用户聚合时可实现（目前仅统计当前 user）。
func (s *StatsService) UploadsByUser(ctx context.Context, user string) ([]types.StatsTypeItem, error) {
	if user == "" {
		return nil, fmt.Errorf("user required")
	}
	// 以 category 作为示例聚合
	rows, err := s.queryAgg(
		ctx,
		"COALESCE(NULLIF(category,''),'unknown') as k, COUNT(*) as cnt, COALESCE(SUM(size),0) as sum",
		"user = ?",
		"k",
		user,
	)
	if err != nil {
		return nil, err
	}

	out := make([]types.StatsTypeItem, 0, len(rows))
	for _, r := range rows {
		out = append(out, types.StatsTypeItem{Type: r.Key, Count: int(r.Cnt), Size: r.Sum})
	}

	return out, nil
}

// SystemPerformance/SystemErrors/SystemUsage 暂以空实现返回 501，便于前端联调。
func (s *StatsService) SystemPerformance(_ context.Context) (any, error) {
	return nil, gorm.ErrNotImplemented
}
func (s *StatsService) SystemErrors(_ context.Context) (any, error) {
	return nil, gorm.ErrNotImplemented
}
func (s *StatsService) SystemUsage(_ context.Context) (any, error) {
	return nil, gorm.ErrNotImplemented
}

// Dashboard/Report 基础占位。
func (s *StatsService) Dashboard(ctx context.Context, user string) (map[string]any, error) {
	fs, err := s.FilesSummary(ctx, user)
	if err != nil {
		return nil, err
	}

	buckets, _ := s.FilesBySizeBuckets(ctx, user)
	typesAgg, _ := s.FilesByType(ctx, user)
	trend, _ := s.FilesTrend(ctx, user, defaultTrendDays)

	return map[string]any{"summary": fs, "size_buckets": buckets, "types": typesAgg, "trend": trend}, nil
}

// queryAgg 针对 Files 表执行通用聚合查询，约定 select 中将分组键命名为别名 k。
func (s *StatsService) queryAgg(ctx context.Context, selectExpr string, whereExpr string, groupExpr string, args ...any) ([]aggRow, error) {
	dbx := s.dbClient.GetDB().WithContext(ctx)

	rows := []aggRow{}
	if err := dbx.Model(&model.Files{}).
		Select(selectExpr).
		Where(whereExpr, args...).
		Group(groupExpr).
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	return rows, nil
}
