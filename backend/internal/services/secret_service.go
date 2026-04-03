package services

import (
	"context"

	"github.com/google/uuid"

	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/utils"
)

type SecretService struct {
	q             db.Querier
	encryptionKey string
}

func NewSecretService(q db.Querier, encryptionKey string) *SecretService {
	return &SecretService{q: q, encryptionKey: encryptionKey}
}

func (s *SecretService) CreateSecret(ctx context.Context, key, value, description string) (db.ClaimctlSecret, error) {
	encryptedValue, err := utils.Encrypt(value, s.encryptionKey)
	if err != nil {
		slog.Error("failed to encrypt secret value", "error", err)
		return db.ClaimctlSecret{}, err
	}
	valueToStore := "ENC:" + encryptedValue

	return s.q.CreateSecret(ctx, db.CreateSecretParams{
		Key:         key,
		Value:       valueToStore,
		Description: pgtype.Text{String: description, Valid: description != ""},
	})
}

func (s *SecretService) decryptSecret(secret db.ClaimctlSecret) db.ClaimctlSecret {
	if strings.HasPrefix(secret.Value, "ENC:") {
		decrypted, err := utils.Decrypt(strings.TrimPrefix(secret.Value, "ENC:"), s.encryptionKey)
		if err == nil {
			secret.Value = decrypted
		} else {
			slog.Warn("Failed to decrypt secret value", "key", secret.Key, "error", err)
		}
	}
	return secret
}

func (s *SecretService) GetSecret(ctx context.Context, id uuid.UUID) (db.ClaimctlSecret, error) {
	secret, err := s.q.GetSecret(ctx, id)
	if err != nil {
		return secret, err
	}
	return s.decryptSecret(secret), nil
}

func (s *SecretService) GetSecretByKey(ctx context.Context, key string) (db.ClaimctlSecret, error) {
	secret, err := s.q.GetSecretByKey(ctx, key)
	if err != nil {
		return secret, err
	}
	return s.decryptSecret(secret), nil
}

func (s *SecretService) ListSecrets(ctx context.Context) ([]db.ClaimctlSecret, error) {
	secrets, err := s.q.ListSecrets(ctx)
	if err != nil {
		return nil, err
	}
	for i, secret := range secrets {
		secrets[i] = s.decryptSecret(secret)
	}
	return secrets, nil
}

func (s *SecretService) UpdateSecret(ctx context.Context, id uuid.UUID, value, description string) (db.ClaimctlSecret, error) {
	encryptedValue, err := utils.Encrypt(value, s.encryptionKey)
	if err != nil {
		slog.Error("failed to encrypt secret value", "error", err)
		return db.ClaimctlSecret{}, err
	}
	valueToStore := "ENC:" + encryptedValue

	return s.q.UpdateSecret(ctx, db.UpdateSecretParams{
		ID:          id,
		Value:       valueToStore,
		Description: pgtype.Text{String: description, Valid: description != ""},
	})
}

func (s *SecretService) DeleteSecret(ctx context.Context, id uuid.UUID) error {
	return s.q.DeleteSecret(ctx, id)
}
