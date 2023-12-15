package utils

import (
	"reflect"
	"testing"
	"time"
)

func Test_ParseDate(t *testing.T) {
	tests := []struct {
		name       string
		dateString Datable
		want       time.Time
		wantErr    bool
	}{
		{
			name:       "RFC1123",
			dateString: "Tue, 14 Nov 2023 18:04:28 GMT",
			want:       time.Date(2023, 11, 14, 18, 4, 28, 0, time.UTC),
			wantErr:    false,
		},
		{
			name:       "RFC3339",
			dateString: "2023-11-13T12:58:48Z",
			want:       time.Date(2023, 11, 13, 12, 58, 48, 0, time.UTC),
			wantErr:    false,
		},
		{
			name:       "RFC3339 without Z",
			dateString: "2023-11-13T12:58:48",
			want:       time.Date(2023, 11, 13, 12, 58, 48, 0, time.UTC),
			wantErr:    false,
		},
		{
			name:       "RFC1123Z",
			dateString: "Mon, 13 Nov 2023 23:00:00 -0000",
			want:       time.Date(2023, 11, 13, 23, 00, 00, 0, time.UTC),
			wantErr:    false,
		},
		{
			name:       "Unix milliseconds as int",
			dateString: 1702450800000,
			want:       time.Date(2023, 12, 13, 07, 00, 00, 0, time.UTC),
			wantErr:    false,
		},
		{
			name:       "Unix milliseconds as int64",
			dateString: int64(1702450800000),
			want:       time.Date(2023, 12, 13, 07, 00, 00, 0, time.UTC),
			wantErr:    false,
		},
		{
			name:       "Unix seconds",
			dateString: 1702450800,
			want:       time.Date(2023, 12, 13, 07, 00, 00, 0, time.UTC),
			wantErr:    false,
		},
		{
			name:       "nil",
			dateString: nil,
			want:       time.Time{},
			wantErr:    false,
		},
		{
			name:       "empty string",
			dateString: "",
			want:       time.Time{},
			wantErr:    false,
		},
		{
			name:       "errorTest",
			dateString: "1234567890",
			want:       time.Time{},
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDate(tt.dateString)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseDate() = %v, want %v", got, tt.want)
			}
		})
	}
}
