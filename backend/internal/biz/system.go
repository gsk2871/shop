// Package biz defines application use cases and repository contracts.
package biz

import "context"

// Checker tests database availability.
type Checker interface{ Check(context.Context) error }

// System provides system readiness checks.
type System struct{ checker Checker }

// NewSystem constructs the readiness use case.
func NewSystem(checker Checker) *System { return &System{checker: checker} }

// Ready checks whether the database is reachable.
func (s *System) Ready(ctx context.Context) error { return s.checker.Check(ctx) }
