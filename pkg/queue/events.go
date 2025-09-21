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
