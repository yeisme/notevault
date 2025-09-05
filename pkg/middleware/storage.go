package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/yeisme/notevault/pkg/context"
	"github.com/yeisme/notevault/pkg/internal/storage"
)

func StorageMiddleware(manager *storage.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.WithStorageManager(c.Request.Context(), manager)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
