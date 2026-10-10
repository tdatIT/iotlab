package svcerr

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestErrorMessage(t *testing.T) {
	err := &Error{Message: "msg"}
	require.Equal(t, "msg", err.Error())

	ret := err.WithError(errors.New("inner"))
	require.Equal(t, "inner", ret.Error())
}

func TestWithErrorDoesNotMutateReceiver(t *testing.T) {
	ret := ErrInternalServer.WithError(errors.New("db exploded"))

	require.NotSame(t, error(ErrInternalServer), ret)
	require.Empty(t, ErrInternalServer.InternalErrorMessage)
	require.Equal(t, "Internal server error", ErrInternalServer.Error())
	require.Equal(t, "db exploded", ret.Error())
}

func TestWithErrorKeepsIdentityAndCause(t *testing.T) {
	cause := errors.New("db exploded")
	ret := ErrInternalServer.WithError(cause)

	require.ErrorIs(t, ret, ErrInternalServer)
	require.ErrorIs(t, ret, cause)
	require.NotErrorIs(t, ret, ErrRecordNotFound)

	var svcErr *Error
	require.ErrorAs(t, ret, &svcErr)
	require.Equal(t, ErrInternalServer.Status, svcErr.Status)
	require.Equal(t, ErrInternalServer.Code, svcErr.Code)
}
