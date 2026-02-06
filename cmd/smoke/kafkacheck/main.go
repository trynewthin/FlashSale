// kafkacheck 用于校验 Kafka 生产与消费链路。
package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"flashsale/pkg/base/config"
	"flashsale/pkg/base/kafkax"
	"github.com/google/uuid"
)

// main 执行 Kafka smoke 检查：启动消费协程、发送消息并等待回执。
func main() {
	configPath := flag.String("config", "configs/local/dev.yaml", "config file path")
	timeout := flag.Duration("timeout", 20*time.Second, "consume timeout")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		panic(err)
	}

	producer, err := kafkax.NewProducer(cfg.Kafka)
	if err != nil {
		panic(err)
	}
	defer producer.Close()

	consumer, err := kafkax.NewConsumer(cfg.Kafka)
	if err != nil {
		panic(err)
	}

	topic := cfg.Kafka.Topics.OrderCreate
	if topic == "" {
		topic = "order.create"
	}
	msgKey := uuid.NewString()
	group := fmt.Sprintf("smoke-%d", time.Now().UnixNano())

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	received := make(chan struct{}, 1)
	errCh := make(chan error, 1)

	go func() {
		err := consumer.Consume(ctx, group, []string{topic}, func(ctx context.Context, msg kafkax.Message) error {
			if string(msg.Key) == msgKey {
				received <- struct{}{}
				cancel()
			}
			return nil
		})
		if err != nil && ctx.Err() == nil {
			errCh <- err
		}
	}()

	time.Sleep(500 * time.Millisecond)
	headers := map[string]string{kafkax.IdempotencyHeader: uuid.NewString()}
	if err := producer.Publish(ctx, topic, []byte(msgKey), []byte("smoke"), headers); err != nil {
		panic(err)
	}

	select {
	case <-received:
		fmt.Println("kafka ok")
	case err := <-errCh:
		panic(err)
	case <-ctx.Done():
		panic(fmt.Errorf("kafka consume timeout: %w", ctx.Err()))
	}
}
