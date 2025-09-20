// Package queue 定义消息主题常量与通配模式，供发布/订阅使用.
package queue

// 主题命名规范：nv.<域>.<动作>[.<状态>][.<子类型>]，尽量稳定且向后兼容.
// 域：object(对象存储)、vector(向量解析)、meta(元数据)、kg(知识图谱)、process(数据处理)、audit(审核)等
// 动作：存储相关(stored/updated/deleted)、处理相关(parse/build/sync/extract)
// 状态：请求(requested)、进行中(ing)、完成(ed)、失败(failed)
// 子类型：针对多模态细分场景(如text/image/audio)

const (
	// 对象存储领域.
	TopicObjectStored      = "nv.object.stored"       // 已写入对象存储（包含版本号、基础元数据） 只有在元数据同步到数据库后，才会触发后续处理流程
	TopicObjectUpdated     = "nv.object.updated"      // 对象存储内容更新（新版本创建）
	TopicObjectDeleted     = "nv.object.deleted"      // 对象从存储中删除（包含被删除的版本信息）
	TopicObjectVersioned   = "nv.object.versioned"    // 对象产生新版本（用于版本控制跟踪）
	TopicObjectRestored    = "nv.object.restored"     // 对象从历史版本恢复
	TopicObjectMoved       = "nv.object.moved"        // 对象存储路径变更
	TopicObjectAccessed    = "nv.object.accessed"     // 对象被访问（用于热点数据统计）
	TopicObjectStorageFull = "nv.object.storage.full" // 对象存储空间不足告警

	// 按数据类型细分的对象存储主题.
	TopicObjectTextStored  = "nv.object.text.stored"  // 文本类型对象存储完成
	TopicObjectImageStored = "nv.object.image.stored" // 图像类型对象存储完成
	TopicObjectAudioStored = "nv.object.audio.stored" // 音频类型对象存储完成
	TopicObjectVideoStored = "nv.object.video.stored" // 视频类型对象存储完成
	TopicObjectMixedStored = "nv.object.mixed.stored" // 混合模态对象存储完成

	// 数据预处理领域.
	TopicProcessRequested = "nv.process.requested"  // 请求数据预处理（格式转换/清洗）
	TopicProcessing       = "nv.process.processing" // 数据预处理中
	TopicProcessed        = "nv.process.processed"  // 数据预处理完成
	TopicProcessFailed    = "nv.process.failed"     // 数据预处理失败

	// 按处理类型细分的预处理主题.
	TopicProcessConverted  = "nv.process.converted"  // 格式转换完成
	TopicProcessCleaned    = "nv.process.cleaned"    // 数据清洗完成
	TopicProcessCompressed = "nv.process.compressed" // 数据压缩完成

	// 向量解析领域.
	TopicVectorParseRequested = "nv.vector.parse.requested" // 请求对指定对象进行内容解析与向量化处理
	TopicVectorParsing        = "nv.vector.parsing"         // 向量解析任务正在执行中
	TopicVectorParsed         = "nv.vector.parsed"          // 向量解析完成
	TopicVectorParseFailed    = "nv.vector.parse.failed"    // 向量解析失败
	TopicVectorIndexed        = "nv.vector.indexed"         // 向量已写入向量数据库
	TopicVectorIndexFailed    = "nv.vector.index.failed"    // 向量索引创建失败

	// 按模态细分的向量解析主题.
	TopicVectorTextParsed  = "nv.vector.text.parsed"  // 文本向量解析完成
	TopicVectorImageParsed = "nv.vector.image.parsed" // 图像向量解析完成
	TopicVectorAudioParsed = "nv.vector.audio.parsed" // 音频向量解析完成

	// 元数据同步领域.
	TopicMetaSyncRequested = "nv.meta.sync.requested" // 请求将对象存储的元数据同步到数据库
	TopicMetaSyncing       = "nv.meta.syncing"        // 元数据同步过程中
	TopicMetaSynced        = "nv.meta.synced"         // 元数据成功同步到数据库
	TopicMetaSyncFailed    = "nv.meta.sync.failed"    // 元数据同步失败
	TopicMetaUpdated       = "nv.meta.updated"        // 数据库中元数据被更新
	TopicMetaDeleted       = "nv.meta.deleted"        // 数据库中元数据被删除
	TopicMetaIndexed       = "nv.meta.indexed"        // 元数据已建立搜索索引
	TopicMetaIndexFailed   = "nv.meta.index.failed"   // 元数据索引创建失败

	// 知识图谱领域.
	TopicKGBuildRequested     = "nv.kg.build.requested"     // 请求基于解析结果构建知识图谱
	TopicKGBuilding           = "nv.kg.building"            // 知识图谱构建中
	TopicKGBuilt              = "nv.kg.built"               // 知识图谱构建完成
	TopicKGBuildFailed        = "nv.kg.build.failed"        // 知识图谱构建失败
	TopicKGUpdated            = "nv.kg.updated"             // 知识图谱已有数据更新
	TopicKGMerged             = "nv.kg.merged"              // 知识图谱数据融合完成
	TopicKGEntityExtracted    = "nv.kg.entity.extracted"    // 实体提取完成
	TopicKGRelationExtracted  = "nv.kg.relation.extracted"  // 关系提取完成
	TopicKGAttributeExtracted = "nv.kg.attribute.extracted" // 属性提取完成

	// 内容审核领域.
	TopicAuditRequested = "nv.audit.requested" // 请求内容审核
	TopicAuditing       = "nv.audit.auditing"  // 内容审核中
	TopicAuditPassed    = "nv.audit.passed"    // 内容审核通过
	TopicAuditRejected  = "nv.audit.rejected"  // 内容审核拒绝
	TopicAuditExpired   = "nv.audit.expired"   // 审核任务超时
)

// 主题分组，用于批量操作或权限控制.
var (
	// 对象存储相关主题集合.
	ObjectTopics = []string{
		TopicObjectStored, TopicObjectUpdated, TopicObjectDeleted,
		TopicObjectVersioned, TopicObjectRestored, TopicObjectMoved,
		TopicObjectAccessed, TopicObjectStorageFull,
		TopicObjectTextStored, TopicObjectImageStored,
		TopicObjectAudioStored, TopicObjectVideoStored,
		TopicObjectMixedStored,
	}

	// 数据预处理相关主题集合.
	ProcessTopics = []string{
		TopicProcessRequested, TopicProcessing, TopicProcessed,
		TopicProcessFailed, TopicProcessConverted, TopicProcessCleaned,
		TopicProcessCompressed,
	}

	// 向量解析相关主题集合.
	VectorTopics = []string{
		TopicVectorParseRequested, TopicVectorParsing, TopicVectorParsed,
		TopicVectorParseFailed, TopicVectorIndexed, TopicVectorIndexFailed,
		TopicVectorTextParsed, TopicVectorImageParsed, TopicVectorAudioParsed,
	}

	// 元数据相关主题集合.
	MetaTopics = []string{
		TopicMetaSyncRequested, TopicMetaSyncing, TopicMetaSynced,
		TopicMetaSyncFailed, TopicMetaUpdated, TopicMetaDeleted,
		TopicMetaIndexed, TopicMetaIndexFailed,
	}

	// 知识图谱相关主题集合.
	KGTopics = []string{
		TopicKGBuildRequested, TopicKGBuilding, TopicKGBuilt,
		TopicKGBuildFailed, TopicKGUpdated, TopicKGMerged,
		TopicKGEntityExtracted, TopicKGRelationExtracted,
		TopicKGAttributeExtracted,
	}

	// 内容审核相关主题集合.
	AuditTopics = []string{
		TopicAuditRequested, TopicAuditing, TopicAuditPassed,
		TopicAuditRejected, TopicAuditExpired,
	}
)
