package kafkaprobe

import (
	"context"
	"fmt"
	"time"

	"flashsale/pkg/base/config"
	"flashsale/pkg/base/kafkax"
	"github.com/google/uuid"
)

func Run(configPath string, timeout time.Duration) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	producer, err := kafkax.NewProducer(cfg.Kafka)
	if err != nil {
		return err
	}
	defer producer.Close()

	consumer, err := kafkax.NewConsumer(cfg.Kafka)
	if err != nil {
		return err
	}

	topic := cfg.Kafka.Topics.OrderCreate
	if topic == "" {
		topic = "order.create"
	}
	msgKey := uuid.NewString()
	group := fmt.Sprintf("smoke-%d", time.Now().UnixNano())

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
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
		return err
	}

	select {
	case <-received:
		fmt.Println("kafka ok")
		return nil
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return fmt.Errorf("kafka consume timeout: %w", ctx.Err())
	}
}
