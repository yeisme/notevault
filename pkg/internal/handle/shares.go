package handle

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/yeisme/notevault/pkg/internal/service"
	"github.com/yeisme/notevault/pkg/internal/types"
	"github.com/yeisme/notevault/pkg/log"
)

// CreateShare 创建分享链接.
func CreateShare(c *gin.Context) {
	l := log.Logger()

	var req types.CreateShareRequest
	if err := c.ShouldBind(&req); err != nil {
		l.Warn().Err(err).Msg("invalid request")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		return
	}

	user, err := checkUser(c)
	if user == "" || err != nil {
		l.Warn().Err(err).Msg("missing or invalid user")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user"})

		return
	}

	svc := service.NewShareService(c.Request.Context())

	resp, err := svc.CreateShare(c.Request.Context(), user, &req)
	if err != nil {
		l.Error().Err(err).Msg("create share failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusOK, resp)
}

// ListShares 获取我的分享列表.
func ListShares(c *gin.Context) {
	l := log.Logger()

	user, err := checkUser(c)
	if user == "" || err != nil {
		l.Warn().Err(err).Msg("missing or invalid user")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user"})

		return
	}

	svc := service.NewShareService(c.Request.Context())

	resp, err := svc.ListShares(c.Request.Context(), user)
	if err != nil {
		l.Error().Err(err).Msg("list shares failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusOK, resp)
}

// DeleteShare 删除分享.
func DeleteShare(c *gin.Context) {
	l := log.Logger()

	user, err := checkUser(c)
	if user == "" || err != nil {
		l.Warn().Err(err).Msg("missing or invalid user")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user"})

		return
	}

	shareID := c.Param("shareId")
	if shareID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing shareId"})
		return
	}

	svc := service.NewShareService(c.Request.Context())
	if err := svc.DeleteShare(c.Request.Context(), user, shareID); err != nil {
		l.Error().Err(err).Msg("delete share failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.Status(http.StatusNoContent)
}

// GetShareDetail 获取分享详情.
func GetShareDetail(c *gin.Context) {
	l := log.Logger()

	shareID := c.Param("shareId")
	if shareID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing shareId"})
		return
	}

	svc := service.NewShareService(c.Request.Context())

	resp, err := svc.GetShareDetail(c.Request.Context(), shareID)
	if err != nil {
		l.Error().Err(err).Msg("get share detail failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusOK, resp)
}

// AccessShare 访问分享内容（可选密码）.
func AccessShare(c *gin.Context) {
	l := log.Logger()

	shareID := c.Param("shareId")
	if shareID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing shareId"})
		return
	}

	var req types.AccessShareRequest
	if err := c.ShouldBind(&req); err != nil {
		l.Warn().Err(err).Msg("invalid request")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		return
	}

	svc := service.NewShareService(c.Request.Context())

	resp, err := svc.AccessShare(c.Request.Context(), shareID, req.Password)
	if err != nil {
		l.Error().Err(err).Msg("access share failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusOK, resp)
}

// DownloadShare 下载分享文件（返回直链或重定向）.
func DownloadShare(c *gin.Context) {
	l := log.Logger()

	shareID := c.Param("shareId")
	if shareID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing shareId"})
		return
	}

	svc := service.NewShareService(c.Request.Context())

	url, err := svc.GetShareDownloadURL(c.Request.Context(), shareID)
	if err != nil {
		l.Error().Err(err).Msg("get share download url failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusOK, gin.H{"download_url": url})
}

// GetSharePermissions 获取分享权限.
func GetSharePermissions(c *gin.Context) {
	l := log.Logger()

	user, err := checkUser(c)
	if user == "" || err != nil {
		l.Warn().Err(err).Msg("missing or invalid user")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user"})

		return
	}

	shareID := c.Param("shareId")
	if shareID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing shareId"})
		return
	}

	svc := service.NewShareService(c.Request.Context())

	resp, err := svc.GetSharePermissions(c.Request.Context(), user, shareID)
	if err != nil {
		l.Error().Err(err).Msg("get share permissions failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.JSON(http.StatusOK, resp)
}

// UpdateSharePermissions 更新分享权限.
func UpdateSharePermissions(c *gin.Context) {
	l := log.Logger()

	user, err := checkUser(c)
	if user == "" || err != nil {
		l.Warn().Err(err).Msg("missing or invalid user")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user"})

		return
	}

	shareID := c.Param("shareId")
	if shareID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing shareId"})
		return
	}

	var req types.UpdateSharePermissionsRequest
	if err := c.ShouldBind(&req); err != nil {
		l.Warn().Err(err).Msg("invalid request")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		return
	}

	svc := service.NewShareService(c.Request.Context())
	if err := svc.UpdateSharePermissions(c.Request.Context(), user, shareID, &req); err != nil {
		l.Error().Err(err).Msg("update share permissions failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.Status(http.StatusNoContent)
}

// AddShareUser 添加分享用户.
func AddShareUser(c *gin.Context) {
	l := log.Logger()

	user, err := checkUser(c)
	if user == "" || err != nil {
		l.Warn().Err(err).Msg("missing or invalid user")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user"})

		return
	}

	shareID := c.Param("shareId")
	if shareID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing shareId"})
		return
	}

	var req types.AddShareUserRequest
	if err := c.ShouldBind(&req); err != nil || req.UserID == "" {
		if err == nil {
			err = gin.Error{Err: http.ErrNoCookie} // 占位错误，表示参数缺失
		}

		l.Warn().Err(err).Msg("invalid request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid userId"})

		return
	}

	svc := service.NewShareService(c.Request.Context())
	if err := svc.AddShareUser(c.Request.Context(), user, shareID, req.UserID); err != nil {
		l.Error().Err(err).Msg("add share user failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.Status(http.StatusNoContent)
}

// RemoveShareUser 移除分享用户.
func RemoveShareUser(c *gin.Context) {
	l := log.Logger()

	user, err := checkUser(c)
	if user == "" || err != nil {
		l.Warn().Err(err).Msg("missing or invalid user")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user"})

		return
	}

	shareID := c.Param("shareId")
	if shareID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing shareId"})
		return
	}

	targetUser := c.Param("userId")
	if targetUser == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing userId"})
		return
	}

	svc := service.NewShareService(c.Request.Context())
	if err := svc.RemoveShareUser(c.Request.Context(), user, shareID, targetUser); err != nil {
		l.Error().Err(err).Msg("remove share user failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

		return
	}

	c.Status(http.StatusNoContent)
}
