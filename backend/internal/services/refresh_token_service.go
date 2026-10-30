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
	Issue(ctx context.Context, userID uuid.UUID, credentialVersion string) (string, error)
	Rotate(ctx context.Context, rawToken string) (string, *db.ClaimctlUser, error)
	Revoke(ctx context.Context, rawToken string) error
	RevokeAllByUser(ctx context.Context, userID uuid.UUID) error
}

type refreshTokenService struct {
	store db.Store
}

func NewRefreshTokenService(store db.Store) RefreshTokenService {
	return &refreshTokenService{store: store}
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
func (s *refreshTokenService) Issue(ctx context.Context, userID uuid.UUID, credentialVersion string) (string, error) {
	raw, err := generateRawToken()
	if err != nil {
		return "", err
	}
	err = s.store.ExecTx(ctx, func(q db.Querier) error {
		user, err := q.FindUserByIdForUpdate(ctx, userID)
		if err != nil {
			return fmt.Errorf("failed to lock refresh account: %w", err)
		}
		if user.Status != "active" || credentialVersion == "" || CredentialVersion(user.Password) != credentialVersion {
			return ErrAccessDenied
		}
		_, err = q.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{UserID: userID, TokenHash: hashToken(raw), FamilyID: uuid.New(), ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(7 * 24 * time.Hour), Valid: true}})
		return err
	})
	if err != nil {
		return "", fmt.Errorf("failed to issue refresh token: %w", err)
	}
	return raw, nil
}

// Rotate validates the old token, detects theft, and issues a new one.
// Returns the locked account snapshot used to authorize the new credentials.
func (s *refreshTokenService) Rotate(ctx context.Context, rawToken string) (string, *db.ClaimctlUser, error) {
	hash := hashToken(rawToken)
	initial, err := s.store.GetRefreshTokenByHash(ctx, hash)
	if isNotFound(err) {
		return "", nil, ErrRefreshTokenNotFound
	}
	if err != nil {
		return "", nil, fmt.Errorf("failed to find refresh token: %w", err)
	}
	var newRaw string
	var user db.ClaimctlUser
	var validationErr error
	err = s.store.ExecTx(ctx, func(q db.Querier) error {
		// Lock the account before tokens, matching password update lock ordering.
		var err error
		user, err = q.FindUserByIdForUpdate(ctx, initial.UserID)
		if err != nil {
			return fmt.Errorf("failed to lock refresh account: %w", err)
		}
		if user.Status != "active" {
			return ErrAccessDenied
		}
		record, err := q.GetRefreshTokenByHashForUpdate(ctx, hash)
		if isNotFound(err) {
			return ErrRefreshTokenNotFound
		}
		if err != nil {
			return err
		}
		if record.UserID != user.ID {
			return ErrAccessDenied
		}
		if record.Revoked {
			if err := q.RevokeRefreshTokenFamily(ctx, record.FamilyID); err != nil {
				return err
			}
			validationErr = ErrRefreshTokenRevoked
			return nil
		}
		if !record.ExpiresAt.Valid || !record.ExpiresAt.Time.After(time.Now()) {
			return ErrRefreshTokenExpired
		}
		newRaw, err = generateRawToken()
		if err != nil {
			return err
		}
		if err := q.RevokeRefreshToken(ctx, hash); err != nil {
			return err
		}
		_, err = q.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{UserID: user.ID, TokenHash: hashToken(newRaw), FamilyID: record.FamilyID, ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(7 * 24 * time.Hour), Valid: true}})
		return err
	})
	if err != nil {
		return "", nil, err
	}
	if validationErr != nil {
		return "", nil, validationErr
	}
	return newRaw, &user, nil
}

// Revoke marks the token's family as revoked (used on logout).
func (s *refreshTokenService) Revoke(ctx context.Context, rawToken string) error {
	record, err := s.store.GetRefreshTokenByHash(ctx, hashToken(rawToken))
	if isNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return s.store.ExecTx(ctx, func(q db.Querier) error {
		if _, err := q.FindUserByIdForUpdate(ctx, record.UserID); err != nil {
			return err
		}
		return q.RevokeRefreshTokenFamily(ctx, record.FamilyID)
	})
}

func (s *refreshTokenService) RevokeAllByUser(ctx context.Context, userID uuid.UUID) error {
	return s.store.ExecTx(ctx, func(q db.Querier) error {
		if _, err := q.FindUserByIdForUpdate(ctx, userID); err != nil {
			return err
		}
		return q.RevokeAllUserRefreshTokens(ctx, userID)
	})
}
