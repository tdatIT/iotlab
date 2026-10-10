// Package svcerr defines the service error model returned by the application and transport layers, plus the Echo error handler
// that renders it.
package svcerr

import (
	"net/http"

	"github.com/tdatIT/iotlab-service/internal/domain/enums"
)

// Predefined service errors. They are pointer singletons — compare them with errors.Is, never by value.
var (
	ErrBadRequest = &Error{
		Status:  http.StatusBadRequest,
		Code:    enums.InvalidArgument,
		Message: "Bad request",
	}

	ErrInvalidIDParam = &Error{
		Status:  http.StatusBadRequest,
		Code:    enums.InvalidArgument,
		Message: "Invalid id param",
	}

	ErrPermissionDenied = &Error{
		Status:  http.StatusForbidden,
		Code:    enums.PermissionDenied,
		Message: "Permission denied",
	}

	ErrNotFound = &Error{
		Status:  http.StatusNotFound,
		Code:    enums.NotFound,
		Message: "Not found",
	}

	ErrRequestTimeout = &Error{
		Status:  http.StatusRequestTimeout,
		Code:    enums.DeadlineExceeded,
		Message: "Request timeout",
	}

	ErrAlreadyExists = &Error{
		Status:  http.StatusConflict,
		Code:    enums.AlreadyExists,
		Message: "Already exists",
	}

	ErrUnauthenticated = &Error{
		Status:  http.StatusUnauthorized,
		Code:    enums.Unauthenticated,
		Message: "Unauthorized",
	}

	ErrRecordNotFound = &Error{
		Status:  http.StatusNotFound,
		Code:    enums.RecordNotFound,
		Message: "Record does not exist",
	}

	ErrInvalidParameters = &Error{
		Status:  http.StatusBadRequest,
		Code:    enums.InvalidArgument,
		Message: "Invalid parameters",
	}

	ErrTooManyRequests = &Error{
		Status:  http.StatusTooManyRequests,
		Code:    enums.ResourceExhausted,
		Message: "Too Many Requests",
	}

	ErrInvalidFilter = &Error{
		Status:  http.StatusBadRequest,
		Code:    enums.InvalidArgument,
		Message: "Invalid filters",
	}

	ErrMethodNotAllowed = &Error{
		Status:  http.StatusMethodNotAllowed,
		Code:    enums.MethodNotAllowed,
		Message: "Method not allowed",
	}

	ErrInternalServer = &Error{
		Status:  http.StatusInternalServerError,
		Code:    enums.Internal,
		Message: "Internal server error",
	}

	ErrNotImplemented = &Error{
		Status:  http.StatusNotImplemented,
		Code:    enums.Unimplemented,
		Message: "Not implemented",
	}

	ErrServiceUnavailable = &Error{
		Status:  http.StatusServiceUnavailable,
		Code:    enums.Unavailable,
		Message: "Service unavailable",
	}
)
