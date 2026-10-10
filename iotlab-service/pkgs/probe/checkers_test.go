package probe_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/alicebob/miniredis/v2"
	_ "github.com/glebarez/go-sqlite" // registers the "sqlite" database/sql driver
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tdatIT/iotlab-service/pkgs/probe"
	"github.com/tdatIT/iotlab-service/pkgs/rdclient"
)

// sqliteDB stands in for database.DB: DBChecker only pings SQLDB, so any database/sql engine will do.
type sqliteDB struct {
	sqlDB *sql.DB
}

func (s sqliteDB) Pool() *pgxpool.Pool { return nil }
func (s sqliteDB) SQLDB() *sql.DB      { return s.sqlDB }
func (s sqliteDB) Close() error        { return s.sqlDB.Close() }

func newTestORM(t *testing.T) sqliteDB {
	t.Helper()

	sqlDB, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	return sqliteDB{sqlDB: sqlDB}
}

func TestDBChecker(t *testing.T) {
	ctx := context.Background()

	t.Run("open connection passes", func(t *testing.T) {
		assert.NoError(t, probe.DBChecker(newTestORM(t)).Check(ctx))
	})

	t.Run("closed connection fails", func(t *testing.T) {
		orm := newTestORM(t)
		require.NoError(t, orm.Close())

		assert.Error(t, probe.DBChecker(orm).Check(ctx))
	})
}

func TestRedisChecker(t *testing.T) {
	ctx := context.Background()

	newClient := func(t *testing.T) (*miniredis.Miniredis, rdclient.RedisClient) {
		t.Helper()
		mr, err := miniredis.Run()
		require.NoError(t, err)
		t.Cleanup(mr.Close)

		cfg := rdclient.Config{Address: []string{mr.Addr()}}
		rc, err := rdclient.NewRedisClient(cfg)
		require.NoError(t, err)
		return mr, rc
	}

	t.Run("reachable redis passes", func(t *testing.T) {
		_, rc := newClient(t)
		assert.NoError(t, probe.RedisChecker(rc).Check(ctx))
	})

	t.Run("unreachable redis fails", func(t *testing.T) {
		mr, rc := newClient(t)
		mr.Close()

		assert.Error(t, probe.RedisChecker(rc).Check(ctx))
	})
}
