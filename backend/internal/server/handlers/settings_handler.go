package handlers

import (
	"context"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/thetaqitahmid/claimctl/internal/services"
)

type SettingsHandler struct {
	settingsService *services.SettingsService
	auditService    services.AuditService
}

func NewSettingsHandler(s *services.SettingsService, auditService services.AuditService) *SettingsHandler {
	return &SettingsHandler{settingsService: s, auditService: auditService}
}

func (h *SettingsHandler) GetSettings(c *fiber.Ctx) error {
	settings, err := h.settingsService.GetSettings(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Mask secrets
	type SettingResponse struct {
		Key         string `json:"key"`
		Value       string `json:"value"`
		Category    string `json:"category"`
		Description string `json:"description"`
		IsSecret    bool   `json:"is_secret"`
	}

	response := make([]SettingResponse, 0, len(settings))
	internalKeys := map[string]bool{
		"jwt_private_key": true,
		"jwt_public_key":  true,
	}

	for _, s := range settings {
		if internalKeys[s.Key] {
			continue
		}

		val := s.Value
		if s.IsSecret {
			val = "********" // Masked
		}
		response = append(response, SettingResponse{
			Key:         s.Key,
			Value:       val,
			Category:    s.Category,
			Description: s.Description.String,
			IsSecret:    s.IsSecret,
		})
	}

	return c.JSON(response)
}

func (h *SettingsHandler) UpdateSetting(c *fiber.Ctx) error {
	var payload struct {
		Key         string `json:"key"`
		Value       string `json:"value"`
		Category    string `json:"category"`
		Description string `json:"description"`
		IsSecret    bool   `json:"is_secret"`
	}

	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid payload"})
	}

	// If updating a secret and sending mask, assume no change?
	// Or simplistic approach: if value is literally "********", ignore update?
	if payload.IsSecret && payload.Value == "********" {
		current := h.settingsService.GetString(c.Context(), payload.Key)
		if current != "" {
			payload.Value = current
		}
	}

	updated, err := h.settingsService.Set(context.Background(), payload.Key, payload.Value, payload.Category, payload.Description, payload.IsSecret)
	if err != nil {
		slog.Error("Failed to update setting", "key", payload.Key, "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update setting"})
	}

	actor, _ := GetUserFromContext(c)
	if actor != nil {
		h.auditService.Log(c.Context(), actor.ID, "UPDATE", "SETTING", payload.Key, updated, c.IP())
	}

	return c.JSON(updated)
}

