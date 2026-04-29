package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/thetaqitahmid/claimctl/internal/db"
)

var ErrRefreshTokenRevoked = errors.New("refresh token revoked")
var ErrRefreshTokenExpired = errors.New("refresh token expired")
var ErrRefreshTokenNotFound = errors.New("refresh token not found")

type RefreshTokenService interface {
	Issue(ctx context.Context, userID uuid.UUID) (string, error)
	Rotate(ctx context.Context, rawToken string) (string, uuid.UUID, error)
	Revoke(ctx context.Context, rawToken string) error
	RevokeAllByUser(ctx context.Context, userID uuid.UUID) error
}

type refreshTokenService struct {
	db db.Querier
}

func NewRefreshTokenService(db db.Querier) RefreshTokenService {
	return &refreshTokenService{db: db}
}

func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

func generateRawToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return "rt_" + hex.EncodeToString(b), nil
}

// Issue creates a new refresh token for the given user.
func (s *refreshTokenService) Issue(ctx context.Context, userID uuid.UUID) (string, error) {
	raw, err := generateRawToken()
	if err != nil {
		return "", err
	}

	_, err = s.db.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		UserID:    userID,
		TokenHash: hashToken(raw),
		FamilyID:  uuid.New(),
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(7 * 24 * time.Hour), Valid: true},
	})
	if err != nil {
		return "", fmt.Errorf("failed to store refresh token: %w", err)
	}

	return raw, nil
}

// Rotate validates the old token, detects theft, and issues a new one.
// Returns (newRawToken, userID, error).
func (s *refreshTokenService) Rotate(ctx context.Context, rawToken string) (string, uuid.UUID, error) {
	record, err := s.db.GetRefreshTokenByHash(ctx, hashToken(rawToken))
	if err != nil {
		return "", uuid.Nil, ErrRefreshTokenNotFound
	}

	// Theft detection: token was already rotated but reused
	if record.Revoked {
		_ = s.db.RevokeRefreshTokenFamily(ctx, record.FamilyID)
		return "", uuid.Nil, ErrRefreshTokenRevoked
	}

	if record.ExpiresAt.Valid && record.ExpiresAt.Time.Before(time.Now()) {
		return "", uuid.Nil, ErrRefreshTokenExpired
	}

	// Delete old token
	if err := s.db.DeleteRefreshToken(ctx, record.TokenHash); err != nil {
		return "", uuid.Nil, fmt.Errorf("failed to delete old refresh token: %w", err)
	}

	// Issue new token in same family
	newRaw, err := generateRawToken()
	if err != nil {
		return "", uuid.Nil, err
	}

	_, err = s.db.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		UserID:    record.UserID,
		TokenHash: hashToken(newRaw),
		FamilyID:  record.FamilyID,
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(7 * 24 * time.Hour), Valid: true},
	})
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("failed to store new refresh token: %w", err)
	}

	return newRaw, record.UserID, nil
}

// Revoke marks the token's family as revoked (used on logout).
func (s *refreshTokenService) Revoke(ctx context.Context, rawToken string) error {
	record, err := s.db.GetRefreshTokenByHash(ctx, hashToken(rawToken))
	if err != nil {
		return nil
	}
	return s.db.RevokeRefreshTokenFamily(ctx, record.FamilyID)
}

func (s *refreshTokenService) RevokeAllByUser(ctx context.Context, userID uuid.UUID) error {
	return s.db.RevokeAllUserRefreshTokens(ctx, userID)
}
