package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/require"

	"github.com/tdatIT/iotlab-service/internal/domain/enums"
)

type baseResPayload struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func TestBaseResJSON(t *testing.T) {
	res := BaseRes{
		Status:  201,
		Code:    123,
		Message: "ok",
		Data:    "value",
	}

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	require.NoError(t, res.JSON(ctx))
	require.Equal(t, http.StatusOK, rec.Code)

	var payload baseResPayload
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Equal(t, res.Code, payload.Code)
	require.Equal(t, res.Message, payload.Message)

	var data string
	require.NoError(t, json.Unmarshal(payload.Data, &data))
	require.Equal(t, "value", data)
}

func TestResponseDefaults(t *testing.T) {
	require.Equal(t, enums.OK, SuccessRes.Code)
	require.Equal(t, "success", SuccessRes.Message)
	require.Equal(t, enums.Internal, ErrorRes.Code)
	require.Equal(t, "internal server error", ErrorRes.Message)
}
