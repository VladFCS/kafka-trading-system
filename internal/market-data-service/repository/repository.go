package repository

import (
	"context"

	"github.com/vladfc/kafka-trading-system/internal/market-data-service/domain"
)

type MarketDataRepository interface {
	GetLatestPrice(ctx context.Context, symbol string) (domain.LatestPrice, error)
}
