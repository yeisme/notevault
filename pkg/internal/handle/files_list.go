package handle

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/yeisme/notevault/pkg/internal/service"
	"github.com/yeisme/notevault/pkg/internal/types"
	"github.com/yeisme/notevault/pkg/log"
)

// SearchFiles 搜索文件，基于查询，有条件的从数据库中筛选文件列表. 对象存储定期（或者通过事件）同步到数据库.
func SearchFiles(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "not implemented"})
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
