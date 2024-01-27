package errlvl

import (
	"errors"
	"fmt"
	"testing"
)

func TestWrap(t *testing.T) {
	type args struct {
		err   error
		level Lvl
	}
	tests := []struct {
		name      string
		args      args
		wantLevel ErrorLevel
	}{
		{
			name: "wrap error with level",
			args: args{
				err:   errors.New("test"),
				level: INFO,
			},
			wantLevel: ErrInfo,
		},
		{
			name: "wrap joined errors",
			args: args{
				err:   errors.Join(errors.New("test1"), errors.New("test2")),
				level: WARN,
			},
			wantLevel: ErrWarn,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Wrap(tt.args.err, tt.args.level)
			if !errors.Is(err, tt.wantLevel) {
				t.Errorf("Wrap() wrong error level = %v, want %v", err, tt.wantLevel)
			}
			if !errors.Is(err, tt.args.err) {
				t.Errorf("Wrap() original error not wrapped = %v, want %v", err, tt.args.err)
			}
		})
	}
}

func Test_hasLevel(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "error with info level",
			args: args{
				err: fmt.Errorf("%w %w", ErrInfo, errors.New("test")),
			},
			want: true,
		},
		{
			name: "error with warn level",
			args: args{
				err: fmt.Errorf("%w %w", ErrWarn, errors.New("test")),
			},
			want: true,
		},
		{
			name: "error with error level",
			args: args{
				err: fmt.Errorf("%w %w", ErrError, errors.New("test")),
			},
			want: true,
		},
		{
			name: "error with debug level",
			args: args{
				err: fmt.Errorf("%w %w", ErrDebug, errors.New("test")),
			},
			want: true,
		},
		{
			name: "error with fatal level",
			args: args{
				err: fmt.Errorf("%w %w", ErrFatal, errors.New("test")),
			},
			want: true,
		},
		{
			name: "error without level",
			args: args{
				err: errors.New("test"),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasLevel(tt.args.err); got != tt.want {
				t.Errorf("hasLevel() = %v, want %v", got, tt.want)
			}
		})
	}
}
