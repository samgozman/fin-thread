package utils

import (
	"errors"
	"github.com/getsentry/sentry-go"
	"github.com/samgozman/fin-thread/pkg/errlvl"
	"github.com/stretchr/testify/mock"
	"testing"
)

type MockHub struct {
	mock.Mock
}

func (m *MockHub) CaptureException(exception error) *sentry.EventID {
	args := m.Called(exception)
	return args.Get(0).(*sentry.EventID)
}

func (m *MockHub) WithScope(callback func(scope *sentry.Scope)) {
	m.Called(callback)
	callback(sentry.NewScope())
}

func TestCaptureSentryException(t *testing.T) {
	type args struct {
		name string
		hub  *MockHub
		err  error
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Test with error",
			args: args{
				name: "someError",
				hub:  new(MockHub),
				err:  errors.New("some error"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.args.hub.On("WithScope", mock.Anything)
			tt.args.hub.On("CaptureException", tt.args.err).Return(new(sentry.EventID))

			CaptureSentryException(tt.args.name, tt.args.hub, tt.args.err)

			tt.args.hub.AssertExpectations(t)
		})
	}
}

func Test_errorsLevelMatcher(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want sentry.Level
	}{
		{
			name: "Test with nil error",
			args: args{
				err: nil,
			},
			want: sentry.LevelDebug,
		},
		{
			name: "Test with generic error",
			args: args{
				err: errors.New("generic error"),
			},
			want: sentry.LevelError,
		},
		{
			name: "Test with ErrError",
			args: args{
				err: errlvl.ErrError,
			},
			want: sentry.LevelError,
		},
		{
			name: "Test with ErrFatal",
			args: args{
				err: errlvl.ErrFatal,
			},
			want: sentry.LevelFatal,
		},
		{
			name: "Test with ErrWarn",
			args: args{
				err: errlvl.ErrWarn,
			},
			want: sentry.LevelWarning,
		},
		{
			name: "Test with ErrInfo",
			args: args{
				err: errlvl.ErrInfo,
			},
			want: sentry.LevelInfo,
		},
		{
			name: "Test with ErrDebug",
			args: args{
				err: errlvl.ErrDebug,
			},
			want: sentry.LevelDebug,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errorsLevelMatcher(tt.args.err); got != tt.want {
				t.Errorf("errorsLevelMatcher() = %v, want %v", got, tt.want)
			}
		})
	}
}
