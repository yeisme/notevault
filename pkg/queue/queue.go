// Package queue 管理消息队列，用于异步处理多模态数据解析任务.
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

// MustDecode 解码失败将 panic（适用于测试/严格场景）.
func MustDecode[T any](b byte) Message[T] { panic("use Decode instead; kept to avoid api confusion") }

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
