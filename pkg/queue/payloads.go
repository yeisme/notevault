package queue

import "time"

// EventHeader 定义所有事件的通用头部元数据.
// 建议在发布消息时填充 TraceID、OccurredAt、Producer 等，便于追踪链路与审计.
type EventHeader struct {
	// Topic 冗余记录消息主题，便于离线处理或转储后定位来源主题.
	Topic string `json:"topic"`
	// TraceID 分布式追踪/关联 ID，可来自中间件或业务生成.
	TraceID string `json:"trace_id,omitempty"`
	// Producer 生产者服务名或节点标识.
	Producer string `json:"producer,omitempty"`
	// OccurredAt 事件发生时间（UTC，RFC3339）.
	OccurredAt time.Time `json:"occurred_at"`
	// Version 事件负载版本，便于向后兼容演进.
	Version string `json:"version,omitempty"`
}

// Message 是统一的消息封装，Header + Payload.
// T 即不同主题对应的负载结构体.
type Message[T any] struct {
	Header  EventHeader `json:"header"`
	Payload T           `json:"payload"`
}

// -------------------------- 对象存储领域 --------------------------

// ObjectRef 标识对象在对象存储与版本信息.
type ObjectRef struct {
	Bucket      string            `json:"bucket"`
	ObjectKey   string            `json:"object_key"`
	VersionID   string            `json:"version_id,omitempty"`
	ETag        string            `json:"etag,omitempty"`
	Size        int64             `json:"size,omitempty"`
	Hash        string            `json:"hash,omitempty"`
	ContentType string            `json:"content_type,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
}

// ObjectStoredPayload 已写入对象存储（含基础元数据与版本）.
type ObjectStoredPayload struct {
	Object ObjectRef `json:"object"`
	// Optional 业务上下文，如触发来源（用户/任务）、文件名等.
	Source   string `json:"source,omitempty"`
	FileName string `json:"file_name,omitempty"`
}

// ObjectUpdatedPayload 对象存储内容更新（新版本创建）.
type ObjectUpdatedPayload struct {
	Object        ObjectRef `json:"object"`
	PrevVersionID string    `json:"prev_version_id,omitempty"`
}

// ObjectDeletedPayload 对象被删除（可包含被删除的版本）.
type ObjectDeletedPayload struct {
	Object       ObjectRef `json:"object"`
	DeletedAll   bool      `json:"deleted_all,omitempty"`   // 是否删除所有版本
	DeletedSince string    `json:"deleted_since,omitempty"` // 起始删除时间（RFC3339），可选
}

// -------------------------- 向量解析领域 --------------------------

// VectorParseRequestedPayload 请求解析并向量化.
type VectorParseRequestedPayload struct {
	Object         ObjectRef `json:"object"`
	Language       string    `json:"language,omitempty"`   // 预期语言/locale
	Strategy       string    `json:"strategy,omitempty"`   // 解析策略（ocr, text, auto...）
	MaxTokens      int       `json:"max_tokens,omitempty"` // 限制解析 token
	EmbeddingModel string    `json:"embedding_model,omitempty"`
}

// VectorProgressPayload 任务进行中，支持进度汇报.
type VectorProgressPayload struct {
	Object   ObjectRef `json:"object"`
	TaskID   string    `json:"task_id,omitempty"`
	Progress int       `json:"progress,omitempty"` // 0-100
	Message  string    `json:"message,omitempty"`
}

// VectorParsedPayload 解析完成，包含索引位置等.
type VectorParsedPayload struct {
	Object      ObjectRef `json:"object"`
	Segments    int       `json:"segments,omitempty"`
	VectorIndex string    `json:"vector_index,omitempty"` // 如 collection/table 名
	Namespace   string    `json:"namespace,omitempty"`
}

// VectorParseFailedPayload 解析失败.
type VectorParseFailedPayload struct {
	Object ObjectRef `json:"object"`
	TaskID string    `json:"task_id,omitempty"`
	Error  string    `json:"error"`
}

// -------------------------- 元数据同步领域 --------------------------

// MetaSyncRequestedPayload 请求从对象存储同步元数据到数据库.
type MetaSyncRequestedPayload struct {
	Object ObjectRef `json:"object"`
	Force  bool      `json:"force,omitempty"`
}

// MetaSyncProgressPayload 同步中状态.
type MetaSyncProgressPayload struct {
	Object   ObjectRef `json:"object"`
	Progress int       `json:"progress,omitempty"`
	Message  string    `json:"message,omitempty"`
}

// MetaSyncedPayload 同步完成.
type MetaSyncedPayload struct {
	Object ObjectRef `json:"object"`
}

// MetaSyncFailedPayload 同步失败.
type MetaSyncFailedPayload struct {
	Object ObjectRef `json:"object"`
	Error  string    `json:"error"`
}

// -------------------------- 知识图谱领域 --------------------------

// KGBuildRequestedPayload 请求基于解析结果构建知识图谱.
type KGBuildRequestedPayload struct {
	Object      ObjectRef `json:"object"`
	Namespace   string    `json:"namespace,omitempty"`
	BuildPolicy string    `json:"build_policy,omitempty"` // 构建策略：full/incremental
}

// KGProgressPayload 构建中.
type KGProgressPayload struct {
	Object   ObjectRef `json:"object"`
	TaskID   string    `json:"task_id,omitempty"`
	Progress int       `json:"progress,omitempty"`
	Message  string    `json:"message,omitempty"`
}

// -------------------------- 数据预处理领域 --------------------------

// ProcessProgressPayload 预处理进度（适用于 nv.process.progress）.
// 可选字段 Stage 用于描述当前阶段（如 convert/clean/compress）.
type ProcessProgressPayload struct {
	Object   ObjectRef `json:"object"`
	TaskID   string    `json:"task_id,omitempty"`
	Stage    string    `json:"stage,omitempty"`
	Progress int       `json:"progress,omitempty"`
	Message  string    `json:"message,omitempty"`
}

// -------------------------- 审核领域 --------------------------

// AuditProgressPayload 审核进度（适用于 nv.audit.progress）.
type AuditProgressPayload struct {
	Object   ObjectRef `json:"object"`
	TaskID   string    `json:"task_id,omitempty"`
	Progress int       `json:"progress,omitempty"`
	Message  string    `json:"message,omitempty"`
}

// KGBuiltPayload 构建完成.
type KGBuiltPayload struct {
	Object    ObjectRef `json:"object"`
	Entities  int       `json:"entities,omitempty"`
	Relations int       `json:"relations,omitempty"`
}

// KGBuildFailedPayload 构建失败.
type KGBuildFailedPayload struct {
	Object ObjectRef `json:"object"`
	Error  string    `json:"error"`
}
