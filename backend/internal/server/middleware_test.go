package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/testutils"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestAdminMiddleware(t *testing.T) {
	app := fiber.New()

	// Mock Authentication Middleware to inject user into context
	app.Use(func(c *fiber.Ctx) error {
		authType := c.Get("X-Auth-Type")

		if authType == "api-token-admin" {
			c.Locals("user_record", &db.ClaimctlUser{
				ID: testutils.TestUUID(1),
				Email: "admin@example.com",

				Role: "admin",
			})
		} else if authType == "api-token-user" {
			c.Locals("user_record", &db.ClaimctlUser{
				ID: testutils.TestUUID(2),
				Email: "user@example.com",

				Role: "user",
			})
		} else if authType == "jwt-admin" {
			token := &jwt.Token{
				Claims: jwt.MapClaims{
					"id":    testutils.TestUUID(1).String(),
					"email": "admin@example.com",
					"name":  "Admin User",

					"role":   "admin",
					"status": "active",
				},
			}
			c.Locals("user", token)
		} else if authType == "jwt-user" {
			token := &jwt.Token{
				Claims: jwt.MapClaims{
					"id":    testutils.TestUUID(2).String(),
					"email": "user@example.com",
					"name":  "Regular User",

					"role":   "user",
					"status": "active",
				},
			}
			c.Locals("user", token)
		}

		return c.Next()
	})

	// Apply Admin Middleware
	app.Get("/admin/protected", AdminMiddleware(), func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	tests := []struct {
		name           string
		authType       string
		expectedStatus int
	}{
		{
			name:           "API Token Admin should convert to 200 OK",
			authType:       "api-token-admin",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "API Token Non-Admin should convert to 403 Forbidden",
			authType:       "api-token-user",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "JWT Admin should convert to 200 OK",
			authType:       "jwt-admin",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "JWT Non-Admin should convert to 403 Forbidden",
			authType:       "jwt-user",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Unauthenticated should convert to 401 Unauthorized",
			authType:       "",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/admin/protected", nil)
			if tt.authType != "" {
				req.Header.Set("X-Auth-Type", tt.authType)
			}

			resp, err := app.Test(req)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}
