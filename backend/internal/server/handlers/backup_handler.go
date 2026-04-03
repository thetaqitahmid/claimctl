package handlers

import (
	"fmt"
	"io"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/thetaqitahmid/claimctl/internal/services"
)

// BackupHandler handles backup and restore HTTP requests.
type BackupHandler struct {
	backupService *services.BackupService
}

// NewBackupHandler creates a new BackupHandler.
func NewBackupHandler(backupService *services.BackupService) *BackupHandler {
	return &BackupHandler{backupService: backupService}
}

// CreateBackup exports all data and returns a downloadable JSON file.
func (h *BackupHandler) CreateBackup(c *fiber.Ctx) error {
	backup, err := h.backupService.CreateBackup(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create backup: " + err.Error(),
		})
	}

	filename := fmt.Sprintf("claimctl-backup-%s.json",
		time.Now().Format("2006-01-02T150405"))

	c.Set("Content-Type", "application/json")
	c.Set("Content-Disposition",
		fmt.Sprintf("attachment; filename=\"%s\"", filename))

	return c.JSON(backup)
}

// RestoreBackup accepts a JSON backup file and replaces all data.
func (h *BackupHandler) RestoreBackup(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Missing backup file. Upload as multipart field 'file'.",
		})
	}

	f, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to read uploaded file",
		})
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to read file content",
		})
	}

	if err := h.backupService.RestoreBackup(c.Context(), data); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Restore failed: " + err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Backup restored successfully",
	})
}
