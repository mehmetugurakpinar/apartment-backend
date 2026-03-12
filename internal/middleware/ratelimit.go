package middleware

import (
	"context"
	"fmt"
	"time"

	"apartment-backend/internal/config"
	"apartment-backend/internal/models"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

func RateLimiter(rdb *redis.Client, cfg config.RateLimitConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ip := c.IP()
		key := fmt.Sprintf("ratelimit:%s", ip)

		ctx := context.Background()

		count, err := rdb.Incr(ctx, key).Result()
		if err != nil {
			// If Redis is down, allow the request
			return c.Next()
		}

		if count == 1 {
			rdb.Expire(ctx, key, cfg.Window)
		}

		ttl, _ := rdb.TTL(ctx, key).Result()

		c.Set("X-RateLimit-Limit", fmt.Sprintf("%d", cfg.Max))
		c.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", max(0, cfg.Max-int(count))))
		c.Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(ttl).Unix()))

		if int(count) > cfg.Max {
			return c.Status(fiber.StatusTooManyRequests).JSON(models.ErrorResponse("Rate limit exceeded. Try again later."))
		}

		return c.Next()
	}
}
