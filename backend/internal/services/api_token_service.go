package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/thetaqitahmid/claimctl/internal/db"
)

type APITokenService interface {
	GenerateToken(ctx context.Context, userID uuid.UUID, name string, expiresAt *time.Time) (string, *db.ClaimctlApiToken, error)
	ListTokens(ctx context.Context, userID uuid.UUID) ([]db.ClaimctlApiToken, error)
	RevokeToken(ctx context.Context, tokenID string, userID uuid.UUID) error
	ValidateToken(ctx context.Context, token string) (*db.ClaimctlUser, error)
}

type apiTokenService struct {
	db db.Querier
}

func NewAPITokenService(db db.Querier) APITokenService {
	return &apiTokenService{db: db}
}

func (s *apiTokenService) GenerateToken(ctx context.Context, userID uuid.UUID, name string, expiresAt *time.Time) (string, *db.ClaimctlApiToken, error) {
	// 0. Ensure unique name
	existingTokens, err := s.ListTokens(ctx, userID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to check existing tokens: %w", err)
	}

	uniqueName := name
	counter := 1
	for {
		exists := false
		for _, t := range existingTokens {
			if t.Name == uniqueName {
				exists = true
				break
			}
		}
		if !exists {
			break
		}
		uniqueName = fmt.Sprintf("%s-%d", name, counter)
		counter++
	}

	// 1. Generate random token
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}
	rawToken := hex.EncodeToString(bytes)
	fullToken := "res_" + rawToken

	// 2. Hash the token
	hash := sha256.Sum256([]byte(fullToken))
	tokenHash := hex.EncodeToString(hash[:])

	// 3. Store in DB
	var pgExpiresAt pgtype.Timestamptz
	if expiresAt != nil {
		pgExpiresAt = pgtype.Timestamptz{Time: *expiresAt, Valid: true}
	} else {
		pgExpiresAt = pgtype.Timestamptz{Valid: false}
	}

	token, err := s.db.CreateAPIToken(ctx, db.CreateAPITokenParams{
		UserID:    userID,
		Name:      uniqueName,
		TokenHash: tokenHash,
		ExpiresAt: pgExpiresAt,
	})
	if err != nil {
		return "", nil, fmt.Errorf("failed to store token in DB: %w", err)
	}

	return fullToken, &token, nil
}

func (s *apiTokenService) ListTokens(ctx context.Context, userID uuid.UUID) ([]db.ClaimctlApiToken, error) {
	tokens, err := s.db.ListAPITokens(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tokens: %w", err)
	}
	return tokens, nil
}

func (s *apiTokenService) RevokeToken(ctx context.Context, tokenID string, userID uuid.UUID) error {
	id, err := uuid.Parse(tokenID)
	if err != nil {
		return fmt.Errorf("invalid token ID format: %w", err)
	}

	err = s.db.RevokeAPIToken(ctx, db.RevokeAPITokenParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}
	return nil
}

func (s *apiTokenService) ValidateToken(ctx context.Context, token string) (*db.ClaimctlUser, error) {
	// 1. Hash the token
	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])

	// 2. Find token in DB
	apiToken, err := s.db.GetAPITokenByHash(ctx, tokenHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid token")
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	// 3. Check expiration
	if apiToken.ExpiresAt.Valid && apiToken.ExpiresAt.Time.Before(time.Now()) {
		return nil, fmt.Errorf("token expired")
	}

	// 4. Update usage
	_ = s.db.UpdateAPITokenLastUsed(ctx, apiToken.ID)

	// 5. Get User
	user, err := s.db.FindUserById(ctx, apiToken.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve user for token: %w", err)
	}

	if user.Status != "active" {
		return nil, fmt.Errorf("user is inactive")
	}

	return &user, nil
}
