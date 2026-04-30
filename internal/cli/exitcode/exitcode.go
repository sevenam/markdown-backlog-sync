// Package exitcode defines the documented exit-code contract for the CLI
// and a typed Error wrapper that carries an exit code through error chains.
package exitcode

import (
	"errors"
	"fmt"
)

// Code is a process exit code.
type Code int

// Documented exit codes. Keep in sync with docs/exit-codes.md.
const (
	OK                 Code = 0
	Generic            Code = 1
	Usage              Code = 2
	Auth               Code = 3
	Conflict           Code = 4
	Network            Code = 5
	WorkspaceIntegrity Code = 6
)

// Error wraps an underlying error with a typed exit code.
type Error struct {
	Code Code
	Err  error
}

// Errorf is a convenience constructor.
func Errorf(code Code, format string, a ...any) *Error {
	return &Error{Code: code, Err: fmt.Errorf(format, a...)}
}

// Wrap attaches an exit code to an existing error.
func Wrap(code Code, err error) *Error {
	if err == nil {
		return nil
	}
	return &Error{Code: code, Err: err}
}

func (e *Error) Error() string { return e.Err.Error() }
func (e *Error) Unwrap() error { return e.Err }

// CodeOf returns the exit code attached to err, or Generic if none.
func CodeOf(err error) Code {
	if err == nil {
		return OK
	}
	var ce *Error
	if errors.As(err, &ce) {
		return ce.Code
	}
	return Generic
}
