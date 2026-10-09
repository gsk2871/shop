package biz

import (
	"context"
	"errors"
	"testing"
)

type checkerFunc func(context.Context) error

func (f checkerFunc) Check(ctx context.Context) error { return f(ctx) }
func TestSystemReady(t *testing.T) {
	failure := errors.New("database unavailable")
	for _, tc := range []struct {
		name string
		err  error
	}{{"success", nil}, {"failure", failure}} {
		t.Run(tc.name, func(t *testing.T) {
			s := NewSystem(checkerFunc(func(context.Context) error { return tc.err }))
			if err := s.Ready(context.Background()); !errors.Is(err, tc.err) {
				t.Fatalf("got %v, want %v", err, tc.err)
			}
		})
	}
}
