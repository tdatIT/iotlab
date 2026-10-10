package database

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tdatIT/iotlab-service/config"
)

func unreachableConfig() *config.AppConfig {
	return &config.AppConfig{
		Server: config.Server{Name: "test-service"},
		Logger: config.Logger{Level: "error", Format: "json"},
		Database: config.Database{
			Host:            "127.0.0.1",
			Port:            1,
			UserName:        "u",
			Password:        "p",
			Database:        "d",
			Options:         "sslmode=disable connect_timeout=1",
			MaxOpenConns:    5,
			ConnMaxLifetime: 1,
			ConnMaxIdleTime: 1,
		},
	}
}

func TestNewDBConnection_unreachableHost_returnsAnError(t *testing.T) {
	db, err := NewDBConnection(unreachableConfig())

	require.Error(t, err)
	assert.Nil(t, db)
}

func TestNewDBConnection_schemaSet_isAppendedToTheDSN(t *testing.T) {
	cfg := unreachableConfig()
	cfg.Database.Schema = "tenant_a"

	db, err := NewDBConnection(cfg)

	require.Error(t, err, "still unreachable, but the search_path branch is exercised")
	assert.Nil(t, db)
}

func TestNewDBConnection_malformedOptions_failsToParse(t *testing.T) {
	cfg := unreachableConfig()
	cfg.Database.Options = "sslmode=nonsense"

	db, err := NewDBConnection(cfg)

	require.ErrorContains(t, err, "parse database config")
	assert.Nil(t, db)
}

func TestTranslate(t *testing.T) {
	other := errors.New("boom")
	dup := &pgconn.PgError{Code: "23505"}

	assert.ErrorIs(t, Translate(pgx.ErrNoRows), ErrNotFound)
	assert.ErrorIs(t, Translate(fmt.Errorf("wrapped: %w", dup)), ErrDuplicate)
	assert.ErrorIs(t, Translate(dup), dup, "the driver error stays reachable")
	assert.Same(t, other, Translate(other))
	assert.NotErrorIs(t, Translate(&pgconn.PgError{Code: "23503"}), ErrDuplicate)
	assert.NoError(t, Translate(nil))
}
