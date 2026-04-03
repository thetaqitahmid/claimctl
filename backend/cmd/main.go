package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/thetaqitahmid/claimctl/internal/connection"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/logger"
	"github.com/thetaqitahmid/claimctl/internal/server"
	"github.com/thetaqitahmid/claimctl/internal/services"
	"github.com/thetaqitahmid/claimctl/internal/telemetry"
	"github.com/thetaqitahmid/claimctl/internal/utils"

	// Import docs package to register swagger
	_ "github.com/thetaqitahmid/claimctl/docs"
)

// @title claimctl API
// @version 0.1.0
// @description API documentation for claimctl resource management system.
// @termsOfService http://swagger.io/terms/

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:3000
// @BasePath /api

func main() {
	// Initialize structured logging
	env := utils.GetEnv("GO_ENV", "development")
	logger.Init(env)
	slog.Info("Starting claimctl", "env", env)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Initialize Telemetry (Tracing & Metrics)
	shutdownTracer, err := telemetry.InitTracer(ctx, "claimctl-backend")
	if err != nil {
		slog.Error("Failed to initialize tracer", "error", err)
	}
	defer func() {
		if err := shutdownTracer(ctx); err != nil {
			slog.Error("failed to shutdown tracer", "error", err)
		}
	}()

	shutdownMeter, err := telemetry.InitMeter(ctx, "claimctl-backend")
	if err != nil {
		slog.Error("Failed to initialize meter", "error", err)
	}
	defer func() {
		if err := shutdownMeter(ctx); err != nil {
			slog.Error("failed to shutdown meter", "error", err)
		}
	}()

	session, err := connection.CreateDBSession()
	if err != nil {
		slog.Error("Failed to create DB session", "error", err)
		panic(err)
	}
	defer session.Close()

	// Initialize services
	queries := db.New(session)
	encryptionKey, err := utils.LoadOrGenerateKey("APP_ENCRYPTION_KEY", "./keys/app.key")
	if err != nil {
		slog.Error("Failed to initialize encryption key", "error", err)
		panic(err)
	}
	settingsService := services.NewSettingsService(queries, encryptionKey)

	privateKey, publicKey, err := settingsService.GetOrGenerateJWTKeys(ctx)
	if err != nil {
		slog.Error("Failed to generate/load JWT keys", "error", err)
		panic(err)
	}

	s := server.NewServer(ctx, session, privateKey, publicKey, encryptionKey)
	err = s.Start(ctx, ":3000")
	if err != nil {
		slog.Error("Server failed to start", "error", err)
		panic(err)
	}
}
