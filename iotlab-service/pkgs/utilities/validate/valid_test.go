package validate

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type validateSample struct {
	Name string `validate:"required"`
}

func TestValidatorValidate(t *testing.T) {
	v := GetValidator()
	require.Error(t, v.Validate(&validateSample{}))
	require.NoError(t, v.Validate(&validateSample{Name: "ok"}))
}

func TestGetValidatorSingleton(t *testing.T) {
	v1 := GetValidator()
	v2 := GetValidator()
	require.Same(t, v1, v2)
}
