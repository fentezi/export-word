package broker

import (
	"context"
	"errors"
	"fmt"
	"github.com/fentezi/export-word/internal/config"
	"github.com/segmentio/kafka-go"
	"log/slog"
)

type Consumer struct {
	log    *slog.Logger
	reader *kafka.Reader
}

func New(log *slog.Logger, broker config.Kafka) (Consumer, error) {
	url := fmt.Sprintf("%s:%s", broker.Address, broker.Port)
	r := kafka.NewReader(
		kafka.ReaderConfig{
			Brokers: []string{url},
			Topic:   broker.Topic,
		},
	)

	return Consumer{log: log, reader: r}, nil
}

func (c *Consumer) Consume(ctx context.Context) (<-chan []byte, error) {
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
				msg, err := c.reader.ReadMessage(ctx)
				if err == nil {
					c.log.Debug(
						"get message", slog.String("key", string(msg.Key)),
						slog.String("value", string(msg.Value)),
					)
					ch <- msg.Value
				} else {
					var kafkaErr kafka.Error
					if errors.As(err, &kafkaErr) {
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
	return c.reader.Close()
}
