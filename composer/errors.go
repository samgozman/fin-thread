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
	Err    error  // Original error
	Source string // Source of the error
	Value  string // Value that caused the error
}

func (e *ErrCompose) Error() string {
	if e.Value != "" {
		return fmt.Sprintf("[Compose] error in %s: %s (value: %s)", e.Source, e.Err, e.Value)
	}

	return fmt.Sprintf("[Compose] error in %s: %s", e.Source, e.Err)
}

// WithValue sets the value that caused the error
func (e *ErrCompose) WithValue(value string) *ErrCompose {
	e.Value = value
	return e
}

// newErrCompose creates a new ErrCompose instance with the given error and source
func newErrCompose(err error, source string) *ErrCompose {
	return &ErrCompose{Err: err, Source: source}
}
