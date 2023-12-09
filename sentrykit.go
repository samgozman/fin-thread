package main

import (
	"context"
	"fmt"
	"github.com/getsentry/sentry-go"
	"log/slog"
)

// SentryKit is a wrapper around sentry-go SDK that provides some convenience methods for logging and tracing
type SentryKit struct {
	log *slog.Logger
}

// GetHub returns a sentry hub from the context, or creates a new one if it's not present
func (s *SentryKit) GetHub(ctx context.Context) *sentry.Hub {
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub().Clone()
		ctx = sentry.SetHubOnContext(ctx, hub)
	}
	return hub
}

// StartJobTransaction starts a new transaction for a job with the given name and returns a span
func (s *SentryKit) StartJobTransaction(ctx context.Context, n string) *sentry.Span {
	return sentry.StartTransaction(ctx, fmt.Sprintf("Job.%s", n))
}

// AddBreadcrumb adds a breadcrumb to the current hub with the given category and message
func (s *SentryKit) AddBreadcrumb(hub *sentry.Hub, c, m string) {
	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Category: c,
		Message:  m,
		Level:    sentry.LevelInfo,
	}, nil)
}

// StartSpan starts a new span with the given operation and transaction name
func (s *SentryKit) StartSpan(ctx context.Context, o, t string) *sentry.Span {
	return sentry.StartSpan(ctx, o, sentry.WithTransactionName(t))
}

// CaptureError logs the error and captures it in the given hub
func (s *SentryKit) CaptureError(hub *sentry.Hub, m string, err error) {
	s.log.Error(m, err)
	hub.CaptureException(err)
}
