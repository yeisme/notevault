package jobs

// 任务名称常量，便于统一管理与引用.
const (
	JobTrashAutoCleanMorning = "trash.auto_clean.morning"
	JobTrashAutoCleanEvening = "trash.auto_clean.evening"
	JobMetaSyncDaily         = "meta.sync.daily"
	JobMetaSyncMonthlyFull   = "meta.sync.monthly_full"
)

// Cron 表达式常量（可选，但推荐一并集中管理）.
const (
	CronTrashAutoCleanMorning = "0 7 * * *"
	CronTrashAutoCleanEvening = "0 19 * * *"
	CronMetaSyncDaily         = "10 2 * * *"
	CronMetaSyncMonthlyFull   = "30 3 1 * *"
)
