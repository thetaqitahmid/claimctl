package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/thetaqitahmid/claimctl/internal/services"
)

type PreferenceHandler struct {
	prefService services.PreferenceService
}

func NewPreferenceHandler(prefService services.PreferenceService) *PreferenceHandler {
	return &PreferenceHandler{prefService: prefService}
}

func (h *PreferenceHandler) GetPreferences(c *fiber.Ctx) error {
	user, err := GetUserFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	userID := user.ID

	prefs, err := h.prefService.GetPreferences(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to fetch preferences"})
	}

	return c.JSON(prefs)
}

func (h *PreferenceHandler) UpdatePreference(c *fiber.Ctx) error {
	user, err := GetUserFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
	}
	userID := user.ID

	var payload struct {
		EventType string `json:"eventType"`
		Channel   string `json:"channel"`
		Enabled   bool   `json:"enabled"`
	}

	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid payload"})
	}

	updated, err := h.prefService.UpsertPreference(c.Context(), userID, payload.EventType, payload.Channel, payload.Enabled)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update preference"})
	}

	return c.JSON(updated)
}
