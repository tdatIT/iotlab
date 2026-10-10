// Package enums holds the numeric service codes returned in every API response body.
package enums

// Service codes shared by every domain.
const (
	OK = iota
	InvalidArgument
	DeadlineExceeded
	NotFound
	AlreadyExists
	Unimplemented
	MethodNotAllowed
	PermissionDenied
	Unauthenticated
	ResourceExhausted
	Internal
	RecordNotFound
	Unavailable
)
