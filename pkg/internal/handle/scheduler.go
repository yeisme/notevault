package handle

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/yeisme/notevault/pkg/middleware"
)

// SchedulerJobs 返回所有调度器任务信息.
func SchedulerJobs(c *gin.Context) {
	sched := middleware.GetScheduler(c)
	jobs := sched.GetJobInfos()
	c.JSON(http.StatusOK, gin.H{"jobs": jobs})
}

// SchedulerStopJobs 停止所有任务.
func SchedulerStopJobs(c *gin.Context) {
	sched := middleware.GetScheduler(c)

	if err := sched.StopJobs(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "jobs stopped"})
}

// SchedulerRemoveJob 根据 id 删除任务.
func SchedulerRemoveJob(c *gin.Context) {
	sched := middleware.GetScheduler(c)

	idStr := c.Param("id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job id"})
		return
	}

	if err := sched.RemoveJob(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "job removed"})
}

// SchedulerQueueWaiting 返回队列中等待的任务数.
func SchedulerQueueWaiting(c *gin.Context) {
	sched := middleware.GetScheduler(c)

	waiting := sched.JobsWaitingInQueue()
	c.JSON(http.StatusOK, gin.H{"waiting": waiting})
}
