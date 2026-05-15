package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, dataSourceName string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dataSourceName)
	if err != nil {
		return nil, err
	}
	err = pool.Ping(ctx)
	if err != nil {
		return nil, err
	}

	return pool, nil
}
