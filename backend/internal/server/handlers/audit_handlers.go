package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/services"
	"github.com/thetaqitahmid/claimctl/internal/utils"
)

type AuditHandler struct {
	auditService services.AuditService
}

func NewAuditHandler(auditService services.AuditService) *AuditHandler {
	return &AuditHandler{auditService: auditService}
}

// GetAuditLogs retrieves a paginated list of audit logs
// @Summary Get audit logs
// @Description Retrieve a list of audit logs for admin monitoring
// @Tags audit
// @Accept json
// @Produce json
// @Param limit query int false "Limit"
// @Param offset query int false "Offset"
// @Success 200 {array} db.GetAuditLogsRow
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/admin/audit-logs [get]
func (h *AuditHandler) GetAuditLogs(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)

	logs, err := h.auditService.GetAuditLogs(c.Context(), int32(limit), int32(offset))
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to retrieve audit logs", err)
	}

	if logs == nil {
		logs = []db.GetAuditLogsRow{}
	}

	return c.JSON(logs)
}
