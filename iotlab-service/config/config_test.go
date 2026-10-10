package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewConfigReadsFile(t *testing.T) {
	t.Setenv("CONFIG_PATH", "config")

	cfg, err := NewConfig()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	require.Equal(t, "iotlab-service", cfg.Server.Name)
	require.Equal(t, ":3000", cfg.Server.Port)
}

func TestNewConfig_missingFile_returnsAnError(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("CONFIG_PATH", "does-not-exist")

	cfg, err := NewConfig()

	require.Error(t, err)
	require.Nil(t, cfg)
}

func TestNewConfig_unmarshalableFile_returnsAnError(t *testing.T) {
	dir := t.TempDir()
	// server.port is a string in AppConfig; a nested map cannot decode into it.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "broken.yml"),
		[]byte("server:\n  port:\n    nested: value\n"), 0o600))
	t.Chdir(dir)
	t.Setenv("CONFIG_PATH", "broken")

	cfg, err := NewConfig()

	require.Error(t, err)
	require.Nil(t, cfg)
}

func TestNewConfig_defaultPath_isUsedWhenTheEnvVarIsEmpty(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("CONFIG_PATH", "")

	cfg, err := NewConfig()

	require.Error(t, err, "nothing lives at the default /config/config path in a test")
	require.Nil(t, cfg)
}
