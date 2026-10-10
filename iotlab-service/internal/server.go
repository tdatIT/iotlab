// Package server wires the whole service together: config, logging, infrastructure clients, health probes, HTTP routes and the
// worker group.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/tdatIT/iotlab-service/config"
	deviceapp "github.com/tdatIT/iotlab-service/internal/app/device"
	devicehandler "github.com/tdatIT/iotlab-service/internal/handler/device"
	"github.com/tdatIT/iotlab-service/internal/infras/mqttpub"
	devicerepos "github.com/tdatIT/iotlab-service/internal/infras/repository/device"
	sensorrepos "github.com/tdatIT/iotlab-service/internal/infras/repository/sensor"
	telemetryrepos "github.com/tdatIT/iotlab-service/internal/infras/repository/telemetry"
	"github.com/tdatIT/iotlab-service/internal/router"
	"github.com/tdatIT/iotlab-service/internal/worker"
	"github.com/tdatIT/iotlab-service/internal/worker/event"
	"github.com/tdatIT/iotlab-service/migrations"
	"github.com/tdatIT/iotlab-service/pkgs/database"
	"github.com/tdatIT/iotlab-service/pkgs/logger"
	mqttpkg "github.com/tdatIT/iotlab-service/pkgs/mqtt"
	"github.com/tdatIT/iotlab-service/pkgs/probe"
)

// Server wires config, logging, DB, MQTT, health probes, the worker group and every registered domain.
type Server struct {
	cfg         *config.AppConfig
	echo        *echo.Echo
	db          database.DB
	mqttClient  mqttpkg.Client
	publisher   mqttpub.Publisher
	workerGroup *worker.Group
}

// NewServer loads the config and builds every component the service runs.
func NewServer() (*Server, error) {
	cfg, err := config.NewConfig()
	if err != nil {
		return nil, err
	}

	slog.SetDefault(logger.NewSlogLogger(
		&logger.SlogConfig{
			Level:       cfg.Logger.Level,
			Format:      cfg.Logger.Format,
			ServiceName: cfg.Server.Name,
		},
		true,
	))

	db, err := database.NewDBConnection(cfg)
	if err != nil {
		return nil, err
	}

	// Run versioned SQL migrations (golang-migrate) on startup when enabled. Migrations live in the embedded migrations/ package.
	// Disable this in production (DATABASE_AUTOMIGRATE=false) if migrations run out-of-band.
	if cfg.Database.AutoMigrate {
		if err := database.RunMigrations(db.SQLDB(), migrations.Files); err != nil {
			slog.Error("failed to run migrations", slog.String("error", err.Error()))
			return nil, fmt.Errorf("run migrations: %w", err)
		}
	}

	mqttCli, err := mqttpkg.NewMQTTClient(cfg)
	if err != nil {
		return nil, err
	}

	readyProbe := probe.New(3*time.Second).
		Register("postgres", probe.DBChecker(db)).
		Register("mqtt", probe.CheckerFunc(func(ctx context.Context) error { return mqttCli.Ping(ctx) }))

	publisher := mqttpub.NewPublisher(mqttCli)

	deviceApplication := deviceapp.NewDeviceApplication(
		devicerepos.NewDeviceRepository(db.Pool()),
		sensorrepos.NewSensorRepository(db.Pool()),
		telemetryrepos.NewTelemetryRepository(db.Pool()),
	)

	echoApp := newHTTPServer(cfg)
	router.RegisterRoutes(echoApp, readyProbe, cfg.RouteRateLimit, devicehandler.NewDeviceHandler(deviceApplication))

	workerGroup := worker.NewGroup()
	workerGroup.Register(
		event.NewSensorWorker(mqttCli, deviceApplication, &cfg.Workers.Sensor),
		event.NewTelemetryWorker(mqttCli, deviceApplication, &cfg.Workers.Telemetry),
		event.NewDeviceStatusWorker(mqttCli, deviceApplication, &cfg.Workers.DeviceStatus),
	)

	return &Server{
		cfg:         cfg,
		echo:        echoApp,
		db:          db,
		mqttClient:  mqttCli,
		publisher:   publisher,
		workerGroup: workerGroup,
	}, nil
}

// API returns the Echo HTTP server component.
func (s *Server) API() *echo.Echo {
	return s.echo
}

// Workers returns the Group component for starting all registered workers.
func (s *Server) Workers() *worker.Group {
	return s.workerGroup
}

// Config returns the loaded application configuration.
func (s *Server) Config() *config.AppConfig {
	return s.cfg
}

// Shutdown cleanly closes all infrastructure connections.
func (s *Server) Shutdown() {
	if s.mqttClient != nil {
		s.mqttClient.Disconnect()
	}

	if s.db != nil {
		if err := s.db.Close(); err != nil {
			slog.Error("failed to close database", slog.String("error", err.Error()))
		}
	}
}
