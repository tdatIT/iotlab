package logger_test

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tdatIT/iotlab-service/pkgs/logger"
)

func TestNewSlogLogger_returnsALoggerHonouringTheConfiguredLevel(t *testing.T) {
	tests := map[string]struct {
		cfg        *logger.SlogConfig
		wantDebug  bool
		sourceInfo bool
	}{
		"json at info drops debug records": {
			cfg: &logger.SlogConfig{Level: "info", Format: "json", ServiceName: "svc"},
		},
		"console at debug keeps debug records": {
			cfg:        &logger.SlogConfig{Level: "debug", Format: "console", ServiceName: "svc"},
			wantDebug:  true,
			sourceInfo: true,
		},
		"nil config falls back to the defaults": {
			cfg: nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			l := logger.NewSlogLogger(tc.cfg, tc.sourceInfo)

			require.NotNil(t, l)
			assert.Equal(t, tc.wantDebug, l.Enabled(t.Context(), slog.LevelDebug))
			assert.True(t, l.Enabled(t.Context(), slog.LevelError))
		})
	}
}
