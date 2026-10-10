package server

import (
	"context"
	"log/slog"
	"time"

	echoprometheus "github.com/labstack/echo-prometheus"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"github.com/oklog/ulid/v2"

	"github.com/tdatIT/iotlab-service/config"
	slogConfig "github.com/tdatIT/iotlab-service/pkgs/logger"
	"github.com/tdatIT/iotlab-service/pkgs/utilities/svcerr"
	"github.com/tdatIT/iotlab-service/pkgs/utilities/validate"
)

const WarningLatency = 250 * time.Millisecond

func newHTTPServer(cfg *config.AppConfig) *echo.Echo {
	e := echo.New()
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))
	e.Use(middleware.ContextTimeout(cfg.Server.CtxDefaultTimeout))
	e.Use(middleware.RequestIDWithConfig(middleware.RequestIDConfig{
		Generator: func() string { return ulid.Make().String() },
	}))
	e.Validator = validate.GetValidator()

	e.HTTPErrorHandler = svcerr.ErrorHandlerEchoFn
	e.Logger = slogConfig.NewSlogLogger(
		&slogConfig.SlogConfig{
			Format:      cfg.Logger.Format,
			ServiceName: cfg.Server.Name,
		},
		false,
	)

	if cfg.RateLimit.Enabled {
		store := middleware.NewRateLimiterMemoryStoreWithConfig(
			middleware.RateLimiterMemoryStoreConfig{
				Rate:      cfg.RateLimit.Rate,
				Burst:     cfg.RateLimit.Burst,
				ExpiresIn: cfg.RateLimit.ExpiresIn,
			},
		)
		e.Use(middleware.RateLimiter(store))
	}

	e.Use(echoprometheus.NewMiddlewareWithConfig(echoprometheus.MiddlewareConfig{
		Skipper: func(c *echo.Context) bool {
			p := c.Request().URL.Path
			return p == "/metrics" || p == "/liveness" || p == "/readiness" || p == "/"
		},
	}))

	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:       true,
		LogURI:          true,
		HandleError:     true,
		LogLatency:      true,
		LogMethod:       true,
		LogURIPath:      true,
		LogRequestID:    true,
		LogResponseSize: true,
		LogUserAgent:    true,
		LogRemoteIP:     true,
		Skipper: func(c *echo.Context) bool {
			p := c.Request().URL.Path
			return p == "/metrics" || p == "/liveness" || p == "/readiness" || p == "/"
		},
		LogValuesFunc: func(_ *echo.Context, v middleware.RequestLoggerValues) error {
			attrs := []slog.Attr{
				slog.Int("status", v.Status),
				slog.String("method", v.Method),
				slog.String("uri", v.URI),
				slog.String("latency", time.Duration(v.Latency.Milliseconds()).String()+"ms"),
				slog.Int64("response_size", v.ResponseSize),
				slog.String("remote_ip", v.RemoteIP),
				slog.String("user_agent", v.UserAgent),
			}

			if v.Error != nil {
				attrs = append(attrs, slog.String("error", v.Error.Error()))
				slog.LogAttrs(context.Background(), slog.LevelError, "request error", attrs...)
			} else if v.Latency > WarningLatency {
				slog.LogAttrs(context.Background(), slog.LevelWarn, "request slow", attrs...)
			}

			return nil
		},
	}))

	return e
}