package connection

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thetaqitahmid/claimctl/internal/db"
	"github.com/thetaqitahmid/claimctl/internal/services"
	"github.com/thetaqitahmid/claimctl/internal/utils"
)

type DBConn = pgxpool.Pool

func DatabaseURL() string {
	if configured := utils.GetEnv("DATABASE_URL", ""); configured != "" {
		return configured
	}
	address := net.JoinHostPort(utils.GetEnv("DB_HOST", ""), utils.GetEnv("DB_PORT", "5432"))
	query := url.Values{"sslmode": []string{utils.GetEnv("SSL_MODE", "disable")}}
	return (&url.URL{Scheme: "postgresql", User: url.UserPassword(utils.GetEnv("DB_USER", ""), utils.GetEnv("DB_PASSWORD", "")), Host: address, Path: "/" + utils.GetEnv("DB_NAME", ""), RawQuery: query.Encode()}).String()
}

// CreateDBSession initializes a DB connection and returns the db session
func CreateDBSession() (*DBConn, error) {
	dataSourceName := DatabaseURL()
	config, err := pgxpool.ParseConfig(dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("error parsing database config")
	}

	// Wrap config with otelpgx for instrumentation
	config.ConnConfig.Tracer = otelpgx.NewTracer()

	// Set connection pool parameters
	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	dbContext := context.Background()
	dbContext, cancel := context.WithTimeout(dbContext, 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(dbContext, config)
	if err != nil {
		return nil, fmt.Errorf("error creating database connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(dbContext); err != nil {
		pool.Close()
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}

	// Run migrations
	if err := RunMigrations("file://./migrations", dataSourceName); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	// Ensure admin user exists
	if err := ensureAdminExists(pool); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ensure admin user exists: %w", err)
	}

	// Ensure default space exists
	if err := ensureDefaultSpaceExists(pool); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ensure default space exists: %w", err)
	}

	return pool, nil
}

// ensureAdminExists creates a default admin user if no admin users exist
func ensureAdminExists(pool *DBConn) error {
	store := db.NewStore(pool)
	userService := services.NewUserService(store)

	ctx := context.Background()
	return userService.EnsureAdminExists(ctx)
}

// ensureDefaultSpaceExists creates the default space if it doesn't exist
func ensureDefaultSpaceExists(pool *DBConn) error {
	store := db.NewStore(pool)
	spaceService := services.NewSpaceService(store)

	ctx := context.Background()
	return spaceService.EnsureDefaultSpaceExists(ctx)
}
