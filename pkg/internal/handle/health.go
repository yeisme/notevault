// Package handle 新增健康检查处理器实现.
package handle

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	ctxPkg "github.com/yeisme/notevault/pkg/context"
)

const timeout = 2 * time.Second

// HealthDB 数据库健康检查.
//
//	@Summary		数据库健康检查
//	@Description	检查数据库连接是否正常
//	@Tags			健康检查
//	@Produce		json
//	@Success		200	{object}	map[string]string	"数据库连接正常"
//	@Failure		503	{object}	map[string]string	"数据库连接异常"
//	@Router			/health/db [get].
func HealthDB(c *gin.Context) {
	dbc := ctxPkg.GetDBClient(c.Request.Context())
	if dbc == nil || dbc.DB == nil { // dbc.DB 来自于嵌入的 *gorm.DB
		c.JSON(http.StatusServiceUnavailable, gin.H{"component": "db", "status": "unhealthy", "error": "db client not initialized"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()

	sqlDB, err := dbc.DB.DB()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"component": "db", "status": "unhealthy", "error": err.Error()})
		return
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"component": "db", "status": "unhealthy", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"component": "db", "status": "ok"})
}

// HealthS3 S3/对象存储健康检查.
//
//	@Summary		S3健康检查
//	@Description	检查S3/对象存储连接是否正常
//	@Tags			健康检查
//	@Produce		json
//	@Success		200	{object}	map[string]string	"S3连接正常"
//	@Failure		503	{object}	map[string]string	"S3连接异常"
//	@Router			/health/s3 [get].
func HealthS3(c *gin.Context) {
	s3c := ctxPkg.GetS3Client(c.Request.Context())
	if s3c == nil || s3c.Client == nil { // s3c.Client 为底层 *minio.Client
		c.JSON(http.StatusServiceUnavailable, gin.H{"component": "s3", "status": "unhealthy", "error": "s3 client not initialized"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()

	if err := s3c.HealthCheck(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"component": "s3", "status": "unhealthy", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"component": "s3", "status": "ok"})
}

// HealthMQ 消息队列健康检查.
//
//	@Summary		消息队列健康检查
//	@Description	检查消息队列连接是否正常
//	@Tags			健康检查
//	@Produce		json
//	@Success		200	{object}	map[string]string	"消息队列连接正常"
//	@Failure		503	{object}	map[string]string	"消息队列连接异常"
//	@Router			/health/mq [get]
func HealthMQ(c *gin.Context) {
	mqc := ctxPkg.GetMQClient(c.Request.Context())
	if mqc == nil { // publisher 与 subscriber 初始化在 New 中, 判空即可
		c.JSON(http.StatusServiceUnavailable, gin.H{"component": "mq", "status": "unhealthy", "error": "mq client not initialized"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"component": "mq", "status": "ok"})
}
