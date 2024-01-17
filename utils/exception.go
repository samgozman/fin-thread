package utils

import "github.com/getsentry/sentry-go"

// CaptureSentryException is a helper function that captures an exception with the given name and error.
// The main purpose of this function is to rewrite the exception type to the given name.
// In Sentry, the exception type is always the name of the error type, which is errors.*something* and is not very useful.
func CaptureSentryException(name string, hub *sentry.Hub, err error) {
	hub.WithScope(func(scope *sentry.Scope) {
		scope.AddEventProcessor(func(e *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			e.Exception[len(e.Exception)-1].Type = name
			return e
		})
		hub.CaptureException(err)
	})
}
