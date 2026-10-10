//go:build e2e

// Package e2e holds the end-to-end suite: it boots the real service (config, Postgres, Redis, MQTT,
// routes) and drives it over HTTP. Only critical flows belong here — unit tests cover the branches.
//
// Run it with the skill script, which brings the dependencies up and gates API coverage:
//
//	bash .agents/skills/service-test/scripts/e2e.sh
package e2e

import (
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	server "github.com/tdatIT/iotlab-service/internal"
)

// srv is the live test server every test in this package talks to.
var srv *httptest.Server

// TestMain boots the service once for the whole suite and tears it down afterwards.
func TestMain(m *testing.M) {
	if err := chdirToModuleRoot(); err != nil {
		fmt.Fprintln(os.Stderr, "e2e: locate module root:", err)
		os.Exit(1)
	}
	if os.Getenv("CONFIG_PATH") == "" {
		// config.NewConfig resolves the file relative to the working directory.
		_ = os.Setenv("CONFIG_PATH", "config/config")
	}

	app, err := server.NewServer()
	if err != nil {
		fmt.Fprintln(os.Stderr, "e2e: NewServer failed — are postgres/redis/emqx5 up?:", err)
		os.Exit(1)
	}

	srv = httptest.NewServer(app.API())

	code := m.Run()

	srv.Close()
	app.Shutdown()
	os.Exit(code)
}

// chdirToModuleRoot walks up from the test's working directory to the directory holding go.mod, so
// the service loads its config and migrations exactly as it does in production.
func chdirToModuleRoot() error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return os.Chdir(dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return fmt.Errorf("go.mod not found above %q", dir)
		}
		dir = parent
	}
}
