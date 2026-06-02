package events

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/segmentio/kafka-go"
)

type KafkaPublisher struct {
	writer *kafka.Writer
	logger *slog.Logger
}

func NewKafkaPublisher(brokers []string, logger *slog.Logger) *KafkaPublisher {
	if logger == nil {
		logger = slog.Default()
	}
	return &KafkaPublisher{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			RequiredAcks: kafka.RequireAll,
			Async:        false,
		},
		logger: logger,
	}
}

func (p *KafkaPublisher) Publish(ctx context.Context, topic string, key string, payload []byte) error {
	msg := kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: payload,
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		p.logger.Error("Failed to publish message to Kafka", "error", err, "topic", topic, "key", key)
		return fmt.Errorf("failed to publish message to Kafka: %w", err)
	}

	p.logger.Info("kafka message published",
		"topic", topic,
		"key", key,
	)

	return nil
}

func (p *KafkaPublisher) Close() error {
	if p == nil || p.writer == nil {
		return nil
	}

	return p.writer.Close()
}
