package decorator

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type fakeCommand struct {
	called bool
	err    error
}

func (f *fakeCommand) Handle(_ context.Context, _ string) error {
	f.called = true
	return f.err
}

type fakeCommandReturn struct {
	called bool
	result int
	err    error
}

func (f *fakeCommandReturn) Handle(_ context.Context, _ string) (int, error) {
	f.called = true
	return f.result, f.err
}

type fakeQuery struct {
	called bool
	result string
	err    error
}

func (f *fakeQuery) Handle(_ context.Context, _ int) (string, error) {
	f.called = true
	return f.result, f.err
}

func TestApplyCommandDecorator(t *testing.T) {
	base := &fakeCommand{}
	decorated := ApplyCommandDecorator[string](base)
	require.NoError(t, decorated.Handle(context.Background(), "ok"))
	require.True(t, base.called)

	base = &fakeCommand{err: errors.New("fail")}
	decorated = ApplyCommandDecorator[string](base)
	require.Error(t, decorated.Handle(context.Background(), "err"))
}

func TestApplyCommandReturnDecorator(t *testing.T) {
	base := &fakeCommandReturn{result: 10}
	decorated := ApplyCommandReturnDecorator[string, int](base)
	got, err := decorated.Handle(context.Background(), "ok")
	require.NoError(t, err)
	require.Equal(t, 10, got)
	require.True(t, base.called)

	base = &fakeCommandReturn{err: errors.New("fail")}
	decorated = ApplyCommandReturnDecorator[string, int](base)
	_, err = decorated.Handle(context.Background(), "err")
	require.Error(t, err)
}

func TestApplyQueryDecorator(t *testing.T) {
	base := &fakeQuery{result: "ok"}
	decorated := ApplyQueryDecorator[int, string](base)
	got, err := decorated.Handle(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, "ok", got)
	require.True(t, base.called)

	base = &fakeQuery{err: errors.New("fail")}
	decorated = ApplyQueryDecorator[int, string](base)
	_, err = decorated.Handle(context.Background(), 2)
	require.Error(t, err)
}
