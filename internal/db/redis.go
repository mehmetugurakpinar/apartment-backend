package db

import (
	"context"
	"fmt"
	"time"

	"apartment-backend/internal/config"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func NewRedisClient(cfg config.RedisConfig, logger *zap.Logger) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	logger.Info("Connected to Redis", zap.String("addr", fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)))

	return client, nil
}
