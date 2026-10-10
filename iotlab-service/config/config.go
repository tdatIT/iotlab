// Package config loads the service configuration from a viper config file overlaid with environment variables.
package config

import (
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// AppConfig is the root configuration decoded from the config file.
type AppConfig struct {
	Server    Server
	Database  Database
	Logger    Logger
	Adapters  Adapters
	MQTT      MQTT
	Workers   Workers
	RateLimit RateLimit
	// RouteRateLimit is applied to each API route separately: every route keeps its own per-client token buckets.
	RouteRateLimit RateLimit
}

// Adapters aggregates HTTPClient config for every outbound REST adapter this service uses. Add one field per adapter, e.g.
// `WeatherService HTTPClient`.
type Adapters struct{}

// Workers aggregates MQTT subscription config for every background worker this service runs. Add one field per worker, e.g.
// `UserEvent WorkerConfig`.
type Workers struct {
	Sensor       WorkerConfig
	Telemetry    WorkerConfig
	DeviceStatus WorkerConfig
}

// HTTPClient is the generic config type used by every outbound adapter.
type HTTPClient struct {
	BaseURL        string
	Timeout        time.Duration
	KeepAlive      time.Duration
	RetryCount     int
	RetryWait      time.Duration
	Debug          bool
	APIKey         string
	CircuitBreaker CircuitBreaker
}

// CircuitBreaker configures the resty circuit breaker for an outbound adapter. Enabled=false leaves the adapter's
// httpclient.Config.CircuitBreaker nil.
type CircuitBreaker struct {
	Enabled          bool
	FailureThreshold uint64
	SuccessThreshold uint64
	ResetTimeout     time.Duration
}

// WorkerConfig holds the MQTT subscription settings for a single worker.
type WorkerConfig struct {
	Topic string
	QoS   byte
}

// Server configures the HTTP server.
type Server struct {
	Debug             bool
	Name              string
	Version           string
	Port              string
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	CtxDefaultTimeout time.Duration
}

// Database configures the Postgres connection and its pool.
type Database struct {
	Host            string
	Port            int
	UserName        string
	Password        string
	Database        string
	Schema          string
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	Options         string
	AutoMigrate     bool
}

// Logger configures the slog handler.
type Logger struct {
	Level  string
	Format string // "json" or "console"
}

// RateLimit holds configuration for the Echo rate limiter middleware. Rate is requests-per-second per client identifier; Burst is
// the token bucket depth; ExpiresIn is how long an idle client's bucket is kept in memory.
type RateLimit struct {
	Enabled   bool
	Rate      float64
	Burst     int
	ExpiresIn time.Duration
}

// MQTT configures the broker connection.
type MQTT struct {
	Broker               string
	ClientID             string
	Username             string
	Password             string
	ConnectTimeout       time.Duration
	KeepAlive            time.Duration
	PublishTimeout       time.Duration
	ConnectRetryInterval time.Duration
}

// defaultConfigPath is used when CONFIG_PATH is not set.
const defaultConfigPath = "/config/config"

// NewConfig loads the config file named by CONFIG_PATH (or defaultConfigPath) and overlays matching environment variables.
func NewConfig() (*AppConfig, error) {
	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = defaultConfigPath
	}

	slog.Info("loading config", slog.String("path", path))

	config := AppConfig{}
	v := viper.New()
	v.SetConfigName(path)
	v.AddConfigPath(".")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		slog.Error("unable to read config file", slog.String("error", err.Error()))
		return nil, err
	}

	err := v.Unmarshal(&config)
	if err != nil {
		slog.Error("unable to decode into struct", slog.String("error", err.Error()))
		return nil, err
	}

	return &config, nil
}
