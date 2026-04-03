package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/thetaqitahmid/claimctl/internal/services"
	"github.com/thetaqitahmid/claimctl/internal/utils"
)

type APITokenHandler struct {
	service services.APITokenService
}

func NewAPITokenHandler(service services.APITokenService) *APITokenHandler {
	return &APITokenHandler{service: service}
}

func (h *APITokenHandler) GenerateToken(c *fiber.Ctx) error {
	user, err := GetUserFromContext(c)
	if err != nil {
		return utils.SendError(c, fiber.StatusUnauthorized, "Unauthorized", err)
	}
	userID := user.ID

	var req struct {
		Name      string `json:"name"`
		ExpiresIn string `json:"expires_in"` // optional, e.g., "30d", "1y"
	}

	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Invalid request body", err)
	}

	if req.Name == "" {
		return utils.SendError(c, fiber.StatusBadRequest, "Token name is required", nil)
	}

	var expiresAt *time.Time
	if req.ExpiresIn != "" {
		duration, err := time.ParseDuration(req.ExpiresIn)
		if err == nil {
			exp := time.Now().Add(duration)
			expiresAt = &exp
		}
		// If parsing fails or not provided, we can default to nil (never expires) or handle error
	}

	tokenString, tokenRecord, err := h.service.GenerateToken(c.Context(), userID, req.Name, expiresAt)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to generate token", err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"token":     tokenString,
		"id":        tokenRecord.ID,
		"name":      tokenRecord.Name,
		"createdAt": tokenRecord.CreatedAt,
		"expiresAt": tokenRecord.ExpiresAt,
	})
}

func (h *APITokenHandler) ListTokens(c *fiber.Ctx) error {
	user, err := GetUserFromContext(c)
	if err != nil {
		return utils.SendError(c, fiber.StatusUnauthorized, "Unauthorized", err)
	}
	userID := user.ID

	tokens, err := h.service.ListTokens(c.Context(), userID)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to list tokens", err)
	}

	return c.JSON(tokens)
}

func (h *APITokenHandler) RevokeToken(c *fiber.Ctx) error {
	user, err := GetUserFromContext(c)
	if err != nil {
		return utils.SendError(c, fiber.StatusUnauthorized, "Unauthorized", err)
	}
	userID := user.ID
	tokenID := c.Params("id")

	err = h.service.RevokeToken(c.Context(), tokenID, userID)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to revoke token", err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
