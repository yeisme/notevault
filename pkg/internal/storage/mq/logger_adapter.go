// Package mq 提供 Watermill 的日志适配器.
// 此文件实现了 zerolog 适配器，将 Watermill 日志接口与应用的 zerolog 日志器桥接，
// 确保在 MQ 层保持一致的日志记录.
//
// 该适配器支持所有 Watermill 日志级别（Error、Info、Debug、Trace），并正确处理带字段的结构化日志.
package mq

import (
	watermill "github.com/ThreeDotsLabs/watermill"
	"github.com/rs/zerolog"
)

// zerologAdapter 将 zerolog 适配为 watermill.LoggerAdapter.
type zerologAdapter struct {
	l *zerolog.Logger
}

func (z *zerologAdapter) Error(msg string, err error, fields watermill.LogFields) {
	ev := z.l.Error().Err(err)
	for k, v := range fields {
		ev = ev.Interface(k, v)
	}

	ev.Msg(msg)
}

func (z *zerologAdapter) Info(msg string, fields watermill.LogFields) {
	ev := z.l.Info()
	for k, v := range fields {
		ev = ev.Interface(k, v)
	}

	ev.Msg(msg)
}

func (z *zerologAdapter) Debug(msg string, fields watermill.LogFields) {
	ev := z.l.Debug()
	for k, v := range fields {
		ev = ev.Interface(k, v)
	}

	ev.Msg(msg)
}

func (z *zerologAdapter) Trace(msg string, fields watermill.LogFields) {
	ev := z.l.Trace()
	for k, v := range fields {
		ev = ev.Interface(k, v)
	}

	ev.Msg(msg)
}

func (z *zerologAdapter) With(fields watermill.LogFields) watermill.LoggerAdapter {
	l := z.l.With()

	for k, v := range fields {
		l = l.Interface(k, v)
	}

	logger := l.Logger()

	return &zerologAdapter{l: &logger}
}

// String 实现 fmt.Stringer.
func (z *zerologAdapter) String() string { return "zerolog-watermill适配器" }
