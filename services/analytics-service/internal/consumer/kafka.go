package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"analytics-service/internal/config"
	"analytics-service/internal/domain"
)

// EventProcessor defines the interface for handling consumed events.
type EventProcessor interface {
	ProcessEvent(ctx context.Context, event *domain.Event) error
	ProcessBatch(ctx context.Context, events []*domain.Event) error
}

// KafkaConsumer manages Kafka topic consumption.
type KafkaConsumer struct {
	cfg       config.KafkaConfig
	processor EventProcessor
	reader    *KafkaReader
	mu        sync.Mutex
	running   bool
}

// KafkaReader abstracts reading from Kafka.
// In production, use github.com/segmentio/kafka-go or confluent-kafka-go.
type KafkaReader struct {
	brokers  []string
	topic    string
	groupID  string
	minBytes int
	maxBytes int
	maxWait  time.Duration
}

// NewKafkaReader creates a new Kafka reader.
func NewKafkaReader(brokers []string, topic, groupID string, minBytes, maxBytes int, maxWait time.Duration) *KafkaReader {
	return &KafkaReader{
		brokers:  brokers,
		topic:    topic,
		groupID:  groupID,
		minBytes: minBytes,
		maxBytes: maxBytes,
		maxWait:  maxWait,
	}
}

// NewKafkaConsumer creates a new Kafka consumer.
func NewKafkaConsumer(cfg config.KafkaConfig, processor EventProcessor) *KafkaConsumer {
	return &KafkaConsumer{
		cfg:       cfg,
		processor: processor,
	}
}

// Start begins consuming from all configured topics.
func (c *KafkaConsumer) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return fmt.Errorf("consumer is already running")
	}
	c.running = true
	c.mu.Unlock()

	log.Printf("[kafka] starting consumer for topics: %v", c.cfg.Topics)

	var wg sync.WaitGroup
	for _, topic := range c.cfg.Topics {
		wg.Add(1)
		go func(t string) {
			defer wg.Done()
			if err := c.consumeTopic(ctx, t); err != nil {
				log.Printf("[kafka] error consuming topic %s: %v", t, err)
			}
		}(topic)
	}

	wg.Wait()
	return nil
}

// Stop gracefully stops the consumer.
func (c *KafkaConsumer) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.running = false
	log.Println("[kafka] consumer stopped")
	return nil
}

// IsRunning returns whether the consumer is active.
func (c *KafkaConsumer) IsRunning() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.running
}

func (c *KafkaConsumer) consumeTopic(ctx context.Context, topic string) error {
	reader := NewKafkaReader(
		c.cfg.Brokers,
		topic,
		c.cfg.ConsumerGroup,
		c.cfg.MinBytes,
		c.cfg.MaxBytes,
		c.cfg.MaxWait,
	)

	log.Printf("[kafka] consuming topic %s with group %s", topic, c.cfg.ConsumerGroup)

	for c.IsRunning() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Read messages in batches.
		messages, err := reader.ReadBatch(ctx)
		if err != nil {
			log.Printf("[kafka] read error on %s: %v", topic, err)
			time.Sleep(time.Second)
			continue
		}

		if len(messages) == 0 {
			continue
		}

		events := make([]*domain.Event, 0, len(messages))
		for _, msg := range messages {
			var event domain.Event
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				log.Printf("[kafka] failed to unmarshal event: %v", err)
				continue
			}

			// Add metadata from Kafka message.
			if event.ID == "" {
				event.ID = fmt.Sprintf("%s-%d", msg.Topic, msg.Offset)
			}
			if event.Timestamp.IsZero() {
				event.Timestamp = msg.Time
			}

			events = append(events, &event)
		}

		if len(events) > 0 {
			if err := c.processor.ProcessBatch(ctx, events); err != nil {
				log.Printf("[kafka] failed to process batch: %v", err)
			}
		}
	}

	return nil
}

// KafkaMessage represents a message read from Kafka.
type KafkaMessage struct {
	Topic     string
	Partition int
	Offset    int64
	Key       []byte
	Value     []byte
	Time      time.Time
}

// ReadBatch reads a batch of messages from Kafka.
func (r *KafkaReader) ReadBatch(ctx context.Context) ([]KafkaMessage, error) {
	// In production, this uses kafka-go's reader.FetchMessage() or similar.
	// This is a placeholder that returns empty slice.
	// A real implementation would look like:

	// reader := kafka.NewReader(kafka.ReaderConfig{
	//     Brokers:     r.brokers,
	//     GroupID:     r.groupID,
	//     Topic:       r.topic,
	//     MinBytes:    r.minBytes,
	//     MaxBytes:    r.maxBytes,
	//     MaxWait:     r.maxWait,
	// })
	//
	// var messages []KafkaMessage
	// for {
	//     m, err := reader.FetchMessage(ctx)
	//     if err != nil {
	//         return nil, err
	//     }
	//     messages = append(messages, KafkaMessage{
	//         Topic:     m.Topic,
	//         Partition: m.Partition,
	//         Offset:    m.Offset,
	//         Key:       m.Key,
	//         Value:     m.Value,
	//         Time:      m.Time,
	//     })
	//     if len(messages) >= 100 {
	//         break
	//     }
	// }

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// Simulate no messages available.
		return nil, nil
	}
}

// CommitMessages acknowledges processed messages.
func (r *KafkaReader) CommitMessages(messages []KafkaMessage) error {
	// In production: reader.CommitMessages(ctx, messages...)
	return nil
}

// Close releases resources used by the reader.
func (r *KafkaReader) Close() error {
	// In production: return r.reader.Close()
	return nil
}
