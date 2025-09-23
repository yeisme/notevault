// Package middleware 提供中间件功能.
package middleware

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/yeisme/notevault/pkg/scheduler"
)

type schedulerKey struct{}

// SchedulerMiddleware 将scheduler注入到context中.
func SchedulerMiddleware(sched *scheduler.Scheduler) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), schedulerKey{}, sched)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// GetScheduler 从context中获取scheduler.
func GetScheduler(c *gin.Context) *scheduler.Scheduler {
	if sched, ok := c.Request.Context().Value(schedulerKey{}).(*scheduler.Scheduler); ok {
		return sched
	}

	return nil
}
