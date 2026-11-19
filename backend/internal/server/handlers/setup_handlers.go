package handlers

import (
	"crypto/subtle"
	"errors"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v2"

	"github.com/thetaqitahmid/claimctl/internal/services"
	"github.com/thetaqitahmid/claimctl/internal/utils"
)

// SetupHandler exposes the first-run setup endpoints. It is only usable while
// no administrator exists and a setup token is available.
type SetupHandler struct {
	userService services.UserService
	mu          sync.Mutex
	token       string
}

func NewSetupHandler(userService services.UserService, token string) *SetupHandler {
	return &SetupHandler{userService: userService, token: token}
}

// GetStatus reports whether the installation still needs its first admin.
func (h *SetupHandler) GetStatus(c *fiber.Ctx) error {
	needsSetup, err := h.userService.NeedsSetup(c.UserContext())
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to determine setup status", err)
	}
	return c.JSON(fiber.Map{"needsSetup": needsSetup})
}

// CreateInitialAdmin creates the first administrator when setup is active.
func (h *SetupHandler) CreateInitialAdmin(c *fiber.Ctx) error {
	if !h.tokenMatches(c.Get("X-Setup-Token")) {
		return utils.SendError(c, fiber.StatusForbidden, "Invalid or expired setup token", nil)
	}

	var req struct {
		Name     string `json:"name" validate:"required"`
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required,min=8"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Invalid request body", err)
	}
	if err := utils.ValidateStruct(req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Validation error", err)
	}

	user, err := h.userService.CreateInitialAdmin(c.UserContext(), services.InitialAdminRequest{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, services.ErrSetupAlreadyCompleted):
			h.invalidateToken()
			return utils.SendError(c, fiber.StatusConflict, "Setup has already been completed", err)
		case errors.Is(err, services.ErrSetupEmailInUse):
			return utils.SendError(c, fiber.StatusConflict, "Email is already in use", err)
		case errors.Is(err, services.ErrInvalidSetupInput):
			message := strings.TrimPrefix(err.Error(), services.ErrInvalidSetupInput.Error()+": ")
			return utils.SendError(c, fiber.StatusBadRequest, message, err)
		default:
			return utils.SendError(c, fiber.StatusInternalServerError, "Failed to create admin account", err)
		}
	}

	h.invalidateToken()
	return c.Status(fiber.StatusCreated).JSON(user)
}

func (h *SetupHandler) tokenMatches(candidate string) bool {
	if candidate == "" {
		return false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.token == "" || len(candidate) != len(h.token) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(candidate), []byte(h.token)) == 1
}

func (h *SetupHandler) invalidateToken() {
	h.mu.Lock()
	h.token = ""
	h.mu.Unlock()
}
