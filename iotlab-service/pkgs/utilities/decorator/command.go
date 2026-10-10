// Package decorator defines the generic CQRS handler interfaces every use case is declared over.
package decorator

import "context"

// CommandHandler is a write use case that returns no payload to the caller.
type CommandHandler[T any] interface {
	Handle(ctx context.Context, req T) error
}

// ApplyCommandDecorator wraps base so cross-cutting behaviour can be added around every command without touching the use case
// itself.
func ApplyCommandDecorator[T any](base CommandHandler[T]) CommandHandler[T] {
	return &commandDecorator[T]{base: base}
}

type commandDecorator[req any] struct {
	base CommandHandler[req]
}

func (d *commandDecorator[req]) Handle(ctx context.Context, input req) error {
	// Pre-processing logic can be added here,
	err := d.base.Handle(ctx, input)
	if err != nil {
		// Handle error if needed
		return err
	}

	// Post-processing logic can be added here
	return nil
}

// CommandReturnHandler is a write use case that returns a payload E.
type CommandReturnHandler[T any, E any] interface {
	Handle(ctx context.Context, req T) (E, error)
}

// ApplyCommandReturnDecorator is ApplyCommandDecorator for commands that return a payload.
func ApplyCommandReturnDecorator[T any, E any](base CommandReturnHandler[T, E]) CommandReturnHandler[T, E] {
	return &commandReturnDecorator[T, E]{base: base}
}

type commandReturnDecorator[T any, E any] struct {
	base CommandReturnHandler[T, E]
}

func (d *commandReturnDecorator[T, E]) Handle(ctx context.Context, req T) (E, error) {
	// Pre-processing logic can be added here,
	result, err := d.base.Handle(ctx, req)
	if err != nil {
		// Handle error if needed
		return result, err
	}

	// Post-processing logic can be added here
	return result, nil
}
