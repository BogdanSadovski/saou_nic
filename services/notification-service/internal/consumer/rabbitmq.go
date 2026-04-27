package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/hr-automation/notification-service/internal/config"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

// MessageHandler defines the interface for processing notification messages.
type MessageHandler interface {
	ProcessNotification(ctx context.Context, data map[string]any) error
}

// RabbitMQConsumer manages the RabbitMQ connection and message consumption.
type RabbitMQConsumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	config  config.RabbitMQConfig
	handler MessageHandler
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

// NewRabbitMQ creates a new RabbitMQ consumer.
func NewRabbitMQ(cfg config.RabbitMQConfig, handler MessageHandler) (*RabbitMQConsumer, error) {
	conn, err := amqp.Dial(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// Declare exchange
	if err := ch.ExchangeDeclare(
		cfg.Exchange,
		"direct",
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,   // arguments
	); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	// Declare queue
	queue, err := ch.QueueDeclare(
		cfg.Queue,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	// Bind queue to exchange
	if err := ch.QueueBind(
		queue.Name,
		cfg.RoutingKey,
		cfg.Exchange,
		false,
		nil,
	); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to bind queue: %w", err)
	}

	// Set prefetch count for fair dispatch
	if err := ch.Qos(cfg.PrefetchCount, 0, false); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	return &RabbitMQConsumer{
		conn:    conn,
		channel: ch,
		config:  cfg,
		handler: handler,
		stopCh:  make(chan struct{}),
	}, nil
}

// Start begins consuming messages from the queue.
func (c *RabbitMQConsumer) Start() error {
	msgs, err := c.channel.Consume(
		c.config.Queue,
		"",    // consumer
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.consume(msgs)
	}()

	return nil
}

// Stop gracefully shuts down the consumer.
func (c *RabbitMQConsumer) Stop() {
	logrus.Info("Stopping RabbitMQ consumer...")
	close(c.stopCh)
	c.wg.Wait()

	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
	logrus.Info("RabbitMQ consumer stopped")
}

func (c *RabbitMQConsumer) consume(msgs <-chan amqp.Delivery) {
	for {
		select {
		case <-c.stopCh:
			return
		case msg, ok := <-msgs:
			if !ok {
				logrus.Warn("RabbitMQ channel closed")
				return
			}

			if err := c.handleMessage(msg); err != nil {
				logrus.WithError(err).Error("Failed to handle message")
				// Reject and requeue the message for retry
				if nackErr := msg.Nack(false, true); nackErr != nil {
					logrus.WithError(nackErr).Error("Failed to nack message")
				}
			} else {
				// Acknowledge the message
				if ackErr := msg.Ack(false); ackErr != nil {
					logrus.WithError(ackErr).Error("Failed to ack message")
				}
			}
		}
	}
}

func (c *RabbitMQConsumer) handleMessage(msg amqp.Delivery) error {
	var data map[string]any
	if err := json.Unmarshal(msg.Body, &data); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	ctx := context.Background()
	if err := c.handler.ProcessNotification(ctx, data); err != nil {
		return fmt.Errorf("failed to process notification: %w", err)
	}

	return nil
}
