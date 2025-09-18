package handle

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	ctxPkg "github.com/yeisme/notevault/pkg/context"
	"github.com/yeisme/notevault/pkg/internal/service"
	"github.com/yeisme/notevault/pkg/internal/types"
	"github.com/yeisme/notevault/pkg/log"
)

// 常量：搜索结果缓存 TTL（秒）.
const searchCacheTTL = 60 * time.Second

// SearchFiles 搜索文件，基于查询，有条件的从数据库中筛选文件列表. 对象存储定期（或者通过事件）同步到数据库.
func SearchFiles(c *gin.Context) {
	l := log.Logger()

	user, err := checkUser(c)
	if user == "" || err != nil {
		l.Warn().Err(err).Msg("missing or invalid user")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user"})

		return
	}

	var req types.SearchFilesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		l.Warn().Err(err).Msg("invalid request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})

		return
	}

	// 合理默认值
	if req.Page <= 0 {
		req.Page = 1
	}

	if req.PageSize <= 0 || req.PageSize > 200 {
		req.PageSize = 50
	}

	// POST 无浏览器缓存，这里手动接入 KV 作为缓存层
	kv := ctxPkg.GetKVClient(c.Request.Context())

	// 无 KV 客户端时，直接查询（早返回，降低嵌套）
	if kv == nil {
		svc := service.NewFileService(c.Request.Context())

		res, err := svc.SearchFiles(c.Request.Context(), user, &req)
		if err != nil {
			l.Error().Err(err).Msg("search files failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

			return
		}

		c.JSON(http.StatusOK, res)

		return
	}

	// 有 KV 客户端，尝试读缓存
	bodyBytes, _ := json.Marshal(req)
	h := sha1.Sum(append([]byte(user+"|/files/search|v1"), bodyBytes...))
	cacheKey := "search:files:" + hex.EncodeToString(h[:])

	if b, err := kv.Get(c.Request.Context(), cacheKey); err == nil && len(b) > 0 {
		var cached types.SearchFilesResponse
		if jsonErr := json.Unmarshal(b, &cached); jsonErr == nil {
			c.JSON(http.StatusOK, cached)
			return
		}
	}

	// 未命中缓存，查询 DB 并回填缓存
	svc := service.NewFileService(c.Request.Context())

	res, err := svc.SearchFiles(c.Request.Context(), user, &req)
	if err != nil {
		l.Error().Err(err).Msg("search files failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	if data, mErr := json.Marshal(res); mErr == nil {
		_ = kv.Set(c.Request.Context(), cacheKey, data, searchCacheTTL)
	}

	c.JSON(http.StatusOK, res)
}

// ListFilesThisMonth 列出用户当月的文件，并返回对象信息列表.
//
//	@Summary		列出用户当月的文件
//	@Description	列出用户当月的文件，并返回对象信息列表.支持通过查询参数 year 和 month 指定年份和月份.
//	@Tags			文件查询
//	@Accept			json
//	@Produce		json
//	@Param			year	query		int						false	"年份，格式为YYYY，例如2023"
//	@Param			month	query		int						false	"月份，格式为MM，例如09"
//	@Success		200		{object}	types.ListFilesResponse	"文件列表"
//	@Failure		400		{object}	map[string]string		"请求参数错误"
//	@Failure		500		{object}	map[string]string		"服务器内部错误"
//	@Router			/api/v1/files/list [get]
func ListFilesThisMonth(c *gin.Context) {
	l := log.Logger()

	user, err := checkUser(c)
	if user == "" || err != nil {
		l.Warn().Err(err).Msg("missing or invalid user")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user"})

		return
	}

	// 支持查询参数形式：
	//  year=YYYY&month=MM（均为数字）
	monthStr := c.Query("month")
	yearStr := c.Query("year")

	now := time.Now().UTC()

	year, month, perr := parseYearMonth(yearStr, monthStr, now)
	if perr != nil {
		// parseYearMonth 已保证错误消息友好
		if strings.Contains(perr.Error(), "year") {
			l.Warn().Str("year", yearStr).Msg(perr.Error())
		} else if strings.Contains(perr.Error(), "month format") || strings.Contains(perr.Error(), "month") {
			l.Warn().Str("month", monthStr).Msg(perr.Error())
		} else {
			l.Warn().Err(perr).Msg("invalid date params")
		}

		c.JSON(http.StatusBadRequest, gin.H{"error": perr.Error()})

		return
	}

	svc := service.NewFileService(c.Request.Context())

	items, err := svc.ListFilesByMonth(c.Request.Context(), user, year, month)
	if err != nil {
		l.Error().Err(err).Msg("list files failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	resp := types.ListFilesResponse{
		Month: time.Date(year, month, 1, 0, 0, 0, 0, time.UTC).Format("2006-01"),
		Files: items,
		Total: len(items),
	}

	c.JSON(http.StatusOK, resp)
}

// parseYearMonth 解析查询参数中的年份与月份.
// 支持形式：year=YYYY & month=MM（均为数字）
// 若两个参数均为空，则返回 now 对应的年与月.
func parseYearMonth(yearStr, monthStr string, now time.Time) (int, time.Month, error) {
	y, m := now.Year(), now.Month()

	// 形式：year=YYYY & month=MM，任意一个存在即尝试解析
	if yearStr != "" {
		yi, err := strconv.Atoi(yearStr)
		if err != nil || yi < 1970 || yi > 9999 {
			return 0, 0, fmt.Errorf("invalid year")
		}

		y = yi
	}

	if monthStr != "" {
		mi, err := strconv.Atoi(monthStr)
		if err != nil || mi < 1 || mi > 12 {
			return 0, 0, fmt.Errorf("invalid month")
		}

		m = time.Month(mi)
	}

	return y, m, nil
}
