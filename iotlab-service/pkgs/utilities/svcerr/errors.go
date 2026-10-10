package svcerr

// Error is the single error type crossing the application and transport layers. Status is the HTTP status to respond with, Code
// the numeric service code from enums, and InternalErrorMessage the raw cause kept out of the response body.
type Error struct {
	Status               int
	InternalErrorMessage string
	Code                 int    `json:"code"`
	Message              string `json:"message"`

	// origin is the error WithError derived this one from, cause the raw error it attached. Both are set together and reached
	// through Unwrap.
	origin error
	cause  error
}

// Error returns the internal cause when one was attached, otherwise the public message.
func (e *Error) Error() string {
	if e.InternalErrorMessage != "" {
		return e.InternalErrorMessage
	}
	return e.Message
}

// WithError returns a copy of e carrying err as its internal cause. The receiver is never mutated, so it is safe to call on the
// package-level singletons in common_err.go.
//
// errors.Is on the result matches both the singleton it came from and err.
func (e *Error) WithError(err error) error {
	derived := *e
	derived.InternalErrorMessage = err.Error()
	derived.origin = e
	derived.cause = err
	return &derived
}

// Unwrap exposes the service error this one was derived from together with the raw cause attached by WithError.
func (e *Error) Unwrap() []error {
	if e.origin == nil {
		return nil
	}
	return []error{e.origin, e.cause}
}
