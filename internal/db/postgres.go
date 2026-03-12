package db

import (
	"context"
	"fmt"
	"time"

	"apartment-backend/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func NewPostgresPool(cfg config.DBConfig, logger *zap.Logger) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("parse db config: %w", err)
	}

	poolCfg.MaxConns = int32(cfg.MaxOpenConns)
	poolCfg.MinConns = int32(cfg.MaxIdleConns)
	poolCfg.MaxConnLifetime = 30 * time.Minute
	poolCfg.MaxConnIdleTime = 5 * time.Minute

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	logger.Info("Connected to PostgreSQL",
		zap.String("host", cfg.Host),
		zap.String("database", cfg.Name),
	)

	return pool, nil
}
