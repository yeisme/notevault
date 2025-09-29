// Package queue 管理消息队列，用于异步处理多模态数据解析任务.
//
// 概览
//   - 采用发布/订阅模型，解耦"对象存储、预处理、解析、索引"等环节
//   - 统一的消息封装：Message[Payload] = Header + Payload
//   - 主题常量见 topics.go，负载结构体见 payloads.go
//   - 默认 JSON 编解码（bytedance/sonic），跨语言易解析
//
// 消息信封（Envelope）JSON 结构
//
//	{
//	  "header": {
//	    "topic": "nv.object.stored",
//	    "trace_id": "optional-trace-id",
//	    "producer": "notevault",
//	    "occurred_at": "2025-01-02T03:04:05.123456Z",
//	    "version": "v1"
//	  },
//	  "payload": { ... 取决于具体主题 ... }
//	}
//
// Go 端：发布/订阅示例
//
//	// 1) 构造负载
//	payload := queue.ObjectStoredPayload{
//	  Object: queue.ObjectRef{
//	    Bucket: "nv-bucket",
//	    ObjectKey: "user/2025/09/20/file.txt",
//	    VersionID: "v123",
//	    ETag: "abc123",
//	    Size: 42,
//	    ContentType: "text/plain",
//	  },
//	  Source: "sync",
//	  FileName: "file.txt",
//	}
//
//	// 2) 构造 watermill 消息（可选设置 TraceID/Producer）
//	msg, _ := queue.NewWatermillMessage(
//	  queue.TopicObjectStored, payload,
//	  queue.WithTraceID("trace-xyz"),
//	  queue.WithProducer("notevault"),
//	)
//
//	// 3) 通过 MQ 客户端发布
//	//   client, _ := mq.New(ctx)
//	//   _ = client.Publish(ctx, queue.TopicObjectStored, msg)
//
//	// 4) 订阅（简化展示）
//	//   ch, _ := client.Subscribe(ctx, queue.TopicObjectStored)
//	//   for m := range ch {
//	//       env, _ := queue.ParseWatermillMessage[queue.ObjectStoredPayload](m)
//	//       // 使用 env.Header / env.Payload ...
//	//       m.Ack()
//	//   }
//
// Python 端：解析示例
//
//	# pip install pydantic (或使用 dataclasses + typing)
//	from datetime import datetime
//	from typing import Any, Optional
//	from pydantic import BaseModel
//	import json
//
//	class EventHeader(BaseModel):
//	    topic: str
//	    trace_id: Optional[str] = None
//	    producer: Optional[str] = None
//	    occurred_at: datetime    # RFC3339 / ISO8601，可由 pydantic 自动解析
//	    version: Optional[str] = None
//
//	class ObjectRef(BaseModel):
//	    bucket: str
//	    object_key: str
//	    version_id: Optional[str] = None
//	    etag: Optional[str] = None
//	    size: Optional[int] = None
//	    hash: Optional[str] = None
//	    content_type: Optional[str] = None
//	    tags: Optional[dict[str, str]] = None
//
//	class ObjectStoredPayload(BaseModel):
//	    object: ObjectRef
//	    source: Optional[str] = None
//	    file_name: Optional[str] = None
//
//	class Envelope(BaseModel):
//	    header: EventHeader
//	    payload: Any  # 或按主题选择对应的 Payload 模型
//
//	# 假设收到的消息体为 bytes -> body
//	def parse_envelope(body: bytes) -> Envelope:
//	    data = json.loads(body.decode('utf-8'))
//	    env = Envelope.model_validate(data)
//	    # 可按主题二次解析 payload
//	    if env.header.topic == 'nv.object.stored':
//	        env.payload = ObjectStoredPayload.model_validate(data['payload'])
//	    return env
//
// 注意事项
//  1. occurred_at 为 UTC，RFC3339 格式；Python 端用 datetime / pydantic 可直接解析
//  2. version 便于后向兼容，建议消费者忽略未知字段
//  3. Header.topic 与消息中间件的 Subject/Topic 可能重复，意在离线可追踪
//  4. 若需要业务级幂等，可将消息 ID 设为"确定性键"（如 bucket|object_key|version_id 的哈希）

// 参考：topics.go（主题）、payloads.go（负载）、internal/storage/mq（MQ 客户端封装）.
package queue

import (
	"time"

	watermill "github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/bytedance/sonic"
)

const (
	PayloadVersionV1 string = "v1"
)

// NewEventHeader 便捷创建事件头.
func NewEventHeader(topic string, opts ...func(*EventHeader)) EventHeader {
	hdr := EventHeader{
		Topic:      topic,
		OccurredAt: time.Now().UTC(),
		Version:    PayloadVersionV1,
	}
	for _, opt := range opts {
		opt(&hdr)
	}

	return hdr
}

// WithTraceID 设置 TraceID.
func WithTraceID(id string) func(*EventHeader) { return func(h *EventHeader) { h.TraceID = id } }

// WithProducer 设置 Producer.
func WithProducer(p string) func(*EventHeader) { return func(h *EventHeader) { h.Producer = p } }

// Encode 将消息封装为 JSON 字节切片.
func Encode[T any](msg Message[T]) ([]byte, error) { return sonic.Marshal(msg) }

// Decode 从 JSON 字节解码为消息.
func Decode[T any](b []byte) (Message[T], error) {
	var m Message[T]

	err := sonic.Unmarshal(b, &m)

	return m, err
}

// NewWatermillMessage 构造一个 watermill 消息，设置 ID 与元数据.
func NewWatermillMessage[T any](topic string, payload T, opts ...func(*EventHeader)) (*message.Message, error) {
	header := NewEventHeader(topic, opts...)
	env := Message[T]{Header: header, Payload: payload}

	data, err := Encode(env)
	if err != nil {
		return nil, err
	}

	msg := message.NewMessage(watermill.NewUUID(), data)
	msg.Metadata.Set("topic", topic)

	if header.TraceID != "" {
		msg.Metadata.Set("trace_id", header.TraceID)
	}

	if header.Producer != "" {
		msg.Metadata.Set("producer", header.Producer)
	}

	msg.Metadata.Set("occurred_at", header.OccurredAt.Format(time.RFC3339Nano))

	if header.Version != "" {
		msg.Metadata.Set("version", header.Version)
	}

	return msg, nil
}

// ParseWatermillMessage 解出泛型负载.
func ParseWatermillMessage[T any](msg *message.Message) (Message[T], error) {
	return Decode[T](msg.Payload)
}
