package mq

import (
	"context"
	"fmt"
	"sync"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/redis/go-redis/v9"

	"github.com/yeisme/notevault/pkg/configs"
)

const (
	// DefaultChannelBufferSize 默认通道缓冲区大小.
	DefaultChannelBufferSize = 100
)

// RedisPublisher Redis Publisher 实现.
type RedisPublisher struct {
	client *redis.Client
}

// RedisSubscriber Redis Subscriber 实现.
type RedisSubscriber struct {
	client     *redis.Client
	subscriber *redis.PubSub
	handlers   map[string]message.HandlerFunc
	mu         sync.RWMutex
	closed     bool
	closeCh    chan struct{}
}

// init 注册 Redis 工厂.
func init() {
	RegisterFactory(configs.MQTypeRedis, redisFactory)
}

// redisFactory 创建 Redis Publisher & Subscriber.
func redisFactory(
	ctx context.Context,
	config any,
	logger watermill.LoggerAdapter) (
	message.Publisher, message.Subscriber, error) {
	cfg, ok := config.(*configs.MQConfig)
	if !ok {
		return nil, nil, fmt.Errorf("invalid Redis config")
	}

	// 创建 Redis 客户端
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// 测试连接
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, nil, err
	}

	// 创建 Publisher
	pub := &RedisPublisher{
		client: rdb,
	}

	// 创建 Subscriber
	sub := &RedisSubscriber{
		client:   rdb,
		handlers: make(map[string]message.HandlerFunc),
		closeCh:  make(chan struct{}),
	}

	return pub, sub, nil
}

// Publish 实现 Publisher 接口.
func (p *RedisPublisher) Publish(topic string, msgs ...*message.Message) error {
	for _, msg := range msgs {
		data := msg.Payload

		err := p.client.Publish(context.Background(), topic, data).Err()
		if err != nil {
			return err
		}

		// 确认消息
		msg.Ack()
	}

	return nil
}

// Close 实现 Publisher 接口.
func (p *RedisPublisher) Close() error {
	return p.client.Close()
}

// Subscribe 实现 Subscriber 接口.
func (s *RedisSubscriber) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil, nil
	}

	ch := make(chan *message.Message, DefaultChannelBufferSize)

	// 订阅主题
	s.subscriber = s.client.Subscribe(ctx, topic)

	// 启动 goroutine 处理消息
	go func() {
		defer close(ch)

		for {
			select {
			case <-s.closeCh:
				return
			case <-ctx.Done():
				return
			default:
				msg, err := s.subscriber.ReceiveMessage(ctx)
				if err != nil {
					return
				}

				// 创建 Watermill 消息
				wmMsg := message.NewMessage(watermill.NewUUID(), []byte(msg.Payload))

				select {
				case ch <- wmMsg:
				case <-s.closeCh:
					return
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return ch, nil
}

// Close 实现 Subscriber 接口.
func (s *RedisSubscriber) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	close(s.closeCh)

	if s.subscriber != nil {
		if err := s.subscriber.Close(); err != nil {
			// 记录错误但不中断关闭过程
			// 这里可以添加日志记录
		}
	}

	return s.client.Close()
}
