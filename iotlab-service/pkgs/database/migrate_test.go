package database

import (
	"database/sql"
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newUnreachableDB returns a lazily-connecting handle; it only fails once something talks to the server.
func newUnreachableDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("pgx", "host=127.0.0.1 port=1 user=u password=p dbname=d sslmode=disable connect_timeout=1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// brokenFS fails every read, standing in for an unreadable migrations bundle.
type brokenFS struct{}

func (brokenFS) Open(string) (fs.File, error) { return nil, errors.New("unreadable") }

func TestRunMigrations_unreadableSource_failsToLoadTheSource(t *testing.T) {
	err := RunMigrations(newUnreachableDB(t), brokenFS{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "load migration source")
}

func TestRunMigrations_unreachableDatabase_failsToInitTheDriver(t *testing.T) {
	fsys := fstest.MapFS{
		"000001_init.up.sql":   &fstest.MapFile{Data: []byte("CREATE TABLE t (id INTEGER);")},
		"000001_init.down.sql": &fstest.MapFile{Data: []byte("DROP TABLE t;")},
	}

	err := RunMigrations(newUnreachableDB(t), fsys)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "init migrate driver")
}
