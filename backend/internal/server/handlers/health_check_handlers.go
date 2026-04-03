package handlers

import (
	"database/sql"
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/services"
	"github.com/thetaqitahmid/claimctl/internal/utils"
)

type HealthCheckHandler struct {
	healthCheckService services.HealthCheckService
}

func NewHealthCheckHandler(healthCheckService services.HealthCheckService) *HealthCheckHandler {
	return &HealthCheckHandler{
		healthCheckService: healthCheckService,
	}
}

type HealthConfigRequest struct {
	Enabled         bool   `json:"enabled"`
	CheckType       string `json:"checkType"`
	Target          string `json:"target"`
	IntervalSeconds int32  `json:"intervalSeconds"`
	TimeoutSeconds  int32  `json:"timeoutSeconds"`
	RetryCount      int32  `json:"retryCount"`
}

// GetHealthConfig retrieves health check configuration for a resource
func (h *HealthCheckHandler) GetHealthConfig(c *fiber.Ctx) error {
	id, err := utils.GetUUIDParam(c, "id")
	if err != nil {
		return nil
	}

	config, err := h.healthCheckService.GetHealthConfig(c.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Health check not configured for this resource",
			})
		}
		slog.Error("Failed to get health config", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get health configuration",
		})
	}

	return c.JSON(config)
}

// UpsertHealthConfig creates or updates health check configuration
func (h *HealthCheckHandler) UpsertHealthConfig(c *fiber.Ctx) error {
	id, err := utils.GetUUIDParam(c, "id")
	if err != nil {
		return nil
	}

	var req HealthConfigRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate check type
	if req.CheckType != "ping" && req.CheckType != "http" && req.CheckType != "tcp" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid check type. Must be 'ping', 'http', or 'tcp'",
		})
	}

	// Validate target
	if req.Target == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Target is required",
		})
	}

	// Set defaults if not provided
	if req.IntervalSeconds == 0 {
		req.IntervalSeconds = 60
	}
	if req.TimeoutSeconds == 0 {
		req.TimeoutSeconds = 5
	}
	if req.RetryCount == 0 {
		req.RetryCount = 3
	}

	params := db.UpsertHealthConfigParams{
		ResourceID:      id,
		Enabled:         pgtype.Bool{Bool: req.Enabled, Valid: true},
		CheckType:       req.CheckType,
		Target:          req.Target,
		IntervalSeconds: pgtype.Int4{Int32: req.IntervalSeconds, Valid: true},
		TimeoutSeconds: pgtype.Int4{Int32: req.TimeoutSeconds, Valid: true},
		RetryCount: pgtype.Int4{Int32: req.RetryCount, Valid: true},
	}

	config, err := h.healthCheckService.UpsertHealthConfig(c.Context(), params)
	if err != nil {
		slog.Error("Failed to upsert health config", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to save health configuration",
		})
	}

	return c.JSON(config)
}

// DeleteHealthConfig removes health check configuration
func (h *HealthCheckHandler) DeleteHealthConfig(c *fiber.Ctx) error {
	id, err := utils.GetUUIDParam(c, "id")
	if err != nil {
		return nil
	}

	if err := h.healthCheckService.DeleteHealthConfig(c.Context(), id); err != nil {
		slog.Error("Failed to delete health config", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to delete health configuration",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GetHealthStatus retrieves current health status for a resource
func (h *HealthCheckHandler) GetHealthStatus(c *fiber.Ctx) error {
	id, err := utils.GetUUIDParam(c, "id")
	if err != nil {
		return nil
	}

	status, err := h.healthCheckService.GetHealthStatus(c.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "No health status available for this resource",
			})
		}
		slog.Error("Failed to get health status", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get health status",
		})
	}

	return c.JSON(status)
}

// GetHealthHistory retrieves health check history for a resource
func (h *HealthCheckHandler) GetHealthHistory(c *fiber.Ctx) error {
	id, err := utils.GetUUIDParam(c, "id")
	if err != nil {
		return nil
	}

	// Get limit from query parameter, default to 10
	limit := int32(10)
	if limitStr := c.Query("limit"); limitStr != "" {
		if limitVal := c.QueryInt("limit", 10); limitVal > 0 {
			limit = int32(limitVal)
		}
	}

	history, err := h.healthCheckService.GetHealthHistory(c.Context(), id, limit)
	if err != nil {
		slog.Error("Failed to get health history", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get health history",
		})
	}

	return c.JSON(history)
}

// TriggerHealthCheck manually triggers a health check for a resource
func (h *HealthCheckHandler) TriggerHealthCheck(c *fiber.Ctx) error {
	id, err := utils.GetUUIDParam(c, "id")
	if err != nil {
		return nil
	}

	if err := h.healthCheckService.ExecuteCheck(c.Context(), id); err != nil {
		slog.Error("Failed to execute health check", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to execute health check",
		})
	}

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"message": "Health check triggered successfully",
	})
}
