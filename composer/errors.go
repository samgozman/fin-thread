package composer

import (
	"errors"
	"fmt"
)

var (
	errEmptyRegexMatch = errors.New("empty regex match")
)

// ComposeError is an error that occurs during news composing process.
type ComposeError struct {
	FnName string // Name of the function that caused the error
	Err    error  // Original error
	Source string // Source of the error
	Value  string // Value that caused the error
}

func (e *ComposeError) Error() string {
	if e.Value != "" {
		return fmt.Sprintf("[%s] error in %s: %s (value: %s)", e.FnName, e.Source, e.Err, e.Value)
	}

	return fmt.Sprintf("[%s] error in %s: %s", e.FnName, e.Source, e.Err)
}

// WithValue sets the value that caused the error.
func (e *ComposeError) WithValue(value string) *ComposeError {
	e.Value = value
	return e
}

// newErr creates a new ComposeError instance with the given error and source.
func newErr(err error, name, source string) *ComposeError {
	return &ComposeError{
		FnName: name,
		Err:    err,
		Source: source,
	}
}
