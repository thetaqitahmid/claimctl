package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/thetaqitahmid/claimctl/internal/services"
	"github.com/thetaqitahmid/claimctl/internal/utils"
)

type SecretHandler struct {
	service *services.SecretService
	auditService services.AuditService
}

func NewSecretHandler(service *services.SecretService, auditService services.AuditService) *SecretHandler {
	return &SecretHandler{
		service: service,
		auditService: auditService,
	}
}

func (h *SecretHandler) CreateSecret(c *fiber.Ctx) error {
	type Request struct {
		Key         string `json:"key"`
		Value       string `json:"value"`
		Description string `json:"description"`
	}
	var req Request
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Invalid request body", err)
	}

	secret, err := h.service.CreateSecret(c.Context(), req.Key, req.Value, req.Description)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to create secret", err)
	}

	actor, _ := GetUserFromContext(c)
	if actor != nil {
		h.auditService.Log(c.Context(), actor.ID, "CREATE", "SECRET", secret.ID.String(), nil, c.IP())
	}

	return c.Status(fiber.StatusCreated).JSON(secret)
}

func (h *SecretHandler) ListSecrets(c *fiber.Ctx) error {
	secrets, err := h.service.ListSecrets(c.Context())
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to list secrets", err)
	}
	// Mask values
	for i := range secrets {
		secrets[i].Value = "*****"
	}
	return c.JSON(secrets)
}

func (h *SecretHandler) UpdateSecret(c *fiber.Ctx) error {
	id, err := utils.GetUUIDParam(c, "id")
	if err != nil {
		return nil
	}

	type Request struct {
		Value       string `json:"value"`
		Description string `json:"description"`
	}
	var req Request
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Invalid request body", err)
	}

	secret, err := h.service.UpdateSecret(c.Context(), id, req.Value, req.Description)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to update secret", err)
	}

	actor, _ := GetUserFromContext(c)
	if actor != nil {
		h.auditService.Log(c.Context(), actor.ID, "UPDATE", "SECRET", id.String(), nil, c.IP())
	}

	return c.JSON(secret)
}

func (h *SecretHandler) DeleteSecret(c *fiber.Ctx) error {
	id, err := utils.GetUUIDParam(c, "id")
	if err != nil {
		return nil
	}

	if err := h.service.DeleteSecret(c.Context(), id); err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to delete secret", err)
	}

	actor, _ := GetUserFromContext(c)
	if actor != nil {
		h.auditService.Log(c.Context(), actor.ID, "DELETE", "SECRET", id.String(), nil, c.IP())
	}

	return c.SendStatus(fiber.StatusNoContent)
}
