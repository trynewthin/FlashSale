package kafkax

import (
	"testing"
	"time"

	"flashsale/pkg/base/config"
	"github.com/segmentio/kafka-go"
)

func TestNewProducerUsesLowLatencyBatchTimeout(t *testing.T) {
	t.Parallel()

	producer, err := NewProducer(config.KafkaConfig{
		Brokers: []string{"kafka:9092"},
	})
	if err != nil {
		t.Fatalf("NewProducer returned error: %v", err)
	}
	if producer == nil || producer.writer == nil {
		t.Fatalf("producer writer was not initialized")
	}
	if producer.writer.BatchTimeout != producerBatchTimeout {
		t.Fatalf("BatchTimeout = %v, want %v", producer.writer.BatchTimeout, producerBatchTimeout)
	}
	if producer.writer.RequiredAcks != kafka.RequireOne {
		t.Fatalf("RequiredAcks = %v, want %v", producer.writer.RequiredAcks, kafka.RequireOne)
	}
	if producerBatchTimeout >= 200*time.Millisecond {
		t.Fatalf("producerBatchTimeout = %v, want < 200ms for hot-path publish deadlines", producerBatchTimeout)
	}
}
