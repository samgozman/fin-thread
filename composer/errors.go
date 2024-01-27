package composer

import (
	"errors"
	"fmt"
	"github.com/samgozman/fin-thread/pkg/errlvl"
)

var (
	errEmptyRegexMatch = errors.New("empty regex match")
)

// Error is an error that occurs during news composing process.
type Error struct {
	level  errlvl.Lvl // severity level of the error
	err    error      // errors stack (preferably generic error + the real error)
	fnName string     // Name of the function that caused the error
	source string     // source of the error
	value  string     // value that caused the error
}

func (e *Error) Error() string {
	if e.value != "" {
		return errlvl.Wrap(fmt.Errorf("[%s] error in %s: %w (value: %s)", e.fnName, e.source, e.err, e.value), e.level).Error()
	}

	return errlvl.Wrap(fmt.Errorf("[%s] error in %s: %w", e.fnName, e.source, e.err), e.level).Error()
}

// WithValue sets the value that caused the error.
func (e *Error) WithValue(value string) *Error {
	e.value = value
	return e
}

// newError creates a new Error instance with the given error and source.
func newError(err error, level errlvl.Lvl, name, source string) *Error {
	return &Error{
		level:  level,
		fnName: name,
		err:    err,
		source: source,
	}
}
