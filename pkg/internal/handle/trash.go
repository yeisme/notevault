package handle

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/yeisme/notevault/pkg/internal/service"
	"github.com/yeisme/notevault/pkg/internal/types"
	"github.com/yeisme/notevault/pkg/log"
)

// ListTrash 获取回收站列表.
func ListTrash(c *gin.Context) {
	l := log.Logger()

	user, err := checkUser(c)
	if user == "" || err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user"})
		return
	}

	page := 1
	size := 50

	if p := c.Query("page"); p != "" {
		_ = c.BindQuery(&page)
	}

	if s := c.Query("size"); s != "" {
		_ = c.BindQuery(&size)
	}

	svc := service.NewTrashService(c.Request.Context())

	resp, e := svc.List(c.Request.Context(), user, page, size)
	if e != nil {
		l.Error().Err(e).Msg("trash list failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": e.Error()})

		return
	}

	c.JSON(http.StatusOK, resp)
}

// RestoreTrash 单个恢复.
func RestoreTrash(c *gin.Context) {
	singleKeyAction(c, "missing id", func(svc *service.TrashService, user string, keys []string) (int, error) {
		return svc.Restore(c.Request.Context(), user, keys)
	})
}

// DeleteTrash 永久删除单个.
func DeleteTrash(c *gin.Context) {
	singleKeyAction(c, "missing id", func(svc *service.TrashService, user string, keys []string) (int, error) {
		return svc.DeletePermanently(c.Request.Context(), user, keys)
	})
}

// RestoreTrashBatch 批量恢复.
func RestoreTrashBatch(c *gin.Context) {
	batchAction(c, func(svc *service.TrashService, user string, keys []string) (int, error) {
		return svc.Restore(c.Request.Context(), user, keys)
	})
}

// DeleteTrashBatch 批量永久删除.
func DeleteTrashBatch(c *gin.Context) {
	batchAction(c, func(svc *service.TrashService, user string, keys []string) (int, error) {
		return svc.DeletePermanently(c.Request.Context(), user, keys)
	})
}

// EmptyTrash 清空回收站.
func EmptyTrash(c *gin.Context) {
	l := log.Logger()

	user, err := checkUser(c)
	if user == "" || err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user"})
		return
	}

	svc := service.NewTrashService(c.Request.Context())

	n, e := svc.Empty(c.Request.Context(), user)
	if e != nil {
		l.Error().Err(e).Msg("trash empty failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": e.Error()})

		return
	}

	c.JSON(http.StatusOK, types.TrashActionResponse{Affected: n})
}

// AutoCleanTrash 自动清理过期回收站记录.
func AutoCleanTrash(c *gin.Context) {
	l := log.Logger()

	user, err := checkUser(c)
	if user == "" || err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user"})
		return
	}

	var req types.TrashAutoCleanRequest

	_ = c.ShouldBindJSON(&req)

	before, ok := req.ParseBefore()
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "before/days required"})
		return
	}

	svc := service.NewTrashService(c.Request.Context())

	n, e := svc.AutoClean(c.Request.Context(), user, before)
	if e != nil {
		l.Error().Err(e).Msg("trash auto clean failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": e.Error()})

		return
	}

	c.JSON(http.StatusOK, types.TrashActionResponse{Affected: n})
}

// singleKeyAction 抽取公共逻辑：校验用户、获取 path id、调用具体动作。
func singleKeyAction(c *gin.Context, missingIDMsg string, act func(svc *service.TrashService, user string, keys []string) (int, error)) {
	l := log.Logger()

	user, err := checkUser(c)
	if user == "" || err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user"})
		return
	}

	key := c.Param("id")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": missingIDMsg})
		return
	}

	svc := service.NewTrashService(c.Request.Context())

	n, e := act(svc, user, []string{key})
	if e != nil {
		l.Error().Err(e).Msg("trash action failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": e.Error()})

		return
	}

	c.JSON(http.StatusOK, types.TrashActionResponse{Affected: n})
}

// batchAction 抽取公共逻辑：校验用户、解析 body、调用具体动作。
func batchAction(c *gin.Context, act func(svc *service.TrashService, user string, keys []string) (int, error)) {
	l := log.Logger()

	user, err := checkUser(c)
	if user == "" || err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing user"})
		return
	}

	var req types.TrashBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	svc := service.NewTrashService(c.Request.Context())

	n, e := act(svc, user, req.ObjectKeys)
	if e != nil {
		l.Error().Err(e).Msg("trash action failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": e.Error()})

		return
	}

	c.JSON(http.StatusOK, types.TrashActionResponse{Affected: n})
}
