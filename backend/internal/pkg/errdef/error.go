// Package errdef supplies stable errors without exposing database internals.
package errdef

type Error struct {
	Status int
	Code   string
}

func (e *Error) Error() string          { return e.Code }
func New(status int, code string) error { return &Error{status, code} }
func Invalid() error                    { return New(400, "INVALID_ARGUMENT") }
func Denied() error                     { return New(403, "PERMISSION_DENIED") }
func Conflict() error                   { return New(409, "CONFLICT") }
func Internal(err error) error          { _ = err; return New(503, "SERVICE_UNAVAILABLE") }
