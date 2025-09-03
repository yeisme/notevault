// Package mq 提供基于 Watermill 库的统一消息队列操作接口。
// 支持发布/订阅模式，并通过工厂模式抽象不同的 MQ 实现。
//
// 支持的 MQ 类型：
//   - NATS（支持 JetStream）
//
// 该包提供封装了 Publisher 和 Subscriber 的 Client，以及便捷的消息发布和订阅方法。
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
	"fmt"
	"sync"

	watermill "github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/yeisme/notevault/pkg/configs"
	nlog "github.com/yeisme/notevault/pkg/log"
)

// Factory 定义创建 Publisher + Subscriber 的工厂函数.
type Factory func(ctx context.Context, cfg *configs.MQConfig, logger watermill.LoggerAdapter) (message.Publisher, message.Subscriber, error)

var (
	factories = map[configs.MQType]Factory{}
)

// RegisterFactory 注册指定 MQType 的工厂.
func RegisterFactory(t configs.MQType, f Factory) {
	factories[t] = f
}

// Client 封装 watermill Publisher 与 Subscriber.
type Client struct {
	Publisher  message.Publisher
	Subscriber message.Subscriber
}

// Publish 便捷发布.
func (c *Client) Publish(ctx context.Context, topic string, msgs ...*message.Message) error {
	if c == nil || c.Publisher == nil {
		return fmt.Errorf("mq publisher not initialized")
	}

	for _, m := range msgs {
		if err := c.Publisher.Publish(topic, m); err != nil {
			return err
		}
	}

	return nil
}

// Subscribe 便捷订阅.
func (c *Client) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	if c == nil || c.Subscriber == nil {
		return nil, fmt.Errorf("mq subscriber not initialized")
	}
	// 调整调用以匹配签名：只传递 ctx 和 topic
	ch, err := c.Subscriber.Subscribe(ctx, topic)
	if err != nil {
		return nil, err
	}

	return ch, nil
}

// Close 关闭资源.
func (c *Client) Close() error {
	var err error

	if c.Publisher != nil {
		if e := c.Publisher.Close(); e != nil {
			err = e
		}
	}

	if c.Subscriber != nil {
		if e := c.Subscriber.Close(); e != nil {
			err = e
		}
	}

	return err
}

var (
	mqOnce sync.Once
	mqInst *Client
	mqErr  error
)

// New 初始化消息队列（单例）.
func New(ctx context.Context) (*Client, error) {
	mqOnce.Do(func() {
		cfg := configs.GetConfig().MQ

		factory, ok := factories[cfg.Type]
		if !ok {
			mqErr = fmt.Errorf("unsupported mq type: %s", cfg.Type)
			return
		}

		logger := &zerologAdapter{l: nlog.Logger()}

		pub, sub, err := factory(ctx, &cfg, logger)
		if err != nil {
			mqErr = fmt.Errorf("init mq (%s): %w", cfg.Type, err)
			return
		}

		mqInst = &Client{Publisher: pub, Subscriber: sub}

		nlog.Logger().Info().Str("type", string(cfg.Type)).Msg("MQ 管理器已初始化")
	})

	return mqInst, mqErr
}
