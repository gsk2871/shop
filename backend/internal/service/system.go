// Package service adapts application use cases to transport handlers.
package service

import (
	"context"
	"peptide-site/internal/biz"
)

// System provides system transport operations.
type System struct{ usecase *biz.System }

// NewSystem constructs the system service.
func NewSystem(usecase *biz.System) *System { return &System{usecase: usecase} }

// Ready delegates the readiness check to the application layer.
func (s *System) Ready(ctx context.Context) error { return s.usecase.Ready(ctx) }
