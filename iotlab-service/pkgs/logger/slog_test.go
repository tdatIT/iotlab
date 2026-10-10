package logger

import (
	"context"
	"testing"

	"log/slog"

	"github.com/stretchr/testify/require"
)

func TestParseSlogLevel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected slog.Level
	}{
		{name: "debug", input: "debug", expected: slog.LevelDebug},
		{name: "warn", input: "warn", expected: slog.LevelWarn},
		{name: "warning", input: "warning", expected: slog.LevelWarn},
		{name: "error", input: "error", expected: slog.LevelError},
		{name: "info", input: "info", expected: slog.LevelInfo},
		{name: "default", input: "unknown", expected: slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, parseSlogLevel(tt.input))
		})
	}
}

func TestNewSlogHandler(t *testing.T) {
	jsonHandler := NewSlogHandler(nil, false)
	require.IsType(t, &slog.JSONHandler{}, jsonHandler)
	require.True(t, jsonHandler.Enabled(context.Background(), slog.LevelInfo))

	consoleHandler := NewSlogHandler(&SlogConfig{Format: "console"}, false)
	require.IsType(t, &slog.TextHandler{}, consoleHandler)

	errHandler := NewSlogHandler(&SlogConfig{Level: "error"}, false)
	require.False(t, errHandler.Enabled(context.Background(), slog.LevelInfo))
	require.True(t, errHandler.Enabled(context.Background(), slog.LevelError))
}
