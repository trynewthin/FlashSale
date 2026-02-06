// Package kafkax 封装 Kafka 生产与消费接口，统一消息模型。
package kafkax

import (
	"context"
	"fmt"
	"time"

	"flashsale/pkg/base/config"
	"github.com/segmentio/kafka-go"
)

// IdempotencyHeader 约定消息头中的幂等键字段名。
const IdempotencyHeader = "x-idempotency-key"

// Message 是项目内部统一的 Kafka 消息结构。
type Message struct {
	Topic     string
	Key       []byte
	Value     []byte
	Headers   map[string]string
	Partition int
	Offset    int64
	Time      time.Time
}

// Handler 定义消费回调处理函数。
type Handler func(ctx context.Context, msg Message) error

// Producer 定义消息生产接口。
type Producer interface {
	Publish(ctx context.Context, topic string, key, value []byte, headers map[string]string) error
}

// Consumer 定义消息消费接口。
type Consumer interface {
	Consume(ctx context.Context, group string, topics []string, handler Handler) error
}

// KafkaProducer 是 Producer 的 kafka-go 实现。
type KafkaProducer struct {
	writer *kafka.Writer
}

// KafkaConsumer 是 Consumer 的 kafka-go 实现。
type KafkaConsumer struct {
	cfg config.KafkaConfig
}

// NewProducer 创建生产者实例。
func NewProducer(cfg config.KafkaConfig) (*KafkaProducer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers are required")
	}
	w := &kafka.Writer{
		Addr:                   kafka.TCP(cfg.Brokers...),
		Balancer:               &kafka.LeastBytes{},
		AllowAutoTopicCreation: false,
		RequiredAcks:           kafka.RequireOne,
	}
	return &KafkaProducer{writer: w}, nil
}

// Publish 发送单条消息。
func (p *KafkaProducer) Publish(ctx context.Context, topic string, key, value []byte, headers map[string]string) error {
	if p == nil || p.writer == nil {
		return fmt.Errorf("kafka producer is nil")
	}
	if topic == "" {
		return fmt.Errorf("topic is empty")
	}
	msgHeaders := make([]kafka.Header, 0, len(headers))
	for k, v := range headers {
		msgHeaders = append(msgHeaders, kafka.Header{Key: k, Value: []byte(v)})
	}
	return p.writer.WriteMessages(ctx, kafka.Message{
		Topic:   topic,
		Key:     key,
		Value:   value,
		Headers: msgHeaders,
	})
}

// Close 关闭生产者连接。
func (p *KafkaProducer) Close() error {
	if p == nil || p.writer == nil {
		return nil
	}
	return p.writer.Close()
}

// NewConsumer 创建消费者实例。
func NewConsumer(cfg config.KafkaConfig) (*KafkaConsumer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers are required")
	}
	return &KafkaConsumer{cfg: cfg}, nil
}

// Consume 持续消费并在 handler 成功后提交位点。
//
// 关键流程说明：
// 1. 参数校验：确保消费者实例、topic、group、handler 完整。
// 2. 初始化 reader：按消费组订阅 topics，并设置同步提交模式。
// 3. 循环拉取消息：FetchMessage 获取原始消息。
// 4. 回调处理：将原始消息转内部 Message 后交给 handler。
// 5. 成功提交位点：仅当 handler 成功才 Commit，避免消息丢失。
// 6. 任一环节报错：立即返回，由上层决定重试或退出。
func (c *KafkaConsumer) Consume(ctx context.Context, group string, topics []string, handler Handler) error {
	if c == nil {
		return fmt.Errorf("kafka consumer is nil")
	}
	if handler == nil {
		return fmt.Errorf("handler is nil")
	}
	if len(topics) == 0 {
		return fmt.Errorf("topics are required")
	}
	if group == "" {
		group = c.cfg.GroupID
	}
	if group == "" {
		return fmt.Errorf("group id is required")
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        c.cfg.Brokers,
		GroupID:        group,
		GroupTopics:    topics,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: 0,
	})
	defer reader.Close()

	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			return err
		}
		if err := handler(ctx, convertMessage(msg)); err != nil {
			return err
		}
		if err := reader.CommitMessages(ctx, msg); err != nil {
			return err
		}
	}
}

// convertMessage 把 kafka-go 消息转换成内部统一消息结构。
func convertMessage(msg kafka.Message) Message {
	headers := make(map[string]string, len(msg.Headers))
	for _, h := range msg.Headers {
		headers[h.Key] = string(h.Value)
	}
	return Message{
		Topic:     msg.Topic,
		Key:       msg.Key,
		Value:     msg.Value,
		Headers:   headers,
		Partition: msg.Partition,
		Offset:    msg.Offset,
		Time:      msg.Time,
	}
}
