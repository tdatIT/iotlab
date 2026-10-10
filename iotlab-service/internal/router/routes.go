// Package router registers every HTTP route the service exposes.
package router

import (
	// Blank import registers the generated OpenAPI spec so the /swagger endpoint can serve it.
	_ "github.com/tdatIT/iotlab-service/docs/swagger"

	"net/http"

	echoprometheus "github.com/labstack/echo-prometheus"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"github.com/tdatIT/iotlab-service/config"
	"github.com/tdatIT/iotlab-service/internal/handler/device"
	"github.com/tdatIT/iotlab-service/pkgs/probe"
)

// RegisterRoutes wires the platform-level routes (liveness, readiness, metrics) and every domain handler this service exposes.
// Each API route gets its own rate limiter built from routeLimit.
func RegisterRoutes(
	e *echo.Echo,
	readyProbe *probe.Probe,
	routeLimit config.RateLimit,
	deviceHandler *device.Handler,
) {

	e.GET("/metrics", echoprometheus.NewHandler())
	e.GET("/liveness", func(c *echo.Context) error { return c.JSON(http.StatusOK, map[string]string{"status": "ok"}) })
	e.GET("/readiness", echo.WrapHandler(readyProbe.Handler()))
	e.GET("/swagger", func(c *echo.Context) error { return c.Redirect(http.StatusMovedPermanently, "/swagger/index.html") })
	e.GET("/swagger/*", echo.WrapHandler(httpSwagger.WrapHandler))

	api := e.Group("/api")
	v1 := api.Group("/v1")

	deviceRoute := v1.Group("/devices")
	deviceRoute.GET("", deviceHandler.ListDevices, rateLimiter(routeLimit))
	deviceRoute.GET("/:id/telemetry/current", deviceHandler.GetCurrentTelemetry, rateLimiter(routeLimit))
	deviceRoute.GET("/:id/telemetry/stats", deviceHandler.GetTelemetryStats, rateLimiter(routeLimit))
	deviceRoute.GET("/:id/sensor/current", deviceHandler.GetCurrentSensor, rateLimiter(routeLimit))
	deviceRoute.GET("/:id/sensor/stats", deviceHandler.GetSensorStats, rateLimiter(routeLimit))
	deviceRoute.GET("/:id/sensor/average-24h", deviceHandler.GetSensorAverage24h, rateLimiter(routeLimit))
	deviceRoute.GET("/:id/sensor/stats-24h", deviceHandler.GetSensorStats24h, rateLimiter(routeLimit))
}

// rateLimiter builds a limiter with its own in-memory store, so each route that calls it is throttled independently.
// It is a pass-through when cfg.Enabled is false.
func rateLimiter(cfg config.RateLimit) echo.MiddlewareFunc {
	if !cfg.Enabled {
		return func(next echo.HandlerFunc) echo.HandlerFunc { return next }
	}
	return middleware.RateLimiter(middleware.NewRateLimiterMemoryStoreWithConfig(
		middleware.RateLimiterMemoryStoreConfig{
			Rate:      cfg.Rate,
			Burst:     cfg.Burst,
			ExpiresIn: cfg.ExpiresIn,
		},
	))
}
