// Package scheduler 提供定时任务调度功能，使用 gocron/v2 库.
package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/yeisme/notevault/pkg/log"
)

const (
	// updateInterval 定义状态更新间隔.
	updateInterval = 10 * time.Second
)

// JobStatus 表示任务的状态类型.
type JobStatus string

const (
	StatusScheduled JobStatus = "scheduled" // 任务已调度
	StatusRunning   JobStatus = "running"   // 任务正在运行
	StatusStopped   JobStatus = "stopped"   // 任务已停止
	StatusError     JobStatus = "error"     // 任务出错
)

// JobInfo 表示定时任务的信息，用于可视化和监控.
type JobInfo struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	CronExpr    string    `json:"cron_expr"`
	NextRun     time.Time `json:"next_run"`
	LastRun     time.Time `json:"last_run"`
	LastSuccess time.Time `json:"last_success,omitempty"`
	Status      JobStatus `json:"status"`
	Error       string    `json:"error,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Scheduler 是定时任务调度器的实现.
type Scheduler struct {
	scheduler gocron.Scheduler
	jobs      map[string]gocron.Job // 以任务名称为键
	jobInfos  map[string]*JobInfo   // 以任务名称为键
	jobIDs    map[uuid.UUID]string  // 以任务ID为键，映射到名称
	mu        sync.RWMutex
	logger    *zerolog.Logger
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewScheduler 创建一个新的 Scheduler 实例.
func NewScheduler() (*Scheduler, error) {
	s, err := gocron.NewScheduler()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	logger := log.Logger()

	scheduler := &Scheduler{
		scheduler: s,
		jobs:      make(map[string]gocron.Job),
		jobInfos:  make(map[string]*JobInfo),
		jobIDs:    make(map[uuid.UUID]string),
		logger:    logger,
		ctx:       ctx,
		cancel:    cancel,
	}

	// 启动后台任务状态更新器
	go scheduler.jobStatusUpdater()

	return scheduler, nil
}

// AddCron 添加一个基于 cron 表达式的定时任务.
func (s *Scheduler) AddCron(name string, cronExpr string, job func(ctx context.Context), ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.jobs[name]; exists {
		return fmt.Errorf("job with name %s already exists", name)
	}

	// 创建包装函数以捕获执行状态
	wrappedJob := func(ctx context.Context) {
		jobInfo := s.getJobInfoByName(name)
		if jobInfo != nil {
			s.updateJobStatus(name, StatusRunning, "")

			defer func() {
				if r := recover(); r != nil {
					errMsg := fmt.Sprintf("panic in job: %v", r)
					s.updateJobStatus(name, StatusError, errMsg)
					s.logger.Error().Str("job", name).Interface("panic", r).Msg("Job panicked")
				}
			}()
		}

		// 执行实际任务
		job(ctx)

		if jobInfo != nil {
			s.updateJobStatus(name, StatusScheduled, "")

			jobInfo.LastSuccess = time.Now()
		}
	}

	j, err := s.scheduler.NewJob(
		gocron.CronJob(cronExpr, false),
		gocron.NewTask(wrappedJob, ctx),
		gocron.WithName(name),
		gocron.WithEventListeners(
			gocron.AfterJobRuns(func(jobID uuid.UUID, jobName string) {
				s.mu.Lock()
				defer s.mu.Unlock()

				if info, exists := s.jobInfos[jobName]; exists {
					info.LastRun = time.Now()
					info.UpdatedAt = time.Now()
				}
			}),
		),
	)
	if err != nil {
		return err
	}

	jobID := j.ID()
	now := time.Now()
	nextRun, _ := j.NextRun()

	s.jobs[name] = j
	s.jobIDs[jobID] = name
	s.jobInfos[name] = &JobInfo{
		ID:        jobID.String(),
		Name:      name,
		CronExpr:  cronExpr,
		NextRun:   nextRun,
		Status:    StatusScheduled,
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.logger.Info().Str("job", name).Str("cron", cronExpr).Msg("Added cron job")

	return nil
}

// RemoveJobByName 通过名称移除任务.
func (s *Scheduler) RemoveJobByName(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[name]
	if !exists {
		return fmt.Errorf("job with name %s does not exist", name)
	}

	err := s.scheduler.RemoveJob(job.ID())
	if err != nil {
		return err
	}

	// 清理内部映射
	delete(s.jobs, name)
	delete(s.jobInfos, name)
	delete(s.jobIDs, job.ID())

	s.logger.Info().Str("job", name).Msg("Removed job")

	return nil
}

// GetJobByName 通过名称获取任务.
func (s *Scheduler) GetJobByName(name string) (gocron.Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, exists := s.jobs[name]
	if !exists {
		return nil, fmt.Errorf("job with name %s does not exist", name)
	}

	return job, nil
}

// GetJobInfoByName 通过名称获取任务信息.
func (s *Scheduler) GetJobInfoByName(name string) (*JobInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	info, exists := s.jobInfos[name]
	if !exists {
		return nil, fmt.Errorf("job with name %s does not exist", name)
	}

	return info, nil
}

// Start 启动调度器.
func (s *Scheduler) Start() {
	s.logger.Info().Msg("Starting scheduler")
	s.scheduler.Start()
}

// Stop 停止调度器.
func (s *Scheduler) Stop() error {
	s.logger.Info().Msg("Stopping scheduler")
	s.cancel()

	return s.scheduler.Shutdown()
}

// Jobs returns all the jobs currently in the scheduler.
func (s *Scheduler) Jobs() []gocron.Job {
	return s.scheduler.Jobs()
}

// NewJob creates a new job in the Scheduler.
func (s *Scheduler) NewJob(def gocron.JobDefinition, task gocron.Task, opts ...gocron.JobOption) (gocron.Job, error) {
	return s.scheduler.NewJob(def, task, opts...)
}

// RemoveByTags removes all jobs that have at least one of the provided tags.
func (s *Scheduler) RemoveByTags(tags ...string) {
	s.scheduler.RemoveByTags(tags...)
}

// RemoveJob removes the job with the provided id.
func (s *Scheduler) RemoveJob(id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	name, exists := s.jobIDs[id]
	if exists {
		delete(s.jobs, name)
		delete(s.jobInfos, name)
		delete(s.jobIDs, id)
	}

	return s.scheduler.RemoveJob(id)
}

// Shutdown shuts down the scheduler.
func (s *Scheduler) Shutdown() error {
	s.cancel()
	return s.scheduler.Shutdown()
}

// StopJobs stops the execution of all jobs.
func (s *Scheduler) StopJobs() error {
	return s.scheduler.StopJobs()
}

// Update replaces the existing Job's JobDefinition.
func (s *Scheduler) Update(id uuid.UUID, def gocron.JobDefinition, task gocron.Task, opts ...gocron.JobOption) (gocron.Job, error) {
	return s.scheduler.Update(id, def, task, opts...)
}

// JobsWaitingInQueue number of jobs waiting in Queue.
func (s *Scheduler) JobsWaitingInQueue() int {
	return s.scheduler.JobsWaitingInQueue()
}

// GetJobInfos 返回所有定时任务的信息，用于可视化和监控.
func (s *Scheduler) GetJobInfos() []JobInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]JobInfo, 0, len(s.jobInfos))
	for _, info := range s.jobInfos {
		jobs = append(jobs, *info)
	}

	return jobs
}

// getJobInfoByName 通过名称获取任务信息（内部使用，不加锁）.
func (s *Scheduler) getJobInfoByName(name string) *JobInfo {
	return s.jobInfos[name]
}

// jobStatusUpdater 定期更新任务状态.
func (s *Scheduler) jobStatusUpdater() {
	ticker := time.NewTicker(updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.updateAllJobStatuses()
		}
	}
}

// updateAllJobStatuses 更新所有任务的状态信息.
func (s *Scheduler) updateAllJobStatuses() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for name, job := range s.jobs {
		info := s.jobInfos[name]
		if info == nil {
			continue
		}

		// 更新下次运行时间
		if nextRun, err := job.NextRun(); err == nil {
			info.NextRun = nextRun
		}

		// 更新上次运行时间
		if lastRun, err := job.LastRun(); err == nil {
			info.LastRun = lastRun
		}

		// 更新状态
		info.Status = StatusScheduled
		info.UpdatedAt = time.Now()
	}
}

// updateJobStatus 更新任务状态.
func (s *Scheduler) updateJobStatus(name string, status JobStatus, errorMsg string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if info, exists := s.jobInfos[name]; exists {
		info.Status = status
		info.Error = errorMsg
		info.UpdatedAt = time.Now()
	}
}
