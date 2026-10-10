package database

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	migratepg "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// RunMigrations applies all pending "up" migrations from fsys against the given database using golang-migrate. It reuses the
// already-open *sql.DB, so it shares GORM's connection pool and inherits the connection's search_path (golang-migrate's
// schema_migrations table lands in that same schema).
//
// It is a no-op when the schema is already up to date.
func RunMigrations(sqlDB *sql.DB, fsys fs.FS) error {
	// The embedded FS holds the .sql files at its root.
	src, err := iofs.New(fsys, ".")
	if err != nil {
		return fmt.Errorf("load migration source: %w", err)
	}

	driver, err := migratepg.WithInstance(sqlDB, &migratepg.Config{})
	if err != nil {
		return fmt.Errorf("init migrate driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "postgres", driver)
	if err != nil {
		return fmt.Errorf("init migrator: %w", err)
	}
	// Intentionally NOT calling m.Close(): WithInstance wraps the shared *sql.DB and Close would shut the pool that the caller
	// (Server) owns.

	switch err := m.Up(); {
	case errors.Is(err, migrate.ErrNoChange):
		slog.Info("no new database migrations")
	case err != nil:
		return fmt.Errorf("apply migrations: %w", err)
	default:
		slog.Info("database migrations applied")
	}

	return nil
}
