package handlers

import (
	"encoding/json"

	"github.com/google/uuid"

	"github.com/gofiber/fiber/v2"
	"github.com/thetaqitahmid/claimctl/internal/services"
	"github.com/thetaqitahmid/claimctl/internal/utils"
)

type WebhookHandler struct {
	service *services.WebhookService
}

func NewWebhookHandler(service *services.WebhookService) *WebhookHandler {
	return &WebhookHandler{service: service}
}

func (h *WebhookHandler) CreateWebhook(c *fiber.Ctx) error {
	type Request struct {
		Name        string            `json:"name"`
		Url         string            `json:"url"`
		Method      string            `json:"method"`
		Headers     map[string]string `json:"headers"`
		Template    string            `json:"template"`
		Description string            `json:"description"`
	}
	var req Request
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Invalid request payload", err)
	}

	if req.Method == "" {
		req.Method = "POST"
	}

	webhook, err := h.service.CreateWebhook(c.Context(), req.Name, req.Url, req.Method, req.Headers, req.Template, req.Description)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to create webhook", err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":            webhook.ID,
		"name":          webhook.Name,
		"url":           webhook.Url,
		"method":        webhook.Method,
		"headers":       json.RawMessage(webhook.Headers),
		"template":      webhook.Template.String,
		"description":   webhook.Description.String,
		"signingSecret": webhook.SigningSecret,
		"createdAt":     webhook.CreatedAt.Int64,
		"updatedAt":     webhook.UpdatedAt.Int64,
	})
}

func (h *WebhookHandler) ListWebhooks(c *fiber.Ctx) error {
	webhooks, err := h.service.ListWebhooks(c.Context())
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to list webhooks", err)
	}

	type WebhookResponse struct {
		ID uuid.UUID           `json:"id"`
		Name          string          `json:"name"`
		Url           string          `json:"url"`
		Method        string          `json:"method"`
		Headers       json.RawMessage `json:"headers"`
		Template      string          `json:"template"`
		Description   string          `json:"description"`
		CreatedAt     int64           `json:"createdAt"`
		UpdatedAt     int64           `json:"updatedAt"`
		SigningSecret string          `json:"signingSecret"`
	}

	response := make([]WebhookResponse, len(webhooks))
	for i, w := range webhooks {
		response[i] = WebhookResponse{
			ID:            w.ID,
			Name:          w.Name,
			Url:           w.Url,
			Method:        w.Method,
			Headers:       json.RawMessage(w.Headers),
			Template:      w.Template.String,
			Description:   w.Description.String,
			CreatedAt:     w.CreatedAt.Int64,
			UpdatedAt:     w.UpdatedAt.Int64,
			SigningSecret: w.SigningSecret,
		}
	}

	return c.JSON(response)
}

func (h *WebhookHandler) UpdateWebhook(c *fiber.Ctx) error {
	id, err := utils.GetUUIDParam(c, "id")
	if err != nil {
		return nil
	}

	type Request struct {
		Name        string            `json:"name"`
		Url         string            `json:"url"`
		Method      string            `json:"method"`
		Headers     map[string]string `json:"headers"`
		Template    string            `json:"template"`
		Description string            `json:"description"`
	}
	var req Request
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Invalid request body", err)
	}

	webhook, err := h.service.UpdateWebhook(c.Context(), id, req.Name, req.Url, req.Method, req.Headers, req.Template, req.Description)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to update webhook", err)
	}
	return c.JSON(fiber.Map{
		"id":            webhook.ID,
		"name":          webhook.Name,
		"url":           webhook.Url,
		"method":        webhook.Method,
		"headers":       json.RawMessage(webhook.Headers),
		"template":      webhook.Template.String,
		"description":   webhook.Description.String,
		"signingSecret": webhook.SigningSecret,
		"createdAt":     webhook.CreatedAt.Int64,
		"updatedAt":     webhook.UpdatedAt.Int64,
	})
}

func (h *WebhookHandler) DeleteWebhook(c *fiber.Ctx) error {
	id, err := utils.GetUUIDParam(c, "id")
	if err != nil {
		return nil
	}

	if err := h.service.DeleteWebhook(c.Context(), id); err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to delete webhook", err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// Resource Webhook Association
func (h *WebhookHandler) AddResourceWebhook(c *fiber.Ctx) error {
	resourceID, err := utils.GetUUIDParam(c, "resourceId")
	if err != nil {
		return nil
	}

	type Request struct {
		WebhookID uuid.UUID `json:"webhook_id"`
		Events    []string  `json:"events"`
	}
	var req Request
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Invalid request body", err)
	}

	if err := h.service.AddResourceWebhook(c.Context(), resourceID, req.WebhookID, req.Events); err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to add resource webhook", err)
	}
	return c.SendStatus(fiber.StatusCreated)
}

func (h *WebhookHandler) RemoveResourceWebhook(c *fiber.Ctx) error {
	resourceID, err := utils.GetUUIDParam(c, "resourceId")
	if err != nil {
		return nil
	}
	webhookID, err := utils.GetUUIDParam(c, "webhookId")
	if err != nil {
		return nil
	}

	if err := h.service.RemoveResourceWebhook(c.Context(), resourceID, webhookID); err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to remove resource webhook", err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *WebhookHandler) GetResourceWebhooks(c *fiber.Ctx) error {
	resourceID, err := utils.GetUUIDParam(c, "resourceId")
	if err != nil {
		return nil
	}

	webhooks, err := h.service.GetResourceWebhooks(c.Context(), resourceID)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to get resource webhooks", err)
	}
	return c.JSON(webhooks)
}

func (h *WebhookHandler) GetWebhookLogs(c *fiber.Ctx) error {
	id, err := utils.GetUUIDParam(c, "id")
	if err != nil {
		return nil
	}

	limit := c.QueryInt("limit", 20)
	offset := c.QueryInt("offset", 0)

	logs, err := h.service.GetWebhookLogs(c.Context(), id, int32(limit), int32(offset))
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to get logs", err)
	}
	return c.JSON(logs)
}
