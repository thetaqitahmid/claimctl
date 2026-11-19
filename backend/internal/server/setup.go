package server

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/utils"
)

const defaultSetupTokenFile = "./keys/setup-token"

// PrepareSetupToken returns the token required to complete first-run setup.
// It returns an empty token when an administrator already exists. The token is
// kept in memory only and is regenerated on every startup unless SETUP_TOKEN is
// provided.
func PrepareSetupToken(ctx context.Context, q db.Querier) (string, error) {
	count, err := q.CountAdminUsers(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to count admin users: %w", err)
	}
	if count != 0 {
		return "", nil
	}

	token := os.Getenv("SETUP_TOKEN")
	if token != "" {
		slog.Warn("FIRST-RUN SETUP REQUIRED: no admin account exists. Complete setup using the configured SETUP_TOKEN.")
		return token, nil
	}

	token, err = utils.GenerateSetupToken()
	if err != nil {
		return "", err
	}

	tokenFile := utils.GetEnv("SETUP_TOKEN_FILE", defaultSetupTokenFile)
	if err := writeSetupTokenFile(tokenFile, token); err != nil {
		slog.Warn("Could not write setup token file; use the setup token logged here", "path", tokenFile, "error", err)
	} else {
		slog.Warn("Wrote first-run setup token to file", "path", tokenFile)
	}

	slog.Warn("FIRST-RUN SETUP REQUIRED: no admin account exists. Create the first administrator at the web UI.", "setup_token", token)
	return token, nil
}

func writeSetupTokenFile(path, token string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(token+"\n"), 0600)
}
