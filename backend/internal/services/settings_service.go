package services

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
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
	store, ok := s.queries.(db.Store)
	if !ok {
		return nil, nil, fmt.Errorf("JWT initialization requires transactional storage")
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	var privateKey *rsa.PrivateKey
	var publicKey *rsa.PublicKey
	err := store.ExecTx(ctx, func(q db.Querier) error {
		if err := q.LockJWTInitialization(ctx); err != nil {
			return err
		}
		var err error
		privateKey, publicKey, err = NewSettingsService(q, s.encryptionKey).loadOrCreateJWTKeys(ctx)
		return err
	})
	return privateKey, publicKey, err
}

func (s *SettingsService) loadOrCreateJWTKeys(ctx context.Context) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	load := func(key string) (string, error) {
		setting, err := s.queries.GetSetting(ctx, key)
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		if err != nil {
			return "", err
		}
		if setting.Value == "" {
			return "", nil
		}
		if setting.IsSecret {
			return utils.Decrypt(setting.Value, s.encryptionKey)
		}
		return setting.Value, nil
	}
	privKeyPEM, err := load("jwt_private_key")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load JWT private key: %w", err)
	}
	pubKeyPEM, err := load("jwt_public_key")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load JWT public key: %w", err)
	}
	if (privKeyPEM == "") != (pubKeyPEM == "") {
		return nil, nil, fmt.Errorf("partial JWT key pair")
	}
	fromEnvironment := privKeyPEM == ""
	if fromEnvironment {
		privKeyPEM = os.Getenv("JWT_PRIVATE_KEY")
		pubKeyPEM = os.Getenv("JWT_PUBLIC_KEY")
	}
	if (privKeyPEM == "") != (pubKeyPEM == "") {
		return nil, nil, fmt.Errorf("partial JWT environment key pair")
	}

	// 1. Try to load existing keys
	if privKeyPEM != "" && pubKeyPEM != "" {
		privateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(privKeyPEM))
		if err != nil {
			slog.Error("Failed to parse JWT private key from settings", "error", err)
			return nil, nil, fmt.Errorf("failed to parse JWT_PRIVATE_KEY: %w", err)
		}

		publicKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(pubKeyPEM))
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse JWT_PUBLIC_KEY: %w", err)
		}

		if err := privateKey.Validate(); err != nil {
			return nil, nil, fmt.Errorf("invalid JWT private key: %w", err)
		}
		if privateKey.N.Cmp(publicKey.N) != 0 || privateKey.E != publicKey.E {
			return nil, nil, fmt.Errorf("mismatched JWT key pair")
		}
		if fromEnvironment {
			if _, err := s.Set(ctx, "jwt_private_key", privKeyPEM, "auth", "JWT Private Key", true); err != nil {
				return nil, nil, err
			}
			if _, err := s.Set(ctx, "jwt_public_key", pubKeyPEM, "auth", "JWT Public Key", false); err != nil {
				return nil, nil, err
			}
		}
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
		return nil, nil, fmt.Errorf("failed to persist JWT private key: %w", err)
	}

	_, err = s.Set(ctx, "jwt_public_key", string(pubPEMBytes), "auth", "JWT Public Key", false)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to persist JWT public key: %w", err)
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
