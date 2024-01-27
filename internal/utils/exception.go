package utils

import (
	"errors"
	"github.com/getsentry/sentry-go"
	"github.com/samgozman/fin-thread/pkg/errlvl"
)

type sentryHub interface {
	CaptureException(exception error) *sentry.EventID
	WithScope(callback func(scope *sentry.Scope))
}

// CaptureSentryException is a helper function that captures an exception with the given name and error.
// The main purpose of this function is to rewrite the exception type to the given name.
// In Sentry, the exception type is always the name of the error type, which is errors.*something* and is not very useful.
func CaptureSentryException(name string, hub sentryHub, err error) {
	errType := errorsLevelMatcher(err)
	hub.WithScope(func(scope *sentry.Scope) {
		scope.AddEventProcessor(func(e *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			// NOTE: we need to change top element type in the stack.
			// e.Exception[0] is the first element in the stack, so it's the bottom one.
			e.Exception[len(e.Exception)-1].Type = name
			e.Level = errType
			return e
		})
		hub.CaptureException(err)
	})
}

// errorsLevelMatcher is a helper function that returns the Sentry level for the given error.
func errorsLevelMatcher(err error) sentry.Level {
	switch {
	case errors.Is(err, errlvl.ErrError):
		return sentry.LevelError
	case errors.Is(err, errlvl.ErrFatal):
		return sentry.LevelFatal
	case errors.Is(err, errlvl.ErrWarn):
		return sentry.LevelWarning
	case errors.Is(err, errlvl.ErrInfo):
		return sentry.LevelInfo
	case errors.Is(err, errlvl.ErrDebug):
		return sentry.LevelDebug
	case err == nil:
		return sentry.LevelDebug
	default:
		return sentry.LevelError
	}
}
