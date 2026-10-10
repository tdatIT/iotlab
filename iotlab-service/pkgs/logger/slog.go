// Package logger builds the stdlib slog handler used as the service-wide default logger.
package logger

import (
	"log/slog"
	"os"
	"strings"
)

// SlogConfig configures the slog handler. Empty fields fall back to the defaults applied by applyDefaults.
type SlogConfig struct {
	Level       string
	ServiceName string
	Format      string // "json" or "console"
}

// applyDefaults applies default values to the SlogConfig if they are not set.
func applyDefaults(config *SlogConfig) *SlogConfig {
	if config == nil {
		config = &SlogConfig{}
	}

	if config.Level == "" {
		config.Level = "info"
	}

	if config.ServiceName == "" {
		config.ServiceName = "iotlab-service"
	}

	return config
}

// NewSlogLogger is the common entry point that returns a ready *slog.Logger emitting either JSON or console text output based on
// config.Format.
func NewSlogLogger(config *SlogConfig, sourceInfo bool) *slog.Logger {
	return slog.New(NewSlogHandler(config, sourceInfo))
}

// NewSlogHandler creates a slog.Handler based on the provided SlogConfig. Format "console" yields a text handler; anything else
// (default) yields JSON.
func NewSlogHandler(config *SlogConfig, sourceInfo bool) slog.Handler {
	config = applyDefaults(config)

	opts := &slog.HandlerOptions{
		Level:     parseSlogLevel(config.Level),
		AddSource: sourceInfo,
	}

	var handler slog.Handler
	switch strings.ToLower(config.Format) {
	case "console", "text":
		handler = slog.NewTextHandler(os.Stdout, opts)
	default: // default to JSON format
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	return handler.WithAttrs([]slog.Attr{
		slog.String("service_name", config.ServiceName),
	})
}

func parseSlogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	case "info":
		fallthrough
	default:
		return slog.LevelInfo
	}
}
