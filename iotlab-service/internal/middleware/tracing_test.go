package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	appmiddleware "github.com/tdatIT/iotlab-service/internal/middleware"
	"github.com/tdatIT/iotlab-service/pkgs/utilities/svcerr"
)

// newTracedEcho registers the middleware on a recording tracer provider and returns the exporter holding the spans it produced.
func newTracedEcho(t *testing.T) (*echo.Echo, *tracetest.InMemoryExporter) {
	t.Helper()

	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	t.Cleanup(func() { require.NoError(t, tp.Shutdown(context.Background())) })

	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() { otel.SetTracerProvider(prev) })

	e := echo.New()
	e.HTTPErrorHandler = svcerr.ErrorHandlerEchoFn
	e.Use(appmiddleware.TracingMiddleware())
	return e, exporter
}

func attrInt(t *testing.T, span tracetest.SpanStub, key attribute.Key) int64 {
	t.Helper()
	for _, a := range span.Attributes {
		if a.Key == key {
			return a.Value.AsInt64()
		}
	}
	t.Fatalf("attribute %s not found", key)
	return 0
}

func TestTracingMiddleware_successfulRequest_recordsAnOkServerSpan(t *testing.T) {
	e, exporter := newTracedEcho(t)
	e.GET("/products/:id", func(c *echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/products/1", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, "/products/:id", spans[0].Name)
	assert.Equal(t, int64(http.StatusOK), attrInt(t, spans[0], "http.response.status_code"))
	assert.NotEqual(t, codes.Error, spans[0].Status.Code)
}

func TestTracingMiddleware_incomingTraceContext_isPropagated(t *testing.T) {
	e, exporter := newTracedEcho(t)
	e.GET("/", func(c *echo.Context) error { return c.NoContent(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	e.ServeHTTP(httptest.NewRecorder(), req)

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", spans[0].SpanContext.TraceID().String())
	assert.Equal(t, "00f067aa0ba902b7", spans[0].Parent.SpanID().String())
}

func TestTracingMiddleware_errorStatusMapping(t *testing.T) {
	tests := map[string]struct {
		err        error
		wantStatus int64
		wantError  bool
	}{
		"service error keeps its own status": {
			err:        svcerr.ErrRecordNotFound,
			wantStatus: http.StatusNotFound,
		},
		"echo http error keeps its code": {
			err:        echo.NewHTTPError(http.StatusTeapot, "teapot"),
			wantStatus: http.StatusTeapot,
		},
		"validation error is a bad request": {
			err:        validator.ValidationErrors{},
			wantStatus: http.StatusBadRequest,
		},
		"unknown error is an internal server error": {
			err:        errors.New("boom"),
			wantStatus: http.StatusInternalServerError,
			wantError:  true,
		},
		"service internal error marks the span failed": {
			err:        svcerr.ErrInternalServer,
			wantStatus: http.StatusInternalServerError,
			wantError:  true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			e, exporter := newTracedEcho(t)
			e.GET("/", func(*echo.Context) error { return tc.err })

			e.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

			spans := exporter.GetSpans()
			require.Len(t, spans, 1)
			assert.Equal(t, tc.wantStatus, attrInt(t, spans[0], "http.response.status_code"))
			assert.NotEmpty(t, spans[0].Events, "the error must be recorded on the span")
			if tc.wantError {
				assert.Equal(t, codes.Error, spans[0].Status.Code)
			} else {
				assert.NotEqual(t, codes.Error, spans[0].Status.Code)
			}
		})
	}
}

func TestTracingMiddleware_unmatchedRoute_fallsBackToTheMethodAsSpanName(t *testing.T) {
	e, exporter := newTracedEcho(t)

	e.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/nope", nil))

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)
	assert.Equal(t, http.MethodGet, spans[0].Name)
}
