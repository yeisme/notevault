// Package router 管理路由配置，用于设置HTTP服务的路由.
package router

import (
	"github.com/gin-gonic/gin"

	"github.com/yeisme/notevault/pkg/internal/handle"
)

// RegisterSchedulerRoutes 注册调度器相关路由.
func RegisterSchedulerRoutes(g *gin.RouterGroup) {
	g.GET("/scheduler/jobs", handle.SchedulerJobs)

	g.POST("/scheduler/jobs/stop", handle.SchedulerStopJobs)

	g.DELETE("/scheduler/jobs/:id", handle.SchedulerRemoveJob)

	g.GET("/scheduler/queue/waiting", handle.SchedulerQueueWaiting)
}
