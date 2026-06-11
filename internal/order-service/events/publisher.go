package events

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"
)

type EventPublisher interface {
	Publish(ctx context.Context, topic, key string, payload []byte) error
}

type OutboxRepository interface {
	ClaimUnpublished(ctx context.Context, limit int32, lockedBy string, reclaimBefore time.Time) ([]OutboxMessage, error)
	MarkPublished(ctx context.Context, id string, lockedBy string) error
	MarkFailed(ctx context.Context, id string, lockedBy string, publishErr error) error
}

type OutboxPublisher struct {
	repository  OutboxRepository
	publisher   EventPublisher
	logger      *slog.Logger
	batchSize   int32
	interval    time.Duration
	lockTimeout time.Duration
	workerID    string
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
		repository:  repository,
		publisher:   publisher,
		logger:      logger,
		batchSize:   batchSize,
		interval:    interval,
		lockTimeout: 30 * time.Second,
		workerID:    newWorkerID(),
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
	reclaimBefore := time.Now().UTC().Add(-p.lockTimeout)

	events, err := p.repository.ClaimUnpublished(ctx, p.batchSize, p.workerID, reclaimBefore)
	if err != nil {
		return err
	}

	for _, event := range events {
		err := p.publisher.Publish(ctx, event.Topic, event.Key, event.Payload)
		if err != nil {
			p.logger.Error("outbox publish failed",
				"event_id", event.ID,
				"event_type", event.EventType,
				"topic", event.Topic,
				"key", event.Key,
				"error", err,
			)
			_ = p.repository.MarkFailed(ctx, event.ID, p.workerID, err)
			continue
		}
		if err := p.repository.MarkPublished(ctx, event.ID, p.workerID); err != nil {
			return err
		}
		p.logger.Info("outbox event published",
			"event_id", event.ID,
			"event_type", event.EventType,
			"topic", event.Topic,
			"key", event.Key,
		)
	}

	return nil
}

func newWorkerID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "outbox-publisher-" + time.Now().UTC().Format("20060102150405.000000000")
	}
	return "outbox-publisher-" + hex.EncodeToString(raw[:])
}
