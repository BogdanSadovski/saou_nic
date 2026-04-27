package messaging

import (
	"context"
	"fmt"
	"sync"
	"time"

	amqp "github.com/streadway/amqp"
	"github.com/real-ass/shared/go-common/logger"
	"go.uber.org/zap"
)

// RabbitMQConfig holds RabbitMQ connection configuration.
type RabbitMQConfig struct {
	Host         string        `yaml:"host" json:"host"`
	Port         int           `yaml:"port" json:"port"`
	User         string        `yaml:"user" json:"user"`
	Password     string        `yaml:"password" json:"password"`
	VHost        string        `yaml:"vhost" json:"vhost"`
	Exchange     string        `yaml:"exchange" json:"exchange"`
	ExchangeType string        `yaml:"exchange_type" json:"exchange_type"`
	Durable      bool          `yaml:"durable" json:"durable"`
	AutoDelete   bool          `yaml:"auto_delete" json:"auto_delete"`
	ReconnectDelay time.Duration `yaml:"reconnect_delay" json:"reconnect_delay"`
}

// DefaultRabbitMQConfig returns a RabbitMQConfig with sensible defaults.
func DefaultRabbitMQConfig() RabbitMQConfig {
	return RabbitMQConfig{
		Host:           "localhost",
		Port:           5672,
		User:           "guest",
		Password:       "guest",
		VHost:          "/",
		Exchange:       "real_ass",
		ExchangeType:   "topic",
		Durable:        true,
		AutoDelete:     false,
		ReconnectDelay: 5 * time.Second,
	}
}

// DSN returns the RabbitMQ connection string.
func (c RabbitMQConfig) DSN() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%d%s",
		c.User, c.Password, c.Host, c.Port, c.VHost,
	)
}

// Publisher publishes messages to RabbitMQ.
type Publisher struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	config  RabbitMQConfig
	mu      sync.Mutex
}

// NewPublisher creates a new RabbitMQ publisher.
func NewPublisher(cfg RabbitMQConfig) (*Publisher, error) {
	conn, err := amqp.Dial(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	if err := ch.ExchangeDeclare(
		cfg.Exchange,
		cfg.ExchangeType,
		cfg.Durable,
		cfg.AutoDelete,
		false,
		false,
		nil,
	); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	pub := &Publisher{
		conn:    conn,
		channel: ch,
		config:  cfg,
	}

	logger.Info("connected to RabbitMQ (publisher)",
		zap.String("host", cfg.Host),
		zap.String("exchange", cfg.Exchange),
	)

	return pub, nil
}

// Publish publishes a message to the specified routing key.
func (p *Publisher) Publish(ctx context.Context, routingKey string, body []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.channel.Publish(
		p.config.Exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/octet-stream",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
		},
	)
}

// PublishWithHeaders publishes a message with custom headers.
func (p *Publisher) PublishWithHeaders(ctx context.Context, routingKey string, body []byte, headers map[string]interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.channel.Publish(
		p.config.Exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/octet-stream",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			Headers:      headers,
		},
	)
}

// Close closes the publisher connection.
func (p *Publisher) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.channel != nil {
		p.channel.Close()
	}
	if p.conn != nil {
		p.conn.Close()
	}
	logger.Info("RabbitMQ publisher closed")
	return nil
}

// Subscriber consumes messages from RabbitMQ.
type Subscriber struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	config   RabbitMQConfig
	queue    amqp.Queue
	mu       sync.Mutex
	closed   bool
}

// NewSubscriber creates a new RabbitMQ subscriber.
func NewSubscriber(cfg RabbitMQConfig, queueName string) (*Subscriber, error) {
	conn, err := amqp.Dial(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	if err := ch.ExchangeDeclare(
		cfg.Exchange,
		cfg.ExchangeType,
		cfg.Durable,
		cfg.AutoDelete,
		false,
		false,
		nil,
	); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	q, err := ch.QueueDeclare(
		queueName,
		cfg.Durable,
		cfg.AutoDelete,
		false,
		false,
		nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	if err := ch.QueueBind(q.Name, "#", cfg.Exchange, false, nil); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to bind queue: %w", err)
	}

	sub := &Subscriber{
		conn:    conn,
		channel: ch,
		config:  cfg,
		queue:   q,
	}

	logger.Info("connected to RabbitMQ (subscriber)",
		zap.String("host", cfg.Host),
		zap.String("queue", queueName),
	)

	return sub, nil
}

// Message represents a consumed message.
type Message struct {
	Body        []byte
	RoutingKey  string
	DeliveryTag uint64
	Headers     map[string]interface{}
	ContentType string
	Timestamp   time.Time
}

// Consume starts consuming messages from the queue.
func (s *Subscriber) Consume(ctx context.Context, handler func(msg Message) error) error {
	msgs, err := s.channel.Consume(s.queue.Name, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case d, ok := <-msgs:
			if !ok {
				return fmt.Errorf("channel closed")
			}

			msg := Message{
				Body:        d.Body,
				RoutingKey:  d.RoutingKey,
				DeliveryTag: d.DeliveryTag,
				Headers:     d.Headers,
				ContentType: d.ContentType,
				Timestamp:   d.Timestamp,
			}

			if err := handler(msg); err != nil {
				logger.Error("message handler error",
					zap.Error(err),
					zap.String("routing_key", msg.RoutingKey),
				)
				d.Nack(false, true)
			} else {
				d.Ack(false)
			}
		}
	}
}

// Close closes the subscriber connection.
func (s *Subscriber) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true

	if s.channel != nil {
		s.channel.Close()
	}
	if s.conn != nil {
		s.conn.Close()
	}
	logger.Info("RabbitMQ subscriber closed")
	return nil
}
