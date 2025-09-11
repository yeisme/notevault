// Package mq 提供基于 Watermill 库的统一消息队列操作接口.
// 支持发布/订阅模式，并通过工厂模式抽象不同的 MQ 实现.
//
// 支持的 MQ 类型：
//   - NATS（支持 JetStream）
//
// 该包提供封装了 Publisher 和 Subscriber 的 Client，以及便捷的消息发布和订阅方法.
//
// 使用示例：
//
//	import "github.com/yeisme/notevault/pkg/internal/mq"
//
//	ctx := context.Background()
//	client, err := mq.New(ctx)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer client.Close()
//
//	// 发布消息
//	msg := message.NewMessage(watermill.NewUUID(), []byte("hello world"))
//	err = client.Publish(ctx, "topic", msg)
//
//	// 订阅主题
//	err = client.Subscribe(ctx, "topic", func(msg *message.Message) error {
//		fmt.Println(string(msg.Payload))
//		msg.Ack()
//		return nil
//	})
package mq

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	watermill "github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/metrics"
	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/yeisme/notevault/pkg/configs"
	nlog "github.com/yeisme/notevault/pkg/log"
)

// Factory 定义创建 Publisher + Subscriber 的工厂函数.
type Factory func(ctx context.Context, config any, logger watermill.LoggerAdapter) (message.Publisher, message.Subscriber, error)

var (
	factories = map[configs.MQType]Factory{}
)

const (
	strictConnectDialTimeout = 2 * time.Second // Strict 模式下的拨号超时
)

// RegisterFactory 注册指定 MQType 的工厂.
func RegisterFactory(t configs.MQType, f Factory) {
	factories[t] = f
}

// Client 封装 watermill Publisher 与 Subscriber.
type Client struct {
	publisher  message.Publisher
	subscriber message.Subscriber
	router     *message.Router
	closeFunc  func() // 用于关闭metrics服务器
}

// Publish 便捷发布.
func (c *Client) Publish(ctx context.Context, topic string, msgs ...*message.Message) error {
	if c == nil || c.publisher == nil {
		return fmt.Errorf("mq publisher not initialized")
	}

	for _, m := range msgs {
		if err := c.publisher.Publish(topic, m); err != nil {
			return err
		}
	}

	return nil
}

// Subscribe 便捷订阅.
func (c *Client) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	if c == nil || c.subscriber == nil {
		return nil, fmt.Errorf("mq subscriber not initialized")
	}
	// 调整调用以匹配签名：只传递 ctx 和 topic
	ch, err := c.subscriber.Subscribe(ctx, topic)
	if err != nil {
		return nil, err
	}

	return ch, nil
}

// Close 关闭资源.
func (c *Client) Close() error {
	var err error

	if c.publisher != nil {
		if e := c.publisher.Close(); e != nil {
			err = e
		}
	}

	if c.subscriber != nil {
		if e := c.subscriber.Close(); e != nil {
			err = e
		}
	}

	if c.router != nil {
		// 停止 router，确保所有 handler 停止运行
		if e := c.router.Close(); e != nil {
			err = e
		}
	}

	if c.closeFunc != nil {
		c.closeFunc()
	}

	return err
}

var (
	mqOnce sync.Once
	mqInst *Client
	mqErr  error
)

// validateBasicConfig 验证 MQ 配置的基本要求.
func validateBasicConfig(cfg *configs.MQConfig) error {
	if cfg.Common.URL == "" && len(cfg.NATS.ClusterURLs) == 0 {
		return errors.New("mq url or cluster_urls required")
	}

	if _, ok := factories[cfg.Type]; !ok {
		return fmt.Errorf("unsupported mq type: %s", cfg.Type)
	}

	return nil
}

// pickProbeTarget 从配置中选择一个用于 TCP 探测的目标地址.
func pickProbeTarget(cfg *configs.MQConfig) string {
	target := cfg.Common.URL
	if target == "" && len(cfg.NATS.ClusterURLs) > 0 {
		target = cfg.NATS.ClusterURLs[0]
	}

	if strings.Contains(target, ",") { // 多地址只取第一个探测
		target = strings.Split(target, ",")[0]
	}

	if !strings.Contains(target, ":") {
		target += ":4222"
	}

	return target
}

// strictProbe 在 StrictConnect 模式下尝试建立 TCP 连接以验证可达性.
func strictProbe(ctx context.Context, cfg *configs.MQConfig) error {
	if !cfg.Common.StrictConnect {
		return nil
	}

	target := pickProbeTarget(cfg)
	d := net.Dialer{Timeout: strictConnectDialTimeout}

	conn, err := d.DialContext(ctx, "tcp", target)
	if err != nil {
		return fmt.Errorf("strict connect dial %s failed: %w", target, err)
	}

	_ = conn.Close()

	return nil
}

// enableMetricsIfNeeded 根据配置决定是否启用 Watermill 的 Prometheus 指标支持.
func enableMetricsIfNeeded(
	ctx context.Context,
	pub message.Publisher,
	sub message.Subscriber,
	logger watermill.LoggerAdapter,
) (message.Publisher, message.Subscriber, *message.Router, func(), error) {
	metricsCfg := configs.GetConfig().MQ
	if !metricsCfg.Common.EnableMetrics {
		return pub, sub, nil, nil, nil
	}

	prometheusRegistry, closeMetricsServer := metrics.CreateRegistryAndServeHTTP(metricsCfg.Common.Endpoint)

	router, err := message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("create router: %w", err)
	}

	go func() {
		if runErr := router.Run(ctx); runErr != nil {
			nlog.Logger().Error().Err(runErr).Msg("router run error")
		}
	}()

	metricsBuilder := metrics.NewPrometheusMetricsBuilder(prometheusRegistry, "", "")
	metricsBuilder.AddPrometheusRouterMetrics(router)

	decoratedPub, err := metricsBuilder.DecoratePublisher(pub)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("decorate publisher with metrics: %w", err)
	}

	decoratedSub, err := metricsBuilder.DecorateSubscriber(sub)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("decorate subscriber with metrics: %w", err)
	}

	nlog.Logger().Info().Str("endpoint", metricsCfg.Common.Endpoint).Msg("MQ metrics enabled")

	return decoratedPub, decoratedSub, router, closeMetricsServer, nil
}

// New 初始化消息队列（单例）.
func New(ctx context.Context) (*Client, error) {
	mqOnce.Do(func() {
		cfg := configs.GetConfig().MQ

		if _, ok := factories[cfg.Type]; !ok {
			mqErr = fmt.Errorf("unsupported mq type: %s", cfg.Type)
			return
		}

		var config any

		switch cfg.Type {
		case configs.MQTypeNATS:
			config = &cfg
		case configs.MQTypeRedis:
			config = &cfg
		default:
			config = &cfg
		}

		if err := strictProbe(ctx, &cfg); err != nil {
			mqErr = err
			return
		}

		logger := &zerologAdapter{l: nlog.Logger()}
		factory := factories[cfg.Type]

		pub, sub, err := factory(ctx, config, logger)
		if err != nil {
			mqErr = fmt.Errorf("init mq (%s): %w", cfg.Type, err)
			return
		}

		pub, sub, router, closeFunc, err := enableMetricsIfNeeded(ctx, pub, sub, logger)
		if err != nil {
			mqErr = err
			return
		}

		mqInst = &Client{publisher: pub, subscriber: sub, router: router, closeFunc: closeFunc}

		nlog.Logger().Info().Str("type", string(cfg.Type)).Bool("strict", cfg.Common.StrictConnect).Msg("MQ 管理器已初始化")
	})

	return mqInst, mqErr
}
