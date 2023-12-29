package composer

import (
	"errors"
	"fmt"
)

var (
	ErrEmptyRegexMatch = errors.New("empty regex match")
)

// ErrCompose is an error that occurs during news composing process
type ErrCompose struct {
	FnName string // Name of the function that caused the error
	Err    error  // Original error
	Source string // Source of the error
	Value  string // Value that caused the error
}

func (e *ErrCompose) Error() string {
	if e.Value != "" {
		return fmt.Sprintf("[%s] error in %s: %s (value: %s)", e.FnName, e.Source, e.Err, e.Value)
	}

	return fmt.Sprintf("[%s] error in %s: %s", e.FnName, e.Source, e.Err)
}

// WithValue sets the value that caused the error
func (e *ErrCompose) WithValue(value string) *ErrCompose {
	e.Value = value
	return e
}

// newErr creates a new ErrCompose instance with the given error and source
func newErr(err error, name, source string) *ErrCompose {
	return &ErrCompose{
		FnName: name,
		Err:    err,
		Source: source,
	}
}
