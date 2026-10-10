package svcerr

import (
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v5"

	"github.com/tdatIT/iotlab-service/internal/domain/enums"
	"github.com/tdatIT/iotlab-service/pkgs/utilities/response"
)

// ErrorHandlerEchoFn is Echo's global error handler. It resolves, in order: *Error, validator.ValidationErrors,
// echo.HTTPStatusCoder, then falls back to 500 Internal.
func ErrorHandlerEchoFn(ctx *echo.Context, err error) {
	if resp, uErr := echo.UnwrapResponse(ctx.Response()); uErr == nil {
		if resp.Committed {
			return
		}
	}

	code := http.StatusInternalServerError
	errResp := response.BaseRes{
		Code:    enums.Internal,
		Message: "internal server error",
	}

	if customErr, ok := errors.AsType[*Error](err); ok {
		code = customErr.Status
		errResp.Code = customErr.Code
		errResp.Message = customErr.Message
	} else if _, ok := errors.AsType[validator.ValidationErrors](err); ok {
		code = http.StatusBadRequest
		errResp.Code = enums.InvalidArgument
		errResp.Message = "validation body error"
	} else {
		var sc echo.HTTPStatusCoder
		if errors.As(err, &sc) {
			if tmp := sc.StatusCode(); tmp != 0 {
				code = tmp
			}
			errResp.Code = code
			errResp.Message = http.StatusText(code)
		}
	}

	errResp.Status = code

	var cErr error
	if ctx.Request().Method == http.MethodHead {
		cErr = ctx.NoContent(code)
	} else {
		cErr = ctx.JSON(code, errResp)
	}

	if cErr != nil {
		ctx.Logger().Error("failed to send error response", "error", errors.Join(err, cErr))
	}
}
