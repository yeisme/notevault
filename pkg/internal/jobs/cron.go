// Package jobs 负责注册与实现业务定时任务（基于 scheduler）。
package jobs

import (
	"context"
	"fmt"
	"time"

	ctxPkg "github.com/yeisme/notevault/pkg/context"
	"github.com/yeisme/notevault/pkg/internal/model"
	"github.com/yeisme/notevault/pkg/internal/service"
	"github.com/yeisme/notevault/pkg/internal/storage"
	"github.com/yeisme/notevault/pkg/log"
	"github.com/yeisme/notevault/pkg/scheduler"
)

// RegisterCronJobs 配置业务定时任务：
//   - 每天 07:00 和 19:00 执行回收站自动清理（默认 30 天前）
//   - 每天 02:10 执行上一日的单日同步
//   - 每月 1 号 03:30 执行全量同步
func RegisterCronJobs(sched *scheduler.Scheduler, mgr *storage.Manager) error {
	if sched == nil {
		return fmt.Errorf("scheduler is nil")
	}

	if mgr == nil {
		return fmt.Errorf("storage manager is nil")
	}

	// 将 storage manager 注入到 context，便于 service 使用
	baseCtx := ctxPkg.WithStorageManager(context.Background(), mgr)

	// 每天 07:00 自动清理回收站
	_ = sched.AddCron(JobTrashAutoCleanMorning, CronTrashAutoCleanMorning, func(ctx context.Context) {
		runTrashAutoClean(ctx, mgr)
	}, baseCtx)

	// 每天 19:00 自动清理回收站
	_ = sched.AddCron(JobTrashAutoCleanEvening, CronTrashAutoCleanEvening, func(ctx context.Context) {
		runTrashAutoClean(ctx, mgr)
	}, baseCtx)

	// 每天 02:10 执行上一日的单日同步
	_ = sched.AddCron(JobMetaSyncDaily, CronMetaSyncDaily, func(ctx context.Context) {
		runDailySync(ctx, mgr)
	}, baseCtx)

	// 每月 1 号 03:30 全量同步
	_ = sched.AddCron(JobMetaSyncMonthlyFull, CronMetaSyncMonthlyFull, func(ctx context.Context) {
		runMonthlyFullSync(ctx, mgr)
	}, baseCtx)

	return nil
}

// runTrashAutoClean 遍历所有用户，执行回收站 30 天前的自动清理。
func runTrashAutoClean(ctx context.Context, mgr *storage.Manager) {
	l := log.Logger().With().Str("job", "trash.auto_clean").Logger()

	users, err := listAllUsers(ctx, mgr)
	if err != nil {
		l.Error().Err(err).Msg("list users failed")
		return
	}

	before := time.Now().AddDate(0, 0, -30)

	for _, u := range users {
		svc := service.NewTrashService(ctx)

		n, e := svc.AutoClean(ctx, u, before)
		if e != nil {
			l.Error().Err(e).Str("user", u).Msg("auto clean failed")
			continue
		}

		if n > 0 {
			l.Info().Str("user", u).Int("affected", n).Time("before", before).Msg("auto cleaned trash")
		}
	}
}

// runDailySync 同步上一日的数据到数据库。
func runDailySync(ctx context.Context, mgr *storage.Manager) {
	l := log.Logger().With().Str("job", "meta.sync.daily").Logger()

	users, err := listAllUsers(ctx, mgr)
	if err != nil {
		l.Error().Err(err).Msg("list users failed")
		return
	}
	// 取上一日的 UTC 年月日
	ymd := time.Now().UTC().Add(-24 * time.Hour)
	y, m, d := ymd.Date()

	for _, u := range users {
		svc := service.NewFileService(ctx)
		if e := svc.SyncObjectsToDBByDate(ctx, u, y, int(m), d); e != nil {
			l.Error().Err(e).Str("user", u).Int("year", y).Int("month", int(m)).Int("day", d).Msg("daily sync failed")
			continue
		}

		l.Info().Str("user", u).Int("year", y).Int("month", int(m)).Int("day", d).Msg("daily sync done")
	}
}

// runMonthlyFullSync 全量同步所有用户的数据到数据库。
func runMonthlyFullSync(ctx context.Context, mgr *storage.Manager) {
	l := log.Logger().With().Str("job", "meta.sync.monthly_full").Logger()

	users, err := listAllUsers(ctx, mgr)
	if err != nil {
		l.Error().Err(err).Msg("list users failed")
		return
	}

	for _, u := range users {
		svc := service.NewFileService(ctx)
		if e := svc.SyncObjectsToDB(ctx, u); e != nil {
			l.Error().Err(e).Str("user", u).Msg("full sync failed")
			continue
		}

		l.Info().Str("user", u).Msg("full sync done")
	}
}

// listAllUsers 查询 DB 中存在文件记录的所有用户。
func listAllUsers(ctx context.Context, mgr *storage.Manager) ([]string, error) {
	if mgr == nil || mgr.GetDBClient() == nil || mgr.GetDBClient().GetDB() == nil {
		return nil, fmt.Errorf("db not initialized")
	}

	dbx := mgr.GetDBClient().GetDB().WithContext(ctx)

	var users []string
	if err := dbx.Model(&model.Files{}).Distinct().Pluck("user", &users).Error; err != nil {
		return nil, err
	}

	return users, nil
}
