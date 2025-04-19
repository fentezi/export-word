package kafka

import (
	"context"
	"errors"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/fentezi/export-word/internal/config"
	"log/slog"
	"time"
)

type Consumer struct {
	log      *slog.Logger
	consumer *kafka.Consumer
}

func New(log *slog.Logger, broker config.Kafka) (Consumer, error) {
	url := fmt.Sprintf("%s:%s", broker.Address, broker.Port)
	conf := &kafka.ConfigMap{
		"bootstrap.servers": url,
		"group.id":          "group1",
		"auto.offset.reset": "earliest",
	}
	c, err := kafka.NewConsumer(conf)
	if err != nil {
		return Consumer{}, fmt.Errorf("failed to create kafka consumer: %w", err)
	}

	return Consumer{log: log, consumer: c}, nil
}

func (c *Consumer) Consume(ctx context.Context, topic string) (<-chan []byte, error) {
	c.log.Info("kafka subscribe", slog.String("topic", topic))
	err := c.consumer.SubscribeTopics([]string{topic}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to topic %s: %w", topic, err)
	}
	ch := make(chan []byte)

	go func() {
		defer close(ch)
		c.log.Info("start consume messages")
		for {
			select {
			case <-ctx.Done():
				c.log.Info("stop consume messages")
				return
			default:
				msg, err := c.consumer.ReadMessage(time.Second)
				if err == nil {
					c.log.Debug(
						"get message", slog.String("key", string(msg.Key)),
						slog.String("value", string(msg.Value)),
					)
					ch <- msg.Value
				} else {
					var kafkaErr kafka.Error
					if errors.As(err, &kafkaErr) && !kafkaErr.IsTimeout() {
						c.log.Error(
							"consumer error", slog.String("error", err.Error()),
							slog.Any("message", msg),
						)
					}
				}

			}
		}

	}()
	return ch, nil
}

func (c *Consumer) Close() error {
	c.log.Info("closing kafka client")
	return c.consumer.Close()
}
