package connection

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Migration history is pinned to the public schema so it does not move when the
// connection's search_path resolves to a different schema (for example after
// migration 1 creates the claimctl schema).
const migrationsTableQuoted = `"public"."schema_migrations"`

func migrationDSN(dsn string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", fmt.Errorf("failed to parse database URL for migrations: %w", err)
	}
	if u.Scheme == "" {
		return "", fmt.Errorf("database URL for migrations must include a scheme")
	}

	query := u.Query()
	query.Set("x-migrations-table", migrationsTableQuoted)
	query.Set("x-migrations-table-quoted", "true")
	u.RawQuery = query.Encode()
	return u.String(), nil
}

func RunMigrations(migrationsPath, dsn string) error {
	migrationURL, err := migrationDSN(dsn)
	if err != nil {
		return err
	}

	m, err := migrate.New(migrationsPath, migrationURL)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			fmt.Println("Migrations: no new migrations to apply")
			return nil
		}

		var dirty migrate.ErrDirty
		if errors.As(err, &dirty) {
			return fmt.Errorf(
				"database schema is dirty at migration version %d: a previous migration was interrupted. "+
					"Confirm the objects for version %d are complete, then clear the dirty flag "+
					"(for example by running migrate force %d or setting schema_migrations.version=%d, dirty=false) "+
					"and restart the backend",
				dirty.Version, dirty.Version, dirty.Version, dirty.Version,
			)
		}

		return fmt.Errorf("failed to run migrations: %w", err)
	}

	fmt.Println("Migrations: applied successfully")
	return nil
}
