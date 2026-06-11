package events

import (
	"context"
	"log/slog"
	"time"
)

type EventPublisher interface {
	Publish(ctx context.Context, topic, key string, payload []byte) error
}

type OutboxRepository interface {
	LockUnpublished(ctx context.Context, limit int32) ([]OutboxMessage, error)
	MarkPublished(ctx context.Context, id string) error
	MarkFailed(ctx context.Context, id string, publishErr error) error
}

type OutboxPublisher struct {
	repository OutboxRepository
	publisher  EventPublisher
	logger     *slog.Logger
	batchSize  int32
	interval   time.Duration
}

func NewOutboxPublisher(
	repository OutboxRepository,
	publisher EventPublisher,
	logger *slog.Logger,
	batchSize int32,
	interval time.Duration,
) *OutboxPublisher {
	if logger == nil {
		logger = slog.Default()
	}
	if batchSize <= 0 {
		batchSize = 100
	}
	if interval <= 0 {
		interval = time.Second
	}

	return &OutboxPublisher{
		repository: repository,
		publisher:  publisher,
		logger:     logger,
		batchSize:  batchSize,
		interval:   interval,
	}
}

func (p *OutboxPublisher) Run(ctx context.Context) error {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		if err := p.PublishOnce(ctx); err != nil {
			p.logger.Error("outbox publish iteration failed", "error", err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (p *OutboxPublisher) PublishOnce(ctx context.Context) error {
	events, err := p.repository.LockUnpublished(ctx, p.batchSize)
	if err != nil {
		return err
	}

	for _, event := range events {
		err := p.publisher.Publish(ctx, event.Topic, event.Key, event.Payload)
		if err != nil {
			_ = p.repository.MarkFailed(ctx, event.ID, err)
			continue
		}
		if err := p.repository.MarkPublished(ctx, event.ID); err != nil {
			return err
		}
	}

	return nil
}
