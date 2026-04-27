package messaging

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/real-ass/shared/go-common/logger"
	"go.uber.org/zap"
)

// KafkaConfig holds Kafka connection configuration.
type KafkaConfig struct {
	Brokers        []string      `yaml:"brokers" json:"brokers"`
	Topic          string        `yaml:"topic" json:"topic"`
	ConsumerGroup  string        `yaml:"consumer_group" json:"consumer_group"`
	ClientID       string        `yaml:"client_id" json:"client_id"`
	MaxAttempts    int           `yaml:"max_attempts" json:"max_attempts"`
	BatchSize      int           `yaml:"batch_size" json:"batch_size"`
	BatchTimeout   time.Duration `yaml:"batch_timeout" json:"batch_timeout"`
	ReadTimeout    time.Duration `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout   time.Duration `yaml:"write_timeout" json:"write_timeout"`
	MinBytes       int           `yaml:"min_bytes" json:"min_bytes"`
	MaxBytes       int           `yaml:"max_bytes" json:"max_bytes"`
}

// DefaultKafkaConfig returns a KafkaConfig with sensible defaults.
func DefaultKafkaConfig() KafkaConfig {
	return KafkaConfig{
		Brokers:        []string{"localhost:9092"},
		Topic:          "real-ass-events",
		ConsumerGroup:  "real-ass-consumer",
		ClientID:       "real-ass",
		MaxAttempts:    3,
		BatchSize:      100,
		BatchTimeout:   time.Second,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MinBytes:       10,
		MaxBytes:       10e6,
	}
}

// Producer publishes messages to Kafka.
type Producer struct {
	writer *kafka.Writer
	config KafkaConfig
}

// NewProducer creates a new Kafka producer.
func NewProducer(cfg KafkaConfig) *Producer {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Topic:        cfg.Topic,
		Balancer:     &kafka.LeastBytes{},
		MaxAttempts:  cfg.MaxAttempts,
		BatchSize:    cfg.BatchSize,
		BatchTimeout: cfg.BatchTimeout,
		WriteTimeout: cfg.WriteTimeout,
		Async:        false,
		Completion: func(messages []kafka.Message, err error) {
			if err != nil {
				logger.Error("kafka write completion error", zap.Error(err))
			}
		},
	}

	producer := &Producer{
		writer: writer,
		config: cfg,
	}

	logger.Info("Kafka producer initialized",
		zap.Strings("brokers", cfg.Brokers),
		zap.String("topic", cfg.Topic),
	)

	return producer
}

// Produce sends a message to Kafka with the given key and value.
func (p *Producer) Produce(ctx context.Context, key string, value []byte) error {
	msg := kafka.Message{
		Key:   []byte(key),
		Value: value,
		Time:  time.Now(),
	}

	return p.writer.WriteMessages(ctx, msg)
}

// ProduceWithHeaders sends a message with custom headers.
func (p *Producer) ProduceWithHeaders(ctx context.Context, key string, value []byte, headers map[string]string) error {
	var kafkaHeaders []kafka.Header
	for k, v := range headers {
		kafkaHeaders = append(kafkaHeaders, kafka.Header{Key: k, Value: []byte(v)})
	}

	msg := kafka.Message{
		Key:     []byte(key),
		Value:   value,
		Time:    time.Now(),
		Headers: kafkaHeaders,
	}

	return p.writer.WriteMessages(ctx, msg)
}

// ProduceBatch sends multiple messages in a single batch.
func (p *Producer) ProduceBatch(ctx context.Context, messages []kafka.Message) error {
	return p.writer.WriteMessages(ctx, messages...)
}

// Close closes the producer.
func (p *Producer) Close() error {
	if err := p.writer.Close(); err != nil {
		return fmt.Errorf("failed to close Kafka producer: %w", err)
	}
	logger.Info("Kafka producer closed")
	return nil
}

// Consumer consumes messages from Kafka.
type Consumer struct {
	reader *kafka.Reader
	config KafkaConfig
}

// NewConsumer creates a new Kafka consumer.
func NewConsumer(cfg KafkaConfig) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Brokers,
		Topic:          cfg.Topic,
		GroupID:        cfg.ConsumerGroup,
		MinBytes:       cfg.MinBytes,
		MaxBytes:       cfg.MaxBytes,
		ReadTimeout:    cfg.ReadTimeout,
		MaxWait:        cfg.ReadTimeout,
		CommitInterval: time.Second,
		StartOffset:    kafka.FirstOffset,
	})

	consumer := &Consumer{
		reader: reader,
		config: cfg,
	}

	logger.Info("Kafka consumer initialized",
		zap.Strings("brokers", cfg.Brokers),
		zap.String("topic", cfg.Topic),
		zap.String("group", cfg.ConsumerGroup),
	)

	return consumer
}

// KafkaMessage represents a consumed Kafka message.
type KafkaMessage struct {
	Key       string
	Value     []byte
	Topic     string
	Partition int
	Offset    int64
	Headers   map[string]string
	Time      time.Time
}

// Consume reads messages from Kafka and processes them with the handler.
func (c *Consumer) Consume(ctx context.Context, handler func(msg KafkaMessage) error) error {
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			logger.Error("kafka read error", zap.Error(err))
			continue
		}

		kafkaMsg := KafkaMessage{
			Key:       string(msg.Key),
			Value:     msg.Value,
			Topic:     msg.Topic,
			Partition: msg.Partition,
			Offset:    msg.Offset,
			Time:      msg.Time,
			Headers:   make(map[string]string),
		}

		for _, h := range msg.Headers {
			kafkaMsg.Headers[h.Key] = string(h.Value)
		}

		if err := handler(kafkaMsg); err != nil {
			logger.Error("kafka message handler error",
				zap.Error(err),
				zap.String("topic", kafkaMsg.Topic),
				zap.Int("partition", kafkaMsg.Partition),
				zap.Int64("offset", kafkaMsg.Offset),
			)
		}
	}
}

// Close closes the consumer.
func (c *Consumer) Close() error {
	if err := c.reader.Close(); err != nil {
		return fmt.Errorf("failed to close Kafka consumer: %w", err)
	}
	logger.Info("Kafka consumer closed")
	return nil
}
