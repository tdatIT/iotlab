package svcerr

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"

	"github.com/tdatIT/iotlab-service/internal/domain/enums"
)

type errorPayload struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func TestErrorHandlerEchoFn_CustomError(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	custom := &Error{Status: 400, Code: enums.InvalidArgument, Message: "bad"}
	ErrorHandlerEchoFn(ctx, custom)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	var payload errorPayload
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Equal(t, custom.Code, payload.Code)
	require.Equal(t, custom.Message, payload.Message)
}

func TestErrorHandlerEchoFn_EchoHTTPError(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	ErrorHandlerEchoFn(ctx, echo.NewHTTPError(http.StatusNotFound, "missing"))

	require.Equal(t, http.StatusNotFound, rec.Code)
	var payload errorPayload
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Equal(t, http.StatusNotFound, payload.Code)
	require.Equal(t, http.StatusText(http.StatusNotFound), payload.Message)
}

func TestErrorHandlerEchoFn_ValidationError(t *testing.T) {
	type request struct {
		Name string `validate:"required"`
	}

	validateErr := validator.New().Struct(request{})
	require.Error(t, validateErr)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	ErrorHandlerEchoFn(ctx, validateErr)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	var payload errorPayload
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Equal(t, enums.InvalidArgument, payload.Code)
	require.Equal(t, "validation body error", payload.Message)
}

func TestErrorHandlerEchoFn_DefaultError(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	ErrorHandlerEchoFn(ctx, errors.New("boom"))

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	var payload errorPayload
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Equal(t, enums.Internal, payload.Code)
	require.Equal(t, "internal server error", payload.Message)
}

func TestErrorHandlerEchoFn_alreadyCommittedResponse_writesNothing(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	require.NoError(t, ctx.String(http.StatusAccepted, "already sent"))
	ErrorHandlerEchoFn(ctx, ErrInternalServer)

	require.Equal(t, http.StatusAccepted, rec.Code)
	require.Equal(t, "already sent", rec.Body.String())
}

func TestErrorHandlerEchoFn_headRequest_repliesWithoutABody(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodHead, "/", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	ErrorHandlerEchoFn(ctx, ErrRecordNotFound)

	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Empty(t, rec.Body.String())
}
