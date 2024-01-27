package publisher

import (
	"errors"
	"github.com/samgozman/fin-thread/pkg/errlvl"
)

// Error is a custom error type that contains the severity level of the error.
type Error struct {
	// severity level of the error
	level errlvl.Lvl
	// errors stack (preferably generic error + the real error)
	errs []error
}

// Error returns the string representation of the error.
func (e *Error) Error() string {
	if len(e.errs) == 1 {
		return errlvl.Wrap(e.errs[0], e.level).Error()
	}

	return errlvl.Wrap(errors.Join(e.errs...), e.level).Error()
}

// newError creates a new Error instance with the given errors.
func newError(lvl errlvl.Lvl, errs ...error) *Error {
	return &Error{
		level: lvl,
		errs:  errs,
	}
}
