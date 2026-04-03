package utils

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// SendError logs the internal error and sends a sanitized error message to the client.
func SendError(c *fiber.Ctx, status int, publicMessage string, internalError error) error {
	if internalError != nil {
		slog.Info("Internal Error", "status", status, "error", internalError)
	}
	// If sensitive internal error is nil, just log the public message as info
	if internalError == nil {
		slog.Info("Error Response", "status", status, "error", publicMessage)
	}

	return c.Status(status).JSON(fiber.Map{
		"error": publicMessage,
	})
}

// GetUUIDParam parses a UUID from a Fiber context parameter.
// It automatically sends a 400 Bad Request response if the UUID is invalid.
// If it returns an error, the caller should return nil to stop further processing.
func GetUUIDParam(c *fiber.Ctx, name string) (uuid.UUID, error) {
	param := c.Params(name)
	id, err := uuid.Parse(param)
	if err != nil {
		return uuid.Nil, SendError(c, fiber.StatusBadRequest, "Invalid ID format. Must be a valid UUID.", err)
	}
	return id, nil
}
