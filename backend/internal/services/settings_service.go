package services

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/utils"
)

// SettingsService handles retrieval and updates of application settings
// with a fallback mechanism to environment variables.
type SettingsService struct {
	queries       db.Querier
	encryptionKey string
}

func NewSettingsService(queries db.Querier, encryptionKey string) *SettingsService {
	return &SettingsService{
		queries:       queries,
		encryptionKey: encryptionKey,
	}
}

// GetOrGenerateJWTKeys retrieves JWT keys from settings (DB or Env) or generates/stores new ones.
func (s *SettingsService) GetOrGenerateJWTKeys(ctx context.Context) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privKeyPEM := s.GetString(ctx, "jwt_private_key")
	pubKeyPEM := s.GetString(ctx, "jwt_public_key")

	// 1. Try to load existing keys
	if privKeyPEM != "" && pubKeyPEM != "" {
		privateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(privKeyPEM))
		if err != nil {
			slog.Error("Failed to parse JWT private key from settings", "error", err)
			// Don't return error yet, might want to regenerate?
			// For now, let's treat it as a hard failure because existing keys are corrupt.
			return nil, nil, fmt.Errorf("failed to parse JWT_PRIVATE_KEY: %w", err)
		}

		publicKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(pubKeyPEM))
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse JWT_PUBLIC_KEY: %w", err)
		}

		slog.Info("Loaded JWT keys from settings")
		return privateKey, publicKey, nil
	}

	// 2. Generate new keys
	slog.Info("JWT keys not found in settings, generating new RSA key pair...")
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate RSA key pair: %w", err)
	}
	publicKey := &privateKey.PublicKey

	// 3. Serialize to PEM
	privPEMBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}
	privPEMBytes := pem.EncodeToMemory(privPEMBlock)

	pubASN1, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal public key: %w", err)
	}
	pubPEMBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubASN1,
	}
	pubPEMBytes := pem.EncodeToMemory(pubPEMBlock)

	// 4. Store in Database
	_, err = s.Set(ctx, "jwt_private_key", string(privPEMBytes), "auth", "JWT Private Key", true)
	if err != nil {
		slog.Warn("Failed to persist JWT private key to DB", "error", err)
	}

	_, err = s.Set(ctx, "jwt_public_key", string(pubPEMBytes), "auth", "JWT Public Key", false)
	if err != nil {
		slog.Warn("Failed to persist JWT public key to DB", "error", err)
	}

	slog.Info("Generated and saved new JWT keys to settings")
	return privateKey, publicKey, nil
}

// GetSettings returns all settings from the database.
func (s *SettingsService) GetSettings(ctx context.Context) ([]db.AppSetting, error) {
	return s.queries.GetSettings(ctx)
}

func (s *SettingsService) GetString(ctx context.Context, key string) string {
	// Try DB
	setting, err := s.queries.GetSetting(ctx, key)
	if err == nil {
		if setting.Value != "" {
			if setting.IsSecret {
				decrypted, err := utils.Decrypt(setting.Value, s.encryptionKey)
				if err != nil {
					slog.Error("failed to decrypt setting", "key", key, "error", err)
					return ""
				}
				return decrypted
			}
			return setting.Value
		}
	} else if err != sql.ErrNoRows {
		slog.Error("failed to get setting from db", "key", key, "error", err)
	}

	// Fallback to Env
	envKey := strings.ToUpper(key)
	envVal := os.Getenv(envKey)
	if envVal != "" {
		return envVal
	}

	return ""
}

// Set updates or inserts a setting value.
func (s *SettingsService) Set(ctx context.Context, key, value, category, description string, isSecret bool) (db.AppSetting, error) {
	if isSecret {
		encrypted, err := utils.Encrypt(value, s.encryptionKey)
		if err != nil {
			return db.AppSetting{}, fmt.Errorf("failed to encrypt setting: %w", err)
		}
		value = encrypted
	}

	params := db.UpsertSettingParams{
		Key:         key,
		Value:       value,
		Category:    category,
		Description: pgtype.Text{String: description, Valid: description != ""},
		IsSecret:    isSecret,
	}
	return s.queries.UpsertSetting(ctx, params)
}

// SyncEnvToDB iterates through all settings in the database.
// If a setting has an empty value in the DB, it checks for a corresponding
// environment variable and updates the DB if one exists.
func (s *SettingsService) SyncEnvToDB(ctx context.Context) error {
	settings, err := s.GetSettings(ctx)
	if err != nil {
		return err
	}

	for _, setting := range settings {
		if setting.Value == "" {
			envKey := strings.ToUpper(setting.Key)
			envVal := os.Getenv(envKey)
			if envVal != "" {
				slog.Info("Syncing setting from Env to DB", "key", setting.Key)
				_, err := s.Set(ctx, setting.Key, envVal, setting.Category, setting.Description.String, setting.IsSecret)
				if err != nil {
					slog.Error("Failed to sync setting from Env", "key", setting.Key, "error", err)
				}
			}
		}
	}
	return nil
}

// Helper methods for specific config
func (s *SettingsService) GetSlackBotToken(ctx context.Context) string {
	return s.GetString(ctx, "slack_bot_token")
}

func (s *SettingsService) GetSMTPConfig(ctx context.Context) (host, port, user, pass, from string) {
	host = s.GetString(ctx, "smtp_host")
	port = s.GetString(ctx, "smtp_port")
	user = s.GetString(ctx, "smtp_user")
	pass = s.GetString(ctx, "smtp_pass")
	from = s.GetString(ctx, "smtp_from")
	return
}
