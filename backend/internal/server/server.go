package server

import (
	"context"
	"crypto/rsa"
	"log/slog"
	"time"

	"github.com/gofiber/contrib/otelfiber/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/csrf"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/thetaqitahmid/claimctl/internal/connection"
	"github.com/thetaqitahmid/claimctl/internal/utils"
)

// Server is the main struct for the server
type Server struct {
	app *fiber.App
	db  *connection.DBConn
}

// NewServer creates a new server
func NewServer(ctx context.Context, db *connection.DBConn, privateKey *rsa.PrivateKey, publicKey *rsa.PublicKey, encryptionKey string) *Server {
	app := fiber.New()
	app.Use(otelfiber.Middleware())

	// CORS Configuration
	allowedOrigins := utils.GetEnv("CORS_ALLOWED_ORIGINS", "")

	// If CORS_ALLOWED_ORIGINS is provided, configure the CORS middleware.
	// If it is empty, we do NOT register the CORS middleware at all.
	// When CORS headers are absent, browsers automatically enforce strict Same-Origin Policy.
	if allowedOrigins != "" {
		app.Use(cors.New(cors.Config{
			AllowCredentials: true,
			AllowOrigins:     allowedOrigins,
		}))
	}

	enableCSRF := utils.GetEnvAsBool("ENABLE_CSRF", true)
	if enableCSRF {
		app.Use(csrf.New(csrf.Config{
			KeyLookup:      "cookie:csrf_",
			CookieName:     "csrf_",
			CookieSameSite: utils.GetEnv("COOKIE_SAMESITE", "Strict"),
			CookieSecure:   utils.GetEnvAsBool("COOKIE_SECURE", true),
			CookieHTTPOnly: true,
			Expiration:     1 * time.Hour,
		}))
	}

	app.Use(limiter.New(limiter.Config{
		Max:        60,
		Expiration: 30 * time.Second,
	}))
	app.Use(func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		duration := time.Since(start)

		slog.Info("HTTP Request",
			"status", c.Response().StatusCode(),
			"method", c.Method(),
			"path", c.Path(),
			"ip", c.IP(),
			"duration", duration.String(),
		)
		return err
	})

	SetupRoutes(ctx, app, db, privateKey, publicKey, encryptionKey)

	return &Server{app: app, db: db}
}

// Start starts the server
func (s *Server) Start(ctx context.Context, port string) error {
	go func() {
		<-ctx.Done()
		slog.Info("Shutting down server...")
		if err := s.app.Shutdown(); err != nil {
			slog.Error("Server shutdown failed", "error", err)
		}
	}()
	return s.app.Listen(port)
}
