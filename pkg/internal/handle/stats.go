package handle

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/yeisme/notevault/pkg/internal/service"
	"github.com/yeisme/notevault/pkg/log"
)

const defaultTrendDays = 14

// doStats 是一个通用封装：
//  1. 统一抽取并校验用户
//  2. 创建 StatsService
//  3. 统一错误处理与 JSON 输出
//
// 回调 fn 中负责具体业务逻辑与返回数据（可返回任意 JSON-able 结构）。
func doStats(c *gin.Context, errLogMsg string, fn func(svc *service.StatsService, user string) (any, error)) {
	l := log.Logger()

	user, err := checkUser(c)
	if user == "" || err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user"})
		return
	}

	svc := service.NewStatsService(c.Request.Context())

	data, e := fn(svc, user)
	if e != nil {
		if errLogMsg == "" {
			errLogMsg = "stats handle failed"
		}

		l.Error().Err(e).Msg(errLogMsg)
		c.JSON(http.StatusInternalServerError, gin.H{"error": e.Error()})

		return
	}

	c.JSON(http.StatusOK, data)
}

// GetFilesStats 汇总文件统计。
//
//	@Summary	文件统计汇总
//	@Tags		统计
//	@Produce	json
//	@Success	200	{object}	map[string]any
//	@Failure	400	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v1/stats/files [get]
func GetFilesStats(c *gin.Context) {
	doStats(c, "files summary failed", func(svc *service.StatsService, user string) (any, error) {
		fs, err := svc.FilesSummary(c.Request.Context(), user)
		if err != nil {
			return nil, err
		}
		// 其余维度失败不影响主结果，保持与原逻辑一致（容错返回部分数据）
		typesAgg, _ := svc.FilesByType(c.Request.Context(), user)
		buckets, _ := svc.FilesBySizeBuckets(c.Request.Context(), user)
		trend, _ := svc.FilesTrend(c.Request.Context(), user, defaultTrendDays)

		return gin.H{"summary": fs, "types": typesAgg, "size_buckets": buckets, "trend": trend}, nil
	})
}

// GetFilesStatsByType 按类型统计。
//
//	@Summary	文件类型统计
//	@Tags		统计
//	@Produce	json
//	@Success	200	{array}		map[string]any
//	@Failure	400	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v1/stats/files/type [get]
func GetFilesStatsByType(c *gin.Context) {
	doStats(c, "files by type failed", func(svc *service.StatsService, user string) (any, error) {
		return svc.FilesByType(c.Request.Context(), user)
	})
}

// GetFilesStatsBySize 文件大小分布。
//
//	@Summary	文件大小分布
//	@Tags		统计
//	@Produce	json
//	@Success	200	{array}		map[string]any
//	@Failure	400	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v1/stats/files/size [get]
func GetFilesStatsBySize(c *gin.Context) {
	doStats(c, "files by size failed", func(svc *service.StatsService, user string) (any, error) {
		return svc.FilesBySizeBuckets(c.Request.Context(), user)
	})
}

// GetFilesTrend 文件数量趋势。
//
//	@Summary	文件数量趋势
//	@Tags		统计
//	@Produce	json
//	@Success	200	{array}		map[string]any
//	@Failure	400	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v1/stats/files/trend [get]
func GetFilesTrend(c *gin.Context) {
	doStats(c, "files trend failed", func(svc *service.StatsService, user string) (any, error) {
		return svc.FilesTrend(c.Request.Context(), user, defaultTrendDays)
	})
}

// StorageStats 存储总体统计。
//
//	@Summary	存储统计汇总
//	@Tags		统计
//	@Produce	json
//	@Success	200	{object}	map[string]any
//	@Failure	400	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v1/stats/storage [get]
func StorageStats(c *gin.Context) {
	doStats(c, "storage summary failed", func(svc *service.StatsService, user string) (any, error) {
		sum, err := svc.StorageSummary(c.Request.Context(), user)
		if err != nil {
			return nil, err
		}

		byBucket, _ := svc.StorageByBucket(c.Request.Context(), user)

		return gin.H{"summary": sum, "by_bucket": byBucket}, nil
	})
}

// StorageByBucket 按存储桶统计。
//
//	@Summary	按桶统计
//	@Tags		统计
//	@Produce	json
//	@Success	200	{array}		map[string]any
//	@Failure	400	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v1/stats/storage/bucket [get]
func StorageByBucket(c *gin.Context) {
	doStats(c, "storage by bucket failed", func(svc *service.StatsService, user string) (any, error) {
		return svc.StorageByBucket(c.Request.Context(), user)
	})
}

// UploadsStats 上传历史统计（支持按年/月/日查询）。
//
//	@Summary	上传历史统计
//	@Tags		统计
//	@Produce	json
//	@Param		year	query		int	false	"年份，格式YYYY"
//	@Param		month	query		int	false	"月份，1-12"
//	@Param		day		query		int	false	"日期，1-31；当提供 day 时需同时提供 year 和 month"
//	@Success	200		{array}		map[string]any
//	@Failure	400		{object}	map[string]string
//	@Failure	500		{object}	map[string]string
//	@Router		/api/v1/stats/uploads [get]
func UploadsStats(c *gin.Context) {
	// 先解析并校验查询参数，确保校验错误以 400 返回
	yearStr := c.Query("year")
	monthStr := c.Query("month")
	dayStr := c.Query("day")

	// 无筛选参数：后续走默认最近 N 天
	hasFilter := yearStr != "" || monthStr != "" || dayStr != ""

	var (
		year, month, day int
		start, end       time.Time
	)

	if hasFilter {
		y, m, d, err := parseAndValidateYMD(yearStr, monthStr, dayStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		year, month, day = y, m, d

		// 计算起止日期（UTC，日粒度，闭区间）
		if year > 0 && month == 0 && day == 0 {
			start = time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC)
			end = time.Date(year, time.December, 31, 0, 0, 0, 0, time.UTC)
		} else if year > 0 && month > 0 && day == 0 {
			start = time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
			nextMonth := start.AddDate(0, 1, 0)
			end = nextMonth.AddDate(0, 0, -1)
		} else {
			start = time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
			end = start
		}
	}

	doStats(c, "uploads stats failed", func(svc *service.StatsService, user string) (any, error) {
		if !hasFilter {
			return svc.UploadsDaily(c.Request.Context(), user, defaultTrendDays)
		}

		return svc.UploadsRange(c.Request.Context(), user, start, end)
	})
}

// UploadsDailyStats 每日上传统计（最近 N 天）。
//
//	@Summary	每日上传统计
//	@Tags		统计
//	@Produce	json
//	@Success	200	{array}		map[string]any
//	@Failure	400	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v1/stats/uploads/daily [get]
func UploadsDailyStats(c *gin.Context) {
	doStats(c, "uploads stats failed", func(svc *service.StatsService, user string) (any, error) {
		return svc.UploadsDaily(c.Request.Context(), user, defaultTrendDays)
	})
}

// UploadsByUser 按用户统计（当前用户视角）。
//
//	@Summary	按用户统计上传
//	@Tags		统计
//	@Produce	json
//	@Success	200	{array}		map[string]any
//	@Failure	400	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v1/stats/uploads/user [get]
func UploadsByUser(c *gin.Context) {
	doStats(c, "uploads by user failed", func(svc *service.StatsService, user string) (any, error) {
		return svc.UploadsByUser(c.Request.Context(), user)
	})
}

// DashboardStats 统计仪表板数据。
//
//	@Summary	统计仪表板
//	@Tags		统计
//	@Produce	json
//	@Success	200	{object}	map[string]any
//	@Failure	400	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v1/stats/dashboard [get]
func DashboardStats(c *gin.Context) {
	doStats(c, "dashboard stats failed", func(svc *service.StatsService, user string) (any, error) {
		return svc.Dashboard(c.Request.Context(), user)
	})
}

// ReportStats 生成统计报告（当前返回与仪表板一致的数据结构，可按需扩展）。
//
//	@Summary	统计报告
//	@Tags		统计
//	@Produce	json
//	@Success	200	{object}	map[string]any
//	@Failure	400	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v1/stats/report [get]
func ReportStats(c *gin.Context) {
	doStats(c, "report stats failed", func(svc *service.StatsService, user string) (any, error) {
		// 先复用 Dashboard 数据，后续可根据 query(format=csv/pdf) 等扩展
		return svc.Dashboard(c.Request.Context(), user)
	})
}
