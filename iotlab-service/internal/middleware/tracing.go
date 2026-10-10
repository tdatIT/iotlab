// Package middleware holds the Echo middleware wired in internal/http.go.
package middleware

import (
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.30.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/tdatIT/iotlab-service/pkgs/utilities/svcerr"
)

const tracerName = "iotlab-service/internal/middleware"

// TracingMiddleware starts an OTel span per request, propagating any incoming trace context, and records the resolved svcerr
// status on the span.
func TracingMiddleware() echo.MiddlewareFunc {
	tracer := otel.Tracer(tracerName)
	propagator := otel.GetTextMapPropagator()

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			req := c.Request()
			ctx := propagator.Extract(req.Context(), propagation.HeaderCarrier(req.Header))

			route := c.RouteInfo().Path
			spanName := route
			if spanName == "" {
				spanName = req.Method
			}

			ctx, span := tracer.Start(ctx, spanName,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					semconv.HTTPRequestMethodKey.String(req.Method),
					semconv.URLPath(req.URL.Path),
					semconv.URLScheme(c.Scheme()),
					semconv.HTTPRoute(route),
					semconv.ServerAddress(req.Host),
					semconv.ClientAddress(c.RealIP()),
					semconv.UserAgentOriginal(req.UserAgent()),
				),
			)
			defer span.End()

			c.SetRequest(req.WithContext(ctx))

			err := next(c)

			status := responseStatus(c, err)
			span.SetAttributes(semconv.HTTPResponseStatusCode(status))
			if err != nil {
				span.RecordError(err)
			}
			if status >= http.StatusInternalServerError {
				span.SetStatus(codes.Error, http.StatusText(status))
			}

			return err
		}
	}
}

func responseStatus(c *echo.Context, err error) int {
	if err == nil {
		if resp, uErr := echo.UnwrapResponse(c.Response()); uErr == nil && resp.Status != 0 {
			return resp.Status
		}
		return http.StatusOK
	}

	if svcErr, ok := errors.AsType[*svcerr.Error](err); ok {
		return svcErr.Status
	}

	if echoErr, ok := errors.AsType[*echo.HTTPError](err); ok {
		return echoErr.Code
	}

	if _, ok := errors.AsType[validator.ValidationErrors](err); ok {
		return http.StatusBadRequest
	}

	return http.StatusInternalServerError
}
