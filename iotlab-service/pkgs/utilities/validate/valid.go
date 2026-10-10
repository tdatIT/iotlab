// Package validate exposes the process-wide go-playground validator used as Echo's validator hook.
package validate

import (
	"sync"

	"github.com/go-playground/validator/v10"
)

var (
	_validator *Validator
	_once      sync.Once
)

// Validator adapts go-playground/validator to Echo's validator hook.
type Validator struct {
	Valid *validator.Validate
}

// GetValidator returns the process-wide validator, building it on first use.
func GetValidator() *Validator {
	_once.Do(func() {
		_validator = &Validator{
			Valid: validator.New(),
		}
	})
	return _validator
}

// Validate runs the struct tags of i.
func (v *Validator) Validate(i any) error {
	err := v.Valid.Struct(i)
	if err != nil {
		return err
	}
	return nil
}
