package server

import (
	"context"
	"crypto/rsa"
	"fmt"
	"time"

	jwtware "github.com/gofiber/contrib/jwt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"

	// Import Swagger dependency
	fiberSwagger "github.com/swaggo/fiber-swagger"
	// Import generated docs
	_ "github.com/thetaqitahmid/claimctl/docs"

	"github.com/thetaqitahmid/claimctl/internal/connection"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/server/handlers"
	"github.com/thetaqitahmid/claimctl/internal/services"
	"github.com/thetaqitahmid/claimctl/internal/workers"
)

// Define the routes
func SetupRoutes(ctx context.Context, app *fiber.App, dbConn *connection.DBConn, privateKey *rsa.PrivateKey, publicKey *rsa.PublicKey, encryptionKey string) {
	dbQueries := db.NewStore(dbConn)
	backupService := services.NewBackupService(dbConn)

	// Services
	settingsService := services.NewSettingsService(dbQueries, encryptionKey)
	// Auto-sync settings from Env to DB on startup
	if err := settingsService.SyncEnvToDB(ctx); err != nil {
		fmt.Printf("Warning: Failed to sync settings from Env: %v\n", err)
	}

	dispatchers := []services.NotificationDispatcher{
		services.NewEmailDispatcher(settingsService),
		services.NewSlackDispatcher(settingsService),
		services.NewTeamsDispatcher(),
	}
	preferenceService := services.NewPreferenceService(dbQueries)
	notificationService := services.NewNotificationService(dbQueries, dispatchers, preferenceService)

	realtimeService := services.NewRealtimeService()
	resourceService := services.NewResourceService(dbQueries, realtimeService)
	userService := services.NewUserService(dbQueries)
	reservationHistoryService := services.NewReservationHistoryService(dbQueries)
	secretService := services.NewSecretService(dbQueries, encryptionKey)
	webhookService := services.NewWebhookService(dbQueries, secretService)
	apiTokenService := services.NewAPITokenService(dbQueries)
	preferenceHandler := handlers.NewPreferenceHandler(preferenceService)
	backupHandler := handlers.NewBackupHandler(backupService)
	reservationService := services.NewReservationService(dbQueries, reservationHistoryService, webhookService, realtimeService, notificationService)

	spaceService := services.NewSpaceService(dbQueries)
	groupService := services.NewGroupService(dbQueries) // New Group Service
	healthCheckService := services.NewHealthCheckService(dbQueries)
	auditService := services.NewAuditService(dbQueries)

	// Workers
	expiryWorker := workers.NewExpiryWorker(reservationService)
	expiryWorker.Start(ctx, 1 * time.Minute)

	// Start health check monitoring
	if err := healthCheckService.StartMonitoring(ctx); err != nil {
		fmt.Printf("Warning: Failed to start health check monitoring: %v\n", err)
	}

	// Start Cleanup Worker
	cleanupWorker := workers.NewCleanupWorker(dbQueries)
	cleanupWorker.Start(ctx)

	// Handlers
	resourceHanlder := handlers.NewResourceHandler(resourceService, auditService)

	// OIDC is now handled dynamically inside UserHandler via SettingsService
	userHandler := handlers.NewUserHandler(userService, settingsService, privateKey, auditService)
	reservationHandler := handlers.NewReservationHandler(reservationService)
	reservationHistoryHandler := handlers.NewReservationHistoryHandler(reservationHistoryService)
	secretHandler := handlers.NewSecretHandler(secretService, auditService)
	webhookHandler := handlers.NewWebhookHandler(webhookService)
	spaceHandler := handlers.NewSpaceHandler(spaceService, auditService)
	groupHandler := handlers.NewGroupHandler(groupService, auditService) // New Group Handler
	realtimeHandler := handlers.NewRealtimeHandler(realtimeService)
	settingsHandler := handlers.NewSettingsHandler(settingsService, auditService)
	apiTokenHandler := handlers.NewAPITokenHandler(apiTokenService)
	healthCheckHandler := handlers.NewHealthCheckHandler(healthCheckService)
	auditHandler := handlers.NewAuditHandler(auditService)

	// API group. All the routes in this group will have the prefix "/api"
	api := app.Group("/api", DBMiddleware(dbConn))

	// Health Check Endpoint (No Auth Required)
	app.Get("/v1/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
		})
	})

	// Register API Token Middleware BEFORE JWT
	api.Use(APITokenMiddleware(apiTokenService))

	// Rate limiter for authentication routes
	authLimiter := limiter.New(limiter.Config{
		Max:        5,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Too many login attempts. Please try again later.",
			})
		},
	})

	api.Post("/login", authLimiter, userHandler.Login)
	api.Post("/logout", userHandler.Logout)
	api.Post("/auth/ldap", authLimiter, userHandler.LoginLDAP)
	api.Get("/auth/oidc/login", authLimiter, userHandler.LoginOIDC)
	api.Get("/auth/oidc/callback", userHandler.CallbackOIDC)

	api.Use(jwtware.New(jwtware.Config{
		SigningKey: jwtware.SigningKey{
			JWTAlg: jwtware.RS256,
			Key:    publicKey,
		},
		TokenLookup: "cookie:jwt,header:Authorization",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			fmt.Println("JWT Error:", err)
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Unauthorized",
			})
		},
		ContextKey: "user",
		Filter: func(c *fiber.Ctx) bool {
			// Skip JWT check if API Token authentication was successful
			return c.Locals("user_record") != nil
		},
	}))

	// endpoint for getting the current user
	api.Get("/me", userHandler.GetMe)
	api.Get("/me/history", reservationHistoryHandler.GetUserHistory)
	api.Get("/resources/:id/history", reservationHistoryHandler.GetResourceHistory)

	// endpoint for resource handling
	api.Get("/resources", resourceHanlder.GetResources)
	api.Get("/resources/with-status", resourceHanlder.GetAllResourcesWithStatus)
	api.Get("/resources/:id", resourceHanlder.GetResourceByID)
	api.Get("/resources/:id/with-status", resourceHanlder.GetResourceWithStatus)
	api.Post("/resources", resourceHanlder.CreateResource)
	api.Patch("/resources/:id", resourceHanlder.UpdateResource)
	api.Delete("/resources/:id", resourceHanlder.DeleteResource)

	// Health Check endpoints
	api.Get("/resources/:id/health/config", healthCheckHandler.GetHealthConfig)
	api.Put("/resources/:id/health/config", AdminMiddleware(), healthCheckHandler.UpsertHealthConfig)
	api.Delete("/resources/:id/health/config", AdminMiddleware(), healthCheckHandler.DeleteHealthConfig)
	api.Get("/resources/:id/health/status", healthCheckHandler.GetHealthStatus)
	api.Get("/resources/:id/health/history", healthCheckHandler.GetHealthHistory)
	api.Post("/resources/:id/health/check", healthCheckHandler.TriggerHealthCheck)

	// Maintenance endpoints
	api.Put("/resources/:id/maintenance", resourceHanlder.SetMaintenanceMode)
	api.Get("/resources/:id/maintenance/history", resourceHanlder.GetMaintenanceHistory)

	// Settings Handlers (Admin)
	admin := api.Group("/admin", AdminMiddleware())
	admin.Get("/settings", settingsHandler.GetSettings)
	admin.Put("/settings", settingsHandler.UpdateSetting)
	admin.Get("/audit-logs", auditHandler.GetAuditLogs)

	// Backup & Restore (Admin Only)
	admin.Get("/backup", backupHandler.CreateBackup)
	admin.Post("/restore", backupHandler.RestoreBackup)

	// Webhooks (Admin Only)
	webhooks := api.Group("/webhooks", AdminMiddleware())
	webhooks.Get("/", webhookHandler.ListWebhooks)
	webhooks.Post("/", webhookHandler.CreateWebhook)
	webhooks.Put("/:id", webhookHandler.UpdateWebhook)
	webhooks.Delete("/:id", webhookHandler.DeleteWebhook)
	webhooks.Get("/:id/logs", webhookHandler.GetWebhookLogs)

	// Secrets (Admin Only)
	secrets := api.Group("/secrets", AdminMiddleware())
	secrets.Get("/", secretHandler.ListSecrets)
	secrets.Post("/", secretHandler.CreateSecret)
	secrets.Put("/:id", secretHandler.UpdateSecret)
	secrets.Delete("/:id", secretHandler.DeleteSecret)

	// Resource Webhooks (Admin Only)
	api.Post("/resources/:resourceId/webhooks", AdminMiddleware(), webhookHandler.AddResourceWebhook)
	api.Delete("/resources/:resourceId/webhooks/:webhookId", AdminMiddleware(), webhookHandler.RemoveResourceWebhook)
	api.Get("/resources/:resourceId/webhooks", webhookHandler.GetResourceWebhooks)

	// Reservations: Manage resource reservations
	api.Get("/reservations", reservationHandler.GetUserReservations)
	api.Post("/reservations", reservationHandler.CreateReservation)
	api.Post("/reservations/timed", reservationHandler.CreateTimedReservation)
	api.Get("/reservations/:id", reservationHandler.GetReservation)
	api.Patch("/reservations/:id/activate", AdminMiddleware(), reservationHandler.ActivateReservation)
	api.Patch("/reservations/:id/complete", reservationHandler.CompleteReservation)
	api.Patch("/reservations/:id/cancel", reservationHandler.CancelReservation)

	// Admin: Cancel all reservations for a resource
	api.Delete("/admin/resources/:id/reservations", reservationHandler.CancelAllReservations)

	// Resource-specific reservation endpoints
	api.Get("/resources/:resourceId/reservations", reservationHandler.GetResourceReservations)
	api.Get("/resources/:resourceId/queue/next", reservationHandler.GetNextInQueue)
	api.Get("/resources/:resourceId/queue/position", reservationHandler.GetUserQueuePosition)
	api.Get("/resources/:resourceId/queue", reservationHandler.GetQueueForResource)
	api.Post("/resources/:resourceId/queue/process", reservationHandler.ProcessQueue)

	// Admin-only reservation endpoints
	api.Get("/admin/reservations", reservationHandler.GetAllReservations)

	// Users: Only admin users can access these routes
	api.Get("/users", userHandler.GetUsers) // Read is fine for dropdowns
	api.Get("/users/find", userHandler.GetUser)
	api.Post("/users", AdminMiddleware(), userHandler.CreateUser)
	api.Patch("/users/:id", AdminMiddleware(), userHandler.UpdateUser)
	api.Delete("/users/:id", AdminMiddleware(), userHandler.DeleteUser)

	// Spaces: Admin only for Write, Authenticated for Read
	api.Get("/spaces", spaceHandler.GetSpaces)
	api.Get("/spaces/:id", spaceHandler.GetSpace)
	api.Post("/spaces", AdminMiddleware(), spaceHandler.CreateSpace)
	api.Patch("/spaces/:id", AdminMiddleware(), spaceHandler.UpdateSpace)
	api.Delete("/spaces/:id", AdminMiddleware(), spaceHandler.DeleteSpace)

	// Space Permissions
	api.Get("/spaces/:id/permissions", spaceHandler.GetSpacePermissions)
	api.Post("/spaces/:id/permissions", spaceHandler.AddPermission)
	api.Delete("/spaces/:id/permissions", spaceHandler.RemovePermission)

	// Access Groups (Admin Only)
	groups := api.Group("/groups", AdminMiddleware())
	groups.Get("/", groupHandler.ListGroups)
	groups.Post("/", groupHandler.CreateGroup)
	groups.Get("/:id", groupHandler.GetGroup) // Get single group details
	groups.Put("/:id", groupHandler.UpdateGroup)
	groups.Delete("/:id", groupHandler.DeleteGroup)

	// Group Members
	groups.Get("/:id/members", groupHandler.ListGroupMembers)
	groups.Post("/:id/members", groupHandler.AddUserToGroup)
	groups.Delete("/:id/members/:userId", groupHandler.RemoveUserFromGroup)

	// User Channel Config
	admin.Put("/users/:id/channel-config", userHandler.UpdateChannelConfig)
	api.Put("/me/channel-config", userHandler.UpdateChannelConfig)
	api.Post("/me/test-email", userHandler.TestEmailConfig)
	api.Post("/user/password", userHandler.HandleChangePassword)

	// User Preferences
	api.Get("/me/preferences", preferenceHandler.GetPreferences)
	api.Put("/me/preferences", preferenceHandler.UpdatePreference)

	// Realtime Events (SSE)
	api.Get("/events", realtimeHandler.HandleSSE)

	// API Tokens
	api.Get("/tokens", apiTokenHandler.ListTokens)
	api.Post("/tokens", apiTokenHandler.GenerateToken)
	api.Delete("/tokens/:id", apiTokenHandler.RevokeToken)

	// Swagger Documentation
	app.Get("/swagger/*", fiberSwagger.WrapHandler)
}
