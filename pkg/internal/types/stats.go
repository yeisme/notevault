package types

// StatsFilesSummary 文件总体统计（当前用户）.
type StatsFilesSummary struct {
	TotalFiles   int   `json:"total_files"`
	ActiveFiles  int   `json:"active_files"`
	TrashedFiles int   `json:"trashed_files"`
	TotalSize    int64 `json:"total_size"`
	ActiveSize   int64 `json:"active_size"`
	TrashedSize  int64 `json:"trashed_size"`
}

// StatsTypeItem 按类型聚合（以 MIME 一级类型或自定义分类为准）.
type StatsTypeItem struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
	Size  int64  `json:"size"`
}

// StatsSizeBucket 单个大小分桶.
type StatsSizeBucket struct {
	Name  string `json:"name"`
	Min   int64  `json:"min"`
	Max   int64  `json:"max"`
	Count int    `json:"count"`
	Size  int64  `json:"size"`
}

// StatsTrendPoint 趋势点（按日）.
type StatsTrendPoint struct {
	Date  string `json:"date"` // YYYY-MM-DD
	Count int    `json:"count"`
	Size  int64  `json:"size"`
}

// StorageSummary 存储总体统计（活跃与回收）.
type StorageSummary struct {
	ActiveObjects int   `json:"active_objects"`
	ActiveSize    int64 `json:"active_size"`
	TrashObjects  int   `json:"trash_objects"`
	TrashSize     int64 `json:"trash_size"`
}

// StorageByBucketItem 按存储桶聚合（活跃对象）.
type StorageByBucketItem struct {
	Bucket string `json:"bucket"`
	Count  int    `json:"count"`
	Size   int64  `json:"size"`
}
