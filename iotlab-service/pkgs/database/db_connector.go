// Package database opens the pgx connection pool, exposes it behind the DB interface, maps driver errors to sentinels, and
// runs the embedded migrations.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/tdatIT/iotlab-service/config"
)

// connectTimeout bounds the startup ping so an unreachable database fails fast.
const connectTimeout = 5 * time.Second

// Querier is the subset of *pgxpool.Pool the repositories use. Tests substitute pgxmock.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// DB is the shared database handle. Pool serves the repositories; SQLDB is a database/sql view over the same pool, used only
// by golang-migrate.
type DB interface {
	Pool() *pgxpool.Pool
	SQLDB() *sql.DB
	Close() error
}

type db struct {
	pool  *pgxpool.Pool
	sqlDB *sql.DB
}

// NewDBConnection opens the Postgres pool with the limits from config and pings it once.
func NewDBConnection(cfg *config.AppConfig) (DB, error) {
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s %s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.UserName,
		cfg.Database.Password,
		cfg.Database.Database,
		cfg.Database.Options,
	)
	if cfg.Database.Schema != "" {
		dsn += fmt.Sprintf(" search_path=%s", cfg.Database.Schema)
	}

	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}
	if cfg.Database.MaxOpenConns > 0 {
		poolCfg.MaxConns = int32(cfg.Database.MaxOpenConns)
	}
	if cfg.Database.ConnMaxLifetime > 0 {
		poolCfg.MaxConnLifetime = cfg.Database.ConnMaxLifetime
	}
	if cfg.Database.ConnMaxIdleTime > 0 {
		poolCfg.MaxConnIdleTime = cfg.Database.ConnMaxIdleTime
	}

	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("open database pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		slog.Error("error while creating db connection", slog.String("error", err.Error()))
		return nil, fmt.Errorf("ping database: %w", err)
	}

	slog.Info("db connection created",
		slog.String("host", fmt.Sprintf("%v:%v", cfg.Database.Host, cfg.Database.Port)),
		slog.String("db", cfg.Database.Database),
		slog.String("schema", cfg.Database.Schema),
	)

	return &db{pool: pool, sqlDB: stdlib.OpenDBFromPool(pool)}, nil
}

func (d *db) Pool() *pgxpool.Pool { return d.pool }

func (d *db) SQLDB() *sql.DB { return d.sqlDB }

func (d *db) Close() error {
	err := d.sqlDB.Close()
	d.pool.Close()
	return err
}
