package handle

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/yeisme/notevault/pkg/internal/service"
	"github.com/yeisme/notevault/pkg/log"
)

const defaultTrendDays = 14

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
	l := log.Logger()

	user, err := checkUser(c)
	if user == "" || err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user"})
		return
	}

	svc := service.NewStatsService(c.Request.Context())

	fs, e := svc.FilesSummary(c.Request.Context(), user)
	if e != nil {
		l.Error().Err(e).Msg("files summary failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": e.Error()})

		return
	}

	typesAgg, _ := svc.FilesByType(c.Request.Context(), user)
	buckets, _ := svc.FilesBySizeBuckets(c.Request.Context(), user)
	trend, _ := svc.FilesTrend(c.Request.Context(), user, defaultTrendDays)
	c.JSON(http.StatusOK, gin.H{"summary": fs, "types": typesAgg, "size_buckets": buckets, "trend": trend})
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
	l := log.Logger()

	user, err := checkUser(c)
	if user == "" || err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user"})
		return
	}

	svc := service.NewStatsService(c.Request.Context())

	data, e := svc.FilesByType(c.Request.Context(), user)
	if e != nil {
		l.Error().Err(e).Msg("files by type failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": e.Error()})

		return
	}

	c.JSON(http.StatusOK, data)
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
	l := log.Logger()

	user, err := checkUser(c)
	if user == "" || err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user"})
		return
	}

	svc := service.NewStatsService(c.Request.Context())

	data, e := svc.FilesBySizeBuckets(c.Request.Context(), user)
	if e != nil {
		l.Error().Err(e).Msg("files by size failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": e.Error()})

		return
	}

	c.JSON(http.StatusOK, data)
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
	l := log.Logger()

	user, err := checkUser(c)
	if user == "" || err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user"})
		return
	}

	svc := service.NewStatsService(c.Request.Context())

	data, e := svc.FilesTrend(c.Request.Context(), user, defaultTrendDays)
	if e != nil {
		l.Error().Err(e).Msg("files trend failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": e.Error()})

		return
	}

	c.JSON(http.StatusOK, data)
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
	l := log.Logger()

	user, err := checkUser(c)
	if user == "" || err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user"})
		return
	}

	svc := service.NewStatsService(c.Request.Context())

	sum, e := svc.StorageSummary(c.Request.Context(), user)
	if e != nil {
		l.Error().Err(e).Msg("storage summary failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": e.Error()})

		return
	}

	byBucket, _ := svc.StorageByBucket(c.Request.Context(), user)
	c.JSON(http.StatusOK, gin.H{"summary": sum, "by_bucket": byBucket})
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
	l := log.Logger()

	user, err := checkUser(c)
	if user == "" || err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user"})
		return
	}

	svc := service.NewStatsService(c.Request.Context())

	data, e := svc.StorageByBucket(c.Request.Context(), user)
	if e != nil {
		l.Error().Err(e).Msg("storage by bucket failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": e.Error()})

		return
	}

	c.JSON(http.StatusOK, data)
}

// UploadsStats 上传历史统计（简化）。
//
//	@Summary	上传历史统计
//	@Tags		统计
//	@Produce	json
//	@Success	200	{array}		map[string]any
//	@Failure	400	{object}	map[string]string
//	@Failure	500	{object}	map[string]string
//	@Router		/api/v1/stats/uploads [get]
func UploadsStats(c *gin.Context) {
	l := log.Logger()

	user, err := checkUser(c)
	if user == "" || err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user"})
		return
	}

	svc := service.NewStatsService(c.Request.Context())

	data, e := svc.UploadsDaily(c.Request.Context(), user, defaultTrendDays)
	if e != nil {
		l.Error().Err(e).Msg("uploads stats failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": e.Error()})

		return
	}

	c.JSON(http.StatusOK, data)
}
