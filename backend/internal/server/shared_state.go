package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"github.com/thetaqitahmid/claimctl/internal/db"
)

type postgresStorage struct{ q db.Querier }

func NewPostgresStorage(q db.Querier) fiber.Storage { return &postgresStorage{q: q} }

func (s *postgresStorage) Get(key string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	value, err := s.q.GetSharedValue(ctx, key)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return value, err
}
func (s *postgresStorage) Set(key string, value []byte, expiry time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.q.SetSharedValue(ctx, db.SetSharedValueParams{Key: key, Value: value, TtlSeconds: expiry.Seconds()})
}
func (s *postgresStorage) Delete(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.q.DeleteSharedValue(ctx, key)
}
func (s *postgresStorage) Reset() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.q.ResetSharedValues(ctx)
}
func (s *postgresStorage) Close() error { return nil }

func AtomicRateLimiter(q db.Querier, namespace string, max int64, window time.Duration) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		sum := sha256.Sum256([]byte(ClientIP(c)))
		row, err := q.IncrementRateLimit(ctx, db.IncrementRateLimitParams{Key: namespace + ":" + hex.EncodeToString(sum[:]), TtlSeconds: window.Seconds()})
		if err != nil {
			return fiber.ErrServiceUnavailable
		}
		if row.Count > max {
			seconds := int64(time.Until(row.ExpiresAt.Time).Seconds()) + 1
			if seconds < 1 {
				seconds = 1
			}
			c.Set("Retry-After", strconv.FormatInt(seconds, 10))
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "Too many requests"})
		}
		return c.Next()
	}
}
