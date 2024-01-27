package errlvl

import (
	"errors"
	"fmt"
)

type Lvl uint8

const (
	DEBUG Lvl = iota + 1
	INFO
	WARN
	ERROR
	FATAL
)

// ErrorLevel is a type that represents the severity of an error in the application.
//
// This is the global error levels that should be used throughout the application to determine the severity of the error.
type ErrorLevel error

var (
	ErrDebug ErrorLevel = errors.New("[DEBUG]") // ErrDebug is returned when the global level is set to DEBUG.
	ErrInfo  ErrorLevel = errors.New("[INFO]")  // ErrInfo is returned when the global level is set to INFO.
	ErrWarn  ErrorLevel = errors.New("[WARN]")  // ErrWarn is returned when the global level is set to WARN.
	ErrError ErrorLevel = errors.New("[ERROR]") // ErrError is returned when the global level is set to ERROR.
	ErrFatal ErrorLevel = errors.New("[FATAL]") // ErrFatal is returned when the global level is set to FATAL.
)

// Wrap wraps the given error with the given level.
func Wrap(err error, level Lvl) error {
	if hasLevel(err) {
		return err
	}

	switch level {
	case DEBUG:
		return fmt.Errorf("%w %w", ErrDebug, err)
	case INFO:
		return fmt.Errorf("%w %w", ErrInfo, err)
	case WARN:
		return fmt.Errorf("%w %w", ErrWarn, err)
	case ERROR:
		return fmt.Errorf("%w %w", ErrError, err)
	case FATAL:
		return fmt.Errorf("%w %w", ErrFatal, err)
	default:
		return fmt.Errorf("%w %w", ErrError, err)
	}
}

// hasLevel checks if the given error has a level ser already.
func hasLevel(err error) bool {
	return errors.Is(err, ErrDebug) || errors.Is(err, ErrInfo) || errors.Is(err, ErrWarn) || errors.Is(err, ErrError) || errors.Is(err, ErrFatal)
}
