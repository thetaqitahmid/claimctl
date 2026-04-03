package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/services"
	"github.com/thetaqitahmid/claimctl/internal/testutils"
	"github.com/thetaqitahmid/claimctl/internal/utils"
)

func TestSettingsHandler_GetSettings(t *testing.T) {
	app := fiber.New()
	mockDB := &testutils.MockQuerier{}
	settingsService := services.NewSettingsService(mockDB, "dummy_key")
	handler := NewSettingsHandler(settingsService, &testutils.MockAuditService{})

	app.Get("/admin/settings", handler.GetSettings)

	t.Run("Successfully get settings", func(t *testing.T) {
		mockSettings := []db.AppSetting{
			{
				Key:         "smtp_host",
				Value:       "smtp.example.com",
				Category:    "notification",
				Description: pgtype.Text{String: "SMTP Host", Valid: true},
				IsSecret:    false,
			},
			{
				Key:         "slack_token",
				Value:       "real_token",
				Category:    "notification",
				Description: pgtype.Text{String: "Slack Token", Valid: true},
				IsSecret:    true,
			},
		}

		mockDB.On("GetSettings", mock.Anything).Return(mockSettings, nil).Once()

		req := httptest.NewRequest("GET", "/admin/settings", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, 200, resp.StatusCode)

		var response []map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&response)

		assert.Len(t, response, 2)
		assert.Equal(t, "smtp_host", response[0]["key"])
		assert.Equal(t, "smtp.example.com", response[0]["value"])

		// Secret should be masked
		assert.Equal(t, "slack_token", response[1]["key"])
		assert.Equal(t, "********", response[1]["value"])
	})

	t.Run("Filter out internal keys", func(t *testing.T) {
		mockSettings := []db.AppSetting{
			{
				Key:         "jwt_private_key",
				Value:       "private_key_content",
				Category:    "auth",
				Description: pgtype.Text{String: "JWT Private Key", Valid: true},
				IsSecret:    true,
			},
			{
				Key:         "public_setting",
				Value:       "public_value",
				Category:    "general",
				Description: pgtype.Text{String: "Public Setting", Valid: true},
				IsSecret:    false,
			},
		}

		mockDB.On("GetSettings", mock.Anything).Return(mockSettings, nil).Once()

		req := httptest.NewRequest("GET", "/admin/settings", nil)
		resp, _ := app.Test(req)

		assert.Equal(t, 200, resp.StatusCode)

		var response []map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&response)

		// Should only have 1 item (public_setting)
		assert.Len(t, response, 1)
		assert.Equal(t, "public_setting", response[0]["key"])
	})
}

func TestSettingsHandler_UpdateSetting(t *testing.T) {
	app := fiber.New()
	mockDB := &testutils.MockQuerier{}
	settingsService := services.NewSettingsService(mockDB, "dummy_key")
	handler := NewSettingsHandler(settingsService, &testutils.MockAuditService{})

	app.Put("/admin/settings", handler.UpdateSetting)

	t.Run("Successfully update setting", func(t *testing.T) {
		payload := map[string]interface{}{
			"key":         "new_key",
			"value":       "new_value",
			"category":    "general",
			"description": "New Setting",
			"is_secret":   false,
		}
		body, _ := json.Marshal(payload)

		mockDB.On("UpsertSetting", mock.Anything, mock.MatchedBy(func(arg db.UpsertSettingParams) bool {
			return arg.Key == "new_key" && arg.Value == "new_value"
		})).Return(db.AppSetting{
			Key: "new_key", Value: "new_value",
			Category: "general", Description: pgtype.Text{String: "New Setting", Valid: true},
		}, nil).Once()

		req := httptest.NewRequest("PUT", "/admin/settings", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req)

		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("Update secret without changing value (masked)", func(t *testing.T) {
		// Generate valid key
		key := "12345678901234567890123456789012" // 32 chars
		keyBase64 := base64.StdEncoding.EncodeToString([]byte(key))

		// Use a local service with valid key
		localService := services.NewSettingsService(mockDB, keyBase64)
		localHandler := NewSettingsHandler(localService, &testutils.MockAuditService{})

		// Encrypt the current secret so Decrypt sets it correctly
		encryptedCurrent, _ := utils.Encrypt("current_secret", keyBase64)

		// First, need to mock GetSetting to retrieve current value
		mockDB.On("GetSetting", mock.Anything, "secret_key").Return(db.AppSetting{
			Key: "secret_key", Value: encryptedCurrent, IsSecret: true,
		}, nil).Once()

		payload := map[string]interface{}{
			"key":       "secret_key",
			"value":     "********",
			"category":  "auth",
			"is_secret": true,
		}
		body, _ := json.Marshal(payload)

		mockDB.On("UpsertSetting", mock.Anything, mock.MatchedBy(func(arg db.UpsertSettingParams) bool {
			// Verify key matches
			if arg.Key != "secret_key" {
				return false
			}
			// Verify it's not plain text "current_secret"
			if arg.Value == "current_secret" {
				return false
			}
			// We can try to decrypt it to verify it matches
			decrypted, err := utils.Decrypt(arg.Value, keyBase64)
			return err == nil && decrypted == "current_secret"
		})).Return(db.AppSetting{}, nil).Once()

		// Re-register route for local handler
		app.Put("/admin/settings/local", localHandler.UpdateSetting)

		req := httptest.NewRequest("PUT", "/admin/settings/local", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp, _ := app.Test(req)

		assert.Equal(t, 200, resp.StatusCode)
	})
}
