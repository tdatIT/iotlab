package server

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tdatIT/iotlab-service/internal/worker"
	mqttmock "github.com/tdatIT/iotlab-service/pkgs/mqtt/mock"
)

// fakeDB stands in for the database handle Shutdown closes.
type fakeDB struct{ closeErr error }

func (fakeDB) Pool() *pgxpool.Pool { return nil }
func (fakeDB) SQLDB() *sql.DB      { return nil }
func (f fakeDB) Close() error      { return f.closeErr }

// writeConfig drops a viper-readable config in an isolated working directory and points CONFIG_PATH at it.
func writeConfig(t *testing.T, body string) {
	t.Helper()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "testcfg.yml"), []byte(body), 0o600))
	t.Chdir(dir)
	t.Setenv("CONFIG_PATH", "testcfg")
}

const unreachableDBConfig = `
server:
  name: test-service
  ctxdefaulttimeout: 1s
logger:
  level: error
  format: json
database:
  host: 127.0.0.1
  port: 1
  username: u
  password: p
  database: d
  options: sslmode=disable connect_timeout=1
`

func TestNewServer_missingConfigFile_fails(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("CONFIG_PATH", "does-not-exist")

	srv, err := NewServer()

	require.Error(t, err)
	assert.Nil(t, srv)
}

func TestNewServer_unreachableDatabase_fails(t *testing.T) {
	withFreshPrometheusRegistry(t)
	writeConfig(t, unreachableDBConfig)

	srv, err := NewServer()

	require.Error(t, err)
	assert.Nil(t, srv)
}

func TestServer_accessors_exposeTheWiredComponents(t *testing.T) {
	withFreshPrometheusRegistry(t)

	cfg := baseTestConfig()
	group := worker.NewGroup()
	echoApp := newHTTPServer(cfg)
	srv := &Server{cfg: cfg, echo: echoApp, workerGroup: group}

	assert.Same(t, echoApp, srv.API())
	assert.Same(t, group, srv.Workers())
	assert.Same(t, cfg, srv.Config())
}

func TestServer_Shutdown_noComponents_isANoop(t *testing.T) {
	assert.NotPanics(t, func() { (&Server{}).Shutdown() })
}

func TestServer_Shutdown_closesEveryComponent(t *testing.T) {
	mqttCli := mqttmock.NewMockClient(t)
	mqttCli.EXPECT().Disconnect().Once()

	srv := &Server{mqttClient: mqttCli, db: fakeDB{}}

	assert.NotPanics(t, srv.Shutdown)
}

func TestServer_Shutdown_componentErrors_areLoggedNotFatal(t *testing.T) {
	mqttCli := mqttmock.NewMockClient(t)
	mqttCli.EXPECT().Disconnect().Once()

	srv := &Server{mqttClient: mqttCli, db: fakeDB{closeErr: errors.New("db close failed")}}

	assert.NotPanics(t, srv.Shutdown)
}
