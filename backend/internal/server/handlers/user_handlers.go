package handlers

import (
	"crypto/rsa"
	"fmt"
	"net/mail"
	"time"

	"github.com/google/uuid"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"

	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/services"
	"github.com/thetaqitahmid/claimctl/internal/utils"

	"golang.org/x/oauth2"
)

type CreateUserParams = db.CreateUserParams
type UpdateUserByIdParams = db.UpdateUserByIdParams

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// JWT Claims structure
type Claims struct {
	UserID uuid.UUID  `json:"id"`
	Email  string `json:"email"`
	Name   string `json:"name"`

	Role   string `json:"role"`
	Status string `json:"status"`
	jwt.RegisteredClaims
}

type UserHandler struct {
	userService     services.UserService
	settingsService *services.SettingsService
	privateKey      *rsa.PrivateKey
	auditService    services.AuditService
}

func NewUserHandler(userService services.UserService, settingsService *services.SettingsService, privateKey *rsa.PrivateKey, auditService services.AuditService) *UserHandler {
	return &UserHandler{
		userService:     userService,
		settingsService: settingsService,
		privateKey:      privateKey,
		auditService:    auditService,
	}
}

// Generate JWT token
func (h *UserHandler) generateJWT(userID uuid.UUID, email string, name string, role string, status string) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		Name:   name,

		Role:   role,
		Status: status,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "claimctl",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(h.privateKey)
}

// Login handles user login
// @Summary User Login
// @Description Authenticate a user and return a JWT token.
// @Tags auth
// @Accept json
// @Produce json
// @Param loginRequest body LoginRequest true "Login Credentials"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /login [post]
func (h *UserHandler) Login(c *fiber.Ctx) error {
	var req struct {
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required"`
	}

	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Invalid request body", err)
	}

	if err := utils.ValidateStruct(req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Validation error", err)
	}

	loginRequest := services.LoginRequest{
		Email: req.Email,
		Password: req.Password,
	}

	user, err := h.userService.Login(c.Context(), loginRequest)
	if err != nil {
		if err.Error() == "account temporarily locked. Please try again later" {
			return utils.SendError(c, fiber.StatusUnauthorized, "Account is locked after too many failed attempts", err)
		}
		return utils.SendError(c, fiber.StatusUnauthorized, "Invalid credentials", err)
	}

	token, err := h.generateJWT(user.ID, user.Email, user.Name, user.Role, user.Status)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to generate token", err)
	}

	// Set JWT in an HTTP-only cookie
	c.Cookie(&fiber.Cookie{
		Name:     "jwt",
		Value:    token,
		Expires:  time.Now().Add(24 * time.Hour),
		HTTPOnly: true,
		Secure:   utils.GetEnvAsBool("COOKIE_SECURE", true),
		SameSite: utils.GetEnv("COOKIE_SAMESITE", "Strict"),
		Path:     "/",
	})

	return c.JSON(fiber.Map{
		"user": fiber.Map{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,

			"role":   user.Role,
			"status": user.Status,
		},
		"message": "Login successful",
	})
}

// Logout logs out the user
func (h *UserHandler) Logout(c *fiber.Ctx) error {
	c.Cookie(&fiber.Cookie{
		Name:     "jwt",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour), // Expire immediately
		HTTPOnly: true,
		Secure:   utils.GetEnvAsBool("COOKIE_SECURE", true),
		SameSite: utils.GetEnv("COOKIE_SAMESITE", "Strict"),
		Path:     "/",
	})
	return c.JSON(fiber.Map{
		"message": "Logout successful",
	})
}

// GetMe returns the current user
// @Summary Get current user
// @Description Retrieve details of the currently authenticated user.
// @Tags users
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Router /me [get]
func (h *UserHandler) GetMe(c *fiber.Ctx) error {
	user, err := GetUserFromContext(c)
	if err != nil {
		return utils.SendError(c, fiber.StatusUnauthorized, "Unauthorized", err)
	}
	fmt.Println("User is authenticated")
	return c.JSON(fiber.Map{
		"user": fiber.Map{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,

			"role":               user.Role,
			"status":             user.Status,
			"slack_destination":  user.SlackDestination.String,
			"teams_webhook_url":  user.TeamsWebhookUrl.String,
			"notification_email": user.NotificationEmail.String,
		},
		"message": "User retrieved successfully",
	})
}

// CreateUser creates a new user
// @Summary Create a new user
// @Description Create a new user account. Requires admin privileges.
// @Tags users
// @Accept json
// @Produce json
// @Param user body CreateUserParams true "New User Details"
// @Success 201 {object} ClaimctlUser
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /users [post]
func (h *UserHandler) CreateUser(c *fiber.Ctx) error {
	if !h.isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You do not have permission to create a user",
		})
	}

	var req struct {
		Email    string `json:"email" validate:"required,email"`
		Name     string `json:"name" validate:"required"`
		Password string `json:"password" validate:"required,min=8"`
		Role     string `json:"role" validate:"omitempty,oneof=admin user"`
		Status   string `json:"status" validate:"omitempty,oneof=active inactive"`
	}
	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Invalid request body", err)
	}

	if err := utils.ValidateStruct(req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Validation error", err)
	}

	user := db.CreateUserParams{
		Email: req.Email,
		Name: req.Name,
		Password: req.Password,
		Role: req.Role,
		Status: req.Status,
	}

	createdUser, err := h.userService.CreateUser(c.Context(), user)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to create user", err)
	}

	actor, _ := GetUserFromContext(c)
	if actor != nil {
		h.auditService.Log(c.Context(), actor.ID, "CREATE", "USER", createdUser.ID.String(), user, c.IP())
	}

	return c.Status(fiber.StatusCreated).JSON(createdUser)
}

// GetUsers returns all users
// @Summary Get all users
// @Description Retrieve a list of all users. Requires admin privileges.
// @Tags users
// @Accept json
// @Produce json
// @Success 200 {array} ClaimctlUser
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /users [get]
func (h *UserHandler) GetUsers(c *fiber.Ctx) error {
	if !h.isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You do not have permission to view users",
		})
	}
	users, err := h.userService.GetUsers(c.Context())
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to get users", err)
	}
	return c.JSON(users)
}

// GetUser returns a user by either ID or email
// @Summary Get a user
// @Description Retrieve a user by ID or email. Requires admin privileges.
// @Tags users
// @Accept json
// @Produce json
// @Param id query int false "User ID"
// @Param email query string false "User Email"
// @Success 200 {object} ClaimctlUser
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /users/find [get]
func (h *UserHandler) GetUser(c *fiber.Ctx) error {
	if !h.isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You do not have permission to view users",
		})
	}

	email := c.Query("email", "")
	idStr := c.Query("id", "")
	var id uuid.UUID
	if idStr != "" {
		var parseErr error
		id, parseErr = uuid.Parse(idStr)
		if parseErr != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid ID format",
			})
		}
	}

	user, err := h.userService.GetUser(c.Context(), email, id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Valid email address or ID is required",
		})
	}

	return c.JSON(user)
}

// UpdateUser updates a user by ID
func (h *UserHandler) UpdateUser(c *fiber.Ctx) error {
	if !h.isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You do not have permission to update users",
		})
	}
	id, err := utils.GetUUIDParam(c, "id")
	if err != nil {
		return nil
	}

	var user *services.UpdateUserRequest
	if err := c.BodyParser(&user); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Invalid request body", err)
	}
	idValue := id
	user.ID = &idValue

	err = h.userService.UpdateUser(c.Context(), user)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to update user", err)
	}

	actor, _ := GetUserFromContext(c)
	if actor != nil {
		h.auditService.Log(c.Context(), actor.ID, "UPDATE", "USER", idValue.String(), user, c.IP())
	}

	return nil
}

// DeleteUser deletes a user by ID
func (h *UserHandler) DeleteUser(c *fiber.Ctx) error {
	if !h.isAdmin(c) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You do not have permission to delete users",
		})
	}
	id, err := utils.GetUUIDParam(c, "id")
	if err != nil {
		return nil
	}

	err = h.userService.DeleteUser(c.Context(), id)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to delete user", err)
	}

	actor, _ := GetUserFromContext(c)
	if actor != nil {
		h.auditService.Log(c.Context(), actor.ID, "DELETE", "USER", id.String(), nil, c.IP())
	}

	return nil
}

// isAdmin checks if the user is an admin
func (h *UserHandler) isAdmin(c *fiber.Ctx) bool {
	user, err := GetUserFromContext(c)
	if err != nil {
		return false
	}
	return user.Role == "admin"
}

// LoginLDAP handles user login via LDAP
func (h *UserHandler) LoginLDAP(c *fiber.Ctx) error {
	var loginRequest services.LoginRequest
	if err := c.BodyParser(&loginRequest); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Invalid request body", err)
	}

	user, err := h.userService.LoginLDAP(c.Context(), loginRequest.Email, loginRequest.Password)
	if err != nil {
		return utils.SendError(c, fiber.StatusUnauthorized, "Invalid credentials", err)
	}

	token, err := h.generateJWT(user.ID, user.Email, user.Name, user.Role, user.Status)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to generate token", err)
	}

	// Set JWT in an HTTP-only cookie
	c.Cookie(&fiber.Cookie{
		Name:     "jwt",
		Value:    token,
		Expires:  time.Now().Add(24 * time.Hour),
		HTTPOnly: true,
		Secure:   utils.GetEnvAsBool("COOKIE_SECURE", true),
		SameSite: utils.GetEnv("COOKIE_SAMESITE", "Strict"),
		Path:     "/",
	})

	return c.JSON(fiber.Map{
		"user": fiber.Map{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,

			"role":   user.Role,
			"status": user.Status,
		},
		"message": "Login successful",
	})
}

// LoginOIDC handles redirection to OIDC provider
func (h *UserHandler) LoginOIDC(c *fiber.Ctx) error {
	_, oauth2Config, err := h.getOIDCConfig(c)
	if err != nil {
		return utils.SendError(c, fiber.StatusServiceUnavailable, "OIDC is not configured", nil)
	}

	state := utils.GenerateRandomString(16)
	verifier := oauth2.GenerateVerifier()

	// Store state and verifier in cookies
	c.Cookie(&fiber.Cookie{
		Name:     "oidc_state",
		Value:    state,
		Expires:  time.Now().Add(10 * time.Minute),
		HTTPOnly: true,
		Secure:   utils.GetEnvAsBool("COOKIE_SECURE", true),
		SameSite: utils.GetEnv("COOKIE_SAMESITE", "Strict"),
	})
	c.Cookie(&fiber.Cookie{
		Name:     "oidc_verifier",
		Value:    verifier,
		Expires:  time.Now().Add(10 * time.Minute),
		HTTPOnly: true,
		Secure:   utils.GetEnvAsBool("COOKIE_SECURE", true),
		SameSite: utils.GetEnv("COOKIE_SAMESITE", "Strict"),
	})

	authUrl := oauth2Config.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.S256ChallengeOption(verifier),
	)

	return c.Redirect(authUrl, fiber.StatusFound)
}

// CallbackOIDC handles the OIDC callback
func (h *UserHandler) CallbackOIDC(c *fiber.Ctx) error {
	oidcProvider, oauth2Config, err := h.getOIDCConfig(c)
	if err != nil {
		return utils.SendError(c, fiber.StatusServiceUnavailable, "OIDC is not configured", nil)
	}

	state := c.Query("state")
	code := c.Query("code")
	cookieState := c.Cookies("oidc_state")
	verifier := c.Cookies("oidc_verifier")

	if state != cookieState {
		return utils.SendError(c, fiber.StatusBadRequest, "Invalid state parameter", nil)
	}
	if verifier == "" {
		return utils.SendError(c, fiber.StatusBadRequest, "Missing PKCE verifier", nil)
	}

	oauth2Token, err := oauth2Config.Exchange(c.Context(), code, oauth2.VerifierOption(verifier))
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to exchange token", err)
	}

	tokenSource := oauth2Config.TokenSource(c.Context(), oauth2Token)
	userInfo, err := oidcProvider.UserInfo(c.Context(), tokenSource)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to get user info", err)
	}

	// Sync user
	var claims struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := userInfo.Claims(&claims); err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to parse claims", err)
	}

	user, err := h.userService.LoginOIDC(c.Context(), claims.Email, claims.Name)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to login/sync OIDC user", err)
	}

	// Generate JWT
	token, err := h.generateJWT(user.ID, user.Email, user.Name, user.Role, user.Status)
	if err != nil {
		return utils.SendError(c, fiber.StatusInternalServerError, "Failed to generate token", err)
	}

	// Set JWT
	c.Cookie(&fiber.Cookie{
		Name:     "jwt",
		Value:    token,
		Expires:  time.Now().Add(24 * time.Hour),
		HTTPOnly: true,
		Secure:   utils.GetEnvAsBool("COOKIE_SECURE", true),
		SameSite: utils.GetEnv("COOKIE_SAMESITE", "Strict"),
		Path:     "/",
	})

	// Clear OIDC cookies
	c.ClearCookie("oidc_state")
	c.ClearCookie("oidc_verifier")

	// Redirect to frontend
	frontendURL := utils.GetEnv("FRONTEND_URL", "/")
	return c.Redirect(frontendURL, fiber.StatusFound)
}

func (h *UserHandler) UpdateChannelConfig(c *fiber.Ctx) error {
	user, err := GetUserFromContext(c)
	if err != nil {
		return utils.SendError(c, fiber.StatusUnauthorized, "Unauthorized", err)
	}
	userID := user.ID

	var payload struct {
		SlackDestination string `json:"slack_destination"`
		TeamsWebhookUrl  string `json:"teams_webhook_url"`
		NotificationEmail string `json:"notification_email"`
	}

	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid payload"})
	}

	if payload.NotificationEmail != "" {
		if _, err := mail.ParseAddress(payload.NotificationEmail); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid notification email address"})
		}
	}

	updatedUser, err := h.userService.UpdateChannelConfig(c.Context(), userID, payload.SlackDestination, payload.TeamsWebhookUrl, payload.NotificationEmail)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to update channel config"})
	}

	return c.JSON(updatedUser)
}

func (h *UserHandler) TestEmailConfig(c *fiber.Ctx) error {
	user, err := GetUserFromContext(c)
	if err != nil {
		return utils.SendError(c, fiber.StatusUnauthorized, "Unauthorized", err)
	}

	dispatcher := services.NewEmailDispatcher(h.settingsService)
	payload := services.NotificationPayload{
		Subject: "Test Email from claimctl",
		Message: "This is a test email to verify your email notification configuration.",
	}

	recipient := user.Email
	if user.NotificationEmail.Valid && user.NotificationEmail.String != "" {
		recipient = user.NotificationEmail.String
	}

	err = dispatcher.Dispatch(c.Context(), recipient, payload)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to send test email: " + err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Test email sent successfully"})
}

// HandleChangePassword handles password change request
// @Summary Change Password
// @Description Change the password for the currently authenticated user.
// @Tags users
// @Accept json
// @Produce json
// @Param request body map[string]string true "Password Change Request"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /user/password [post]
func (h *UserHandler) HandleChangePassword(c *fiber.Ctx) error {
	user, err := GetUserFromContext(c)
	if err != nil {
		return utils.SendError(c, fiber.StatusUnauthorized, "Unauthorized", err)
	}

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}

	if err := c.BodyParser(&req); err != nil {
		return utils.SendError(c, fiber.StatusBadRequest, "Invalid request body", err)
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		return utils.SendError(c, fiber.StatusBadRequest, "Both current and new passwords are required", nil)
	}

	err = h.userService.UpdatePassword(c.Context(), user.ID, req.CurrentPassword, req.NewPassword)
	if err != nil {
		if err.Error() == "incorrect current password" {
			return utils.SendError(c, fiber.StatusUnauthorized, "Incorrect current password", err)
		}
		if err.Error() == "user not found" {
			return utils.SendError(c, fiber.StatusNotFound, "User not found", err)
		}
		return utils.SendError(c, fiber.StatusBadRequest, err.Error(), err)
	}

	return c.JSON(fiber.Map{
		"message": "Password updated successfully",
	})
}
