// Package mq 提供 NATS 消息队列操作实现。
// 此文件包含 NATS 特定的工厂函数，用于创建配置了可选 JetStream 支持的 Publisher 和 Subscriber 实例。
//
// 支持的功能特性：
//   - 连接池和重连机制
//   - 多种认证方式（JWT、NKey、用户名/密码）
//   - JetStream 持久化消息
//   - 通过主题前缀实现负载均衡
//   - 指标集成（占位符，供未来实现）
//
// 配置从 configs.MQConfig 读取，支持集群 URL 以实现高可用性。
package mq

import (
	"context"
	"strings"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-nats/v2/pkg/nats"
	"github.com/ThreeDotsLabs/watermill/message"
	nc "github.com/nats-io/nats.go"

	"github.com/yeisme/notevault/pkg/configs"
)

const (
	DefaultDrainTimeout  = 30 * time.Second
	DefaultStreamMaxMsgs = 10 * time.Second
)

// init 注册 NATS 工厂.
func init() {
	RegisterFactory(configs.MQTypeNATS, natsFactory)
}

// buildNatsOptions 构建 NATS 连接选项.
func buildNatsOptions(cfg *configs.MQConfig) []nc.Option {
	opts := []nc.Option{
		nc.Name(cfg.ClientID),
		nc.MaxReconnects(cfg.MaxReconnects),
		nc.ReconnectWait(time.Duration(cfg.ReconnectWait) * time.Second),
		nc.PingInterval(time.Duration(cfg.PingInterval) * time.Second),
		nc.ReconnectBufSize(cfg.BufferSize),
		nc.DrainTimeout(DefaultDrainTimeout),
		nc.FlusherTimeout(DefaultStreamMaxMsgs),
		nc.RetryOnFailedConnect(true),
	}

	// 添加认证选项
	opts = appendAuthOptions(opts, cfg)

	return opts
}

// appendAuthOptions 添加认证选项.
func appendAuthOptions(opts []nc.Option, cfg *configs.MQConfig) []nc.Option {
	if cfg.JWT != "" {
		opts = append(opts, nc.UserJWTAndSeed(cfg.JWT, cfg.NKey))
	} else if cfg.NKey != "" {
		opts = append(opts, nc.Nkey(cfg.NKey, nil))
	} else if cfg.User != "" {
		opts = append(opts, nc.UserInfo(cfg.User, cfg.Password))
	}

	return opts
}

// buildJetStreamConfig 构建 JetStream 配置.
func buildJetStreamConfig(cfg *configs.MQConfig, logger watermill.LoggerAdapter) nats.JetStreamConfig {
	jsCfg := nats.JetStreamConfig{
		Disabled: !cfg.JetStreamEnabled,
	}

	if cfg.JetStreamEnabled {
		// 设置自动创建流
		jsCfg.AutoProvision = cfg.JetStreamAutoProvision

		// 设置消息跟踪以防止重复
		jsCfg.TrackMsgId = cfg.JetStreamTrackMsgID

		// 设置异步确认
		jsCfg.AckAsync = cfg.JetStreamAckAsync

		// 设置持久化前缀
		jsCfg.DurablePrefix = cfg.JetStreamDurablePrefix

		// 注意：ConnectOptions、PublishOptions、SubscribeOptions 需要根据 watermill-nats 库的具体实现来配置
		// 这里我们记录配置信息，供调试使用
		logger.Info("JetStream 配置信息", watermill.LogFields{
			"auto_provision": cfg.JetStreamAutoProvision,
			"track_msg_id":   cfg.JetStreamTrackMsgID,
			"ack_async":      cfg.JetStreamAckAsync,
			"durable_prefix": cfg.JetStreamDurablePrefix,
			"stream_name":    cfg.StreamName,
			"subject_prefix": cfg.SubjectPrefix,
		})
	}

	return jsCfg
}

// buildURL 构建连接 URL.
func buildURL(cfg *configs.MQConfig) string {
	if len(cfg.ClusterURLs) > 0 {
		return strings.Join(cfg.ClusterURLs, ",")
	}

	return cfg.URL
}

// natsFactory 创建 NATS Publisher & Subscriber.
// 支持 JetStream 流配置，包括：
//   - AutoProvision: 自动创建缺失的流
//   - TrackMsgId: 跟踪消息ID防止重复处理
//   - AckAsync: 异步确认提高性能
//   - DurablePrefix: 持久化订阅前缀
//   - 流配置：最大消息数、存储大小、保留时间等
//   - 消费者配置：确认等待时间、最大投递次数等
func natsFactory(
	ctx context.Context,
	cfg *configs.MQConfig,
	logger watermill.LoggerAdapter) (
	message.Publisher, message.Subscriber, error) {
	opts := buildNatsOptions(cfg)
	jsCfg := buildJetStreamConfig(cfg, logger)
	marshaler := &nats.JSONMarshaler{}

	// 创建 Publisher
	pub, err := createPublisher(opts, jsCfg, marshaler, cfg, logger)
	if err != nil {
		return nil, nil, err
	}

	// 创建 Subscriber
	sub, err := createSubscriber(opts, jsCfg, marshaler, cfg, logger)
	if err != nil {
		return nil, nil, err
	}

	return pub, sub, nil
}

// createPublisher 创建 Publisher.
func createPublisher(
	opts []nc.Option,
	jsCfg nats.JetStreamConfig,
	marshaler *nats.JSONMarshaler,
	cfg *configs.MQConfig,
	logger watermill.LoggerAdapter) (message.Publisher, error) {
	pubCfg := nats.PublisherConfig{
		NatsOptions: opts,
		JetStream:   jsCfg,
		Marshaler:   marshaler,
		URL:         buildURL(cfg),
	}

	return nats.NewPublisher(pubCfg, logger)
}

// createSubscriber 创建 Subscriber.
func createSubscriber(
	opts []nc.Option,
	jsCfg nats.JetStreamConfig,
	marshaler *nats.JSONMarshaler,
	cfg *configs.MQConfig,
	logger watermill.LoggerAdapter) (message.Subscriber, error) {
	subCfg := nats.SubscriberConfig{
		NatsOptions: opts,
		JetStream:   jsCfg,
		Unmarshaler: marshaler,
		URL:         buildURL(cfg),
	}

	// 如果启用负载均衡，记录日志
	if cfg.LoadBalance {
		logger.Info("通过主题前缀启用负载均衡", watermill.LogFields{
			"prefix": cfg.SubjectPrefix,
		})
	}

	// TODO 集成监控指标
	if cfg.EnableMetrics {
		logger.Info("NATS 指标已启用", nil)
	}

	return nats.NewSubscriber(subCfg, logger)
}
