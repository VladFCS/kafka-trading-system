package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vladfc/kafka-trading-system/internal/market-data-service/domain"
	marketdatadb "github.com/vladfc/kafka-trading-system/internal/market-data-service/repository/sqlc"
)

type PostgresRepository struct {
	pool    *pgxpool.Pool
	queries *marketdatadb.Queries
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{
		pool:    pool,
		queries: marketdatadb.New(pool),
	}
}

func (r *PostgresRepository) GetLatestPrice(ctx context.Context, symbol string) (domain.LatestPrice, error) {
	row, err := r.queries.GetLatestPrice(ctx, symbol)
	if err != nil {
		return domain.LatestPrice{}, err
	}

	updatedAt, err := timestamptzToTime(row.UpdatedAt)
	if err != nil {
		return domain.LatestPrice{}, err
	}

	return domain.LatestPrice{
		Symbol:     row.Symbol,
		PriceCents: row.PriceCents,
		UpdatedAt:  updatedAt,
	}, nil
}

func timestamptzToTime(value pgtype.Timestamptz) (time.Time, error) {
	if !value.Valid {
		return time.Time{}, errors.New("invalid timestamptz value")
	}
	if value.InfinityModifier != 0 {
		return time.Time{}, errors.New("invalid timestamptz infinity value")
	}
	return value.Time, nil
}
