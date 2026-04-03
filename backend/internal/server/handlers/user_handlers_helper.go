package handlers

import (
	"fmt"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/oauth2"

	"github.com/thetaqitahmid/claimctl/internal/db"
)

// GetUserFromContext retrieves the user from Fiber context, handling both JWT and API Token
func GetUserFromContext(c *fiber.Ctx) (*db.ClaimctlUser, error) {
	// 1. Check for API Token User Record
	if userRecord := c.Locals("user_record"); userRecord != nil {
		if u, ok := userRecord.(*db.ClaimctlUser); ok {
			return u, nil
		}
	}

	// 2. Check for JWT
	if userToken := c.Locals("user"); userToken != nil {
		if token, ok := userToken.(*jwt.Token); ok {
			claims := token.Claims.(jwt.MapClaims)
			userIDStr := fmt.Sprintf("%v", claims["id"])
			userID, err := uuid.Parse(userIDStr)
			if err != nil {
				return nil, fmt.Errorf("invalid user ID in token")
			}

			return &db.ClaimctlUser{
				ID:    userID,
				Email: claims["email"].(string),
				Name:  claims["name"].(string),

				Role:   claims["role"].(string),
				Status: claims["status"].(string),
			}, nil
		}
	}

	return nil, fmt.Errorf("no authenticated user found in context")
}

// getOIDCConfig retrieves OIDC configuration and initializes provider
func (h *UserHandler) getOIDCConfig(c *fiber.Ctx) (*oidc.Provider, *oauth2.Config, error) {
	ctx := c.Context()
	issuer := h.settingsService.GetString(ctx, "oidc_issuer")
	clientID := h.settingsService.GetString(ctx, "oidc_client_id")
	clientSecret := h.settingsService.GetString(ctx, "oidc_client_secret")
	redirectURL := h.settingsService.GetString(ctx, "oidc_redirect_url")
	scopesStr := h.settingsService.GetString(ctx, "oidc_scopes")

	if issuer == "" || clientID == "" || clientSecret == "" {
		return nil, nil, fmt.Errorf("OIDC configuration incomplete")
	}

	// If redirect URL is not configured, derive it from the request
	if redirectURL == "" {
		redirectURL = c.BaseURL() + "/api/auth/oidc/callback"
	}

	scopes := []string{oidc.ScopeOpenID, "profile", "email"}
	if scopesStr != "" {
		scopes = strings.Split(scopesStr, " ")
	}

	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query provider: %w", err)
	}

	conf := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       scopes,
	}

	return provider, conf, nil
}
