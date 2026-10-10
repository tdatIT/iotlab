// Package response defines the envelope every HTTP response is rendered into.
package response

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/tdatIT/iotlab-service/internal/domain/enums"
)

// BaseRes is the envelope every successful and failed HTTP response is rendered into.
type BaseRes struct {
	Status  int    `json:"-"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// JSON writes the envelope with HTTP 200; the service code inside carries the domain-level outcome.
func (g *BaseRes) JSON(c *echo.Context) error {
	return c.JSON(http.StatusOK, g)
}

// Response templates. Copy one before mutating it — they are package-level values shared by every request.
var (
	SuccessRes = BaseRes{
		Status:  http.StatusOK,
		Code:    enums.OK,
		Message: "success",
		Data:    nil,
	}

	ErrorRes = BaseRes{
		Status:  http.StatusInternalServerError,
		Code:    enums.Internal,
		Message: "internal server error",
		Data:    nil,
	}
)
