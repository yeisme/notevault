package queue

import "github.com/ThreeDotsLabs/watermill/message"

// -------------------------- 基于业务封装 events --------------------------

// PublishObjectStored 发布 nv.object.stored 事件。
// 用于将对象写入对象存储并同步元数据到数据库后，通知下游流程（如解析、索引等）。
// 可通过可选项 opts 注入 TraceID、Producer 等头部信息。
func PublishObjectStored(pub message.Publisher, payload ObjectStoredPayload, opts ...func(*EventHeader)) error {
	msg, err := NewWatermillMessage(TopicObjectStored, payload, opts...)
	if err != nil {
		return err
	}

	return pub.Publish(TopicObjectStored, msg)
}

// ParseObjectStored 将 Watermill 消息解析为强类型 Envelope（ObjectStoredPayload）。
func ParseObjectStored(msg *message.Message) (Message[ObjectStoredPayload], error) {
	return ParseWatermillMessage[ObjectStoredPayload](msg)
}

// PublishObjectUpdated 发布 nv.object.updated 事件（新版本创建/内容更新）.
func PublishObjectUpdated(pub message.Publisher, payload ObjectUpdatedPayload, opts ...func(*EventHeader)) error {
	msg, err := NewWatermillMessage(TopicObjectUpdated, payload, opts...)
	if err != nil {
		return err
	}

	return pub.Publish(TopicObjectUpdated, msg)
}

func ParseObjectUpdated(msg *message.Message) (Message[ObjectUpdatedPayload], error) {
	return ParseWatermillMessage[ObjectUpdatedPayload](msg)
}

// PublishObjectDeleted 发布 nv.object.deleted 事件（对象从存储中删除）。
func PublishObjectDeleted(pub message.Publisher, payload ObjectDeletedPayload, opts ...func(*EventHeader)) error {
	msg, err := NewWatermillMessage(TopicObjectDeleted, payload, opts...)
	if err != nil {
		return err
	}

	return pub.Publish(TopicObjectDeleted, msg)
}

func ParseObjectDeleted(msg *message.Message) (Message[ObjectDeletedPayload], error) {
	return ParseWatermillMessage[ObjectDeletedPayload](msg)
}

// PublishObjectVersioned 发布 nv.object.versioned 事件（创建了新的对象版本）。
func PublishObjectVersioned(pub message.Publisher, payload ObjectVersionedPayload, opts ...func(*EventHeader)) error {
	msg, err := NewWatermillMessage(TopicObjectVersioned, payload, opts...)
	if err != nil {
		return err
	}

	return pub.Publish(TopicObjectVersioned, msg)
}

func ParseObjectVersioned(msg *message.Message) (Message[ObjectVersionedPayload], error) {
	return ParseWatermillMessage[ObjectVersionedPayload](msg)
}

// PublishObjectRestored 发布 nv.object.restored 事件（从历史版本恢复）。
func PublishObjectRestored(pub message.Publisher, payload ObjectRestoredPayload, opts ...func(*EventHeader)) error {
	msg, err := NewWatermillMessage(TopicObjectRestored, payload, opts...)
	if err != nil {
		return err
	}

	return pub.Publish(TopicObjectRestored, msg)
}

func ParseObjectRestored(msg *message.Message) (Message[ObjectRestoredPayload], error) {
	return ParseWatermillMessage[ObjectRestoredPayload](msg)
}

// PublishObjectMoved 发布 nv.object.moved 事件（路径变更/重命名）。
func PublishObjectMoved(pub message.Publisher, payload ObjectMovedPayload, opts ...func(*EventHeader)) error {
	msg, err := NewWatermillMessage(TopicObjectMoved, payload, opts...)
	if err != nil {
		return err
	}

	return pub.Publish(TopicObjectMoved, msg)
}

func ParseObjectMoved(msg *message.Message) (Message[ObjectMovedPayload], error) {
	return ParseWatermillMessage[ObjectMovedPayload](msg)
}

// PublishObjectAccessed 发布 nv.object.accessed 事件（访问统计/审计）。
func PublishObjectAccessed(pub message.Publisher, payload ObjectAccessedPayload, opts ...func(*EventHeader)) error {
	msg, err := NewWatermillMessage(TopicObjectAccessed, payload, opts...)
	if err != nil {
		return err
	}

	return pub.Publish(TopicObjectAccessed, msg)
}

func ParseObjectAccessed(msg *message.Message) (Message[ObjectAccessedPayload], error) {
	return ParseWatermillMessage[ObjectAccessedPayload](msg)
}

// PublishObjectStorageFull 发布 nv.object.storage.full 事件（空间不足告警）。
func PublishObjectStorageFull(pub message.Publisher, payload ObjectStorageFullPayload, opts ...func(*EventHeader)) error {
	msg, err := NewWatermillMessage(TopicObjectStorageFull, payload, opts...)
	if err != nil {
		return err
	}

	return pub.Publish(TopicObjectStorageFull, msg)
}

func ParseObjectStorageFull(msg *message.Message) (Message[ObjectStorageFullPayload], error) {
	return ParseWatermillMessage[ObjectStorageFullPayload](msg)
}
