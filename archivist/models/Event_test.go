package models

import (
	"github.com/google/uuid"
	"github.com/samgozman/fin-thread/composer"
	"gorm.io/gorm"
	"reflect"
	"strings"
	"testing"
)

func TestEvent_BeforeCreate(t *testing.T) {
	type args struct {
		in0 *gorm.DB
	}
	tests := []struct {
		name    string
		fields  Event
		args    args
		wantErr bool
	}{
		{
			name: "valid event",
			fields: Event{
				ChannelID:    "testChannel",
				ProviderName: "testProvider",
				Title:        "testTitle",
			},
			args: args{
				in0: nil, // Since BeforeCreate does not use this argument, it's safe to pass nil in tests
			},
			wantErr: false,
		},
		{
			name: "invalid event with long ChannelID",
			fields: Event{
				ChannelID:    strings.Repeat("a", 65), // ChannelID length > 64
				ProviderName: "testProvider",
				Title:        "testTitle",
			},
			args: args{
				in0: nil,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fields.BeforeCreate(tt.args.in0); (err != nil) != tt.wantErr {
				t.Errorf("BeforeCreate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEvent_BeforeUpdate(t *testing.T) {
	type args struct {
		in0 *gorm.DB
	}
	tests := []struct {
		name    string
		fields  Event
		args    args
		wantErr bool
	}{
		{
			name: "valid event",
			fields: Event{
				ChannelID:    "testChannel",
				ProviderName: "testProvider",
				Title:        "testTitle",
			},
			args: args{
				in0: nil, // Since BeforeUpdate does not use this argument, it's safe to pass nil in tests
			},
			wantErr: false,
		},
		{
			name: "invalid event with long ChannelID",
			fields: Event{
				ChannelID:    strings.Repeat("a", 65), // ChannelID length > 64
				ProviderName: "testProvider",
				Title:        "testTitle",
			},
			args: args{
				in0: nil,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fields.BeforeUpdate(tt.args.in0); (err != nil) != tt.wantErr {
				t.Errorf("BeforeUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEvent_ToHeadline(t *testing.T) {
	okID := uuid.New()

	tests := []struct {
		name   string
		fields Event
		want   *composer.Headline
	}{
		{
			name: "valid event",
			fields: Event{
				ID:           okID,
				ChannelID:    "testChannel",
				ProviderName: "testProvider",
				Title:        "testTitle",
			},
			want: &composer.Headline{
				ID:   okID.String(),
				Text: "testTitle",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fields.ToHeadline(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ToHeadline() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvent_Validate(t *testing.T) {
	tests := []struct {
		name    string
		fields  Event
		wantErr bool
	}{
		{
			name: "valid event",
			fields: Event{
				ChannelID:    "testChannel",
				ProviderName: "testProvider",
				Title:        "testTitle",
			},
			wantErr: false,
		},
		{
			name: "invalid event with long ChannelID",
			fields: Event{
				ChannelID:    strings.Repeat("a", 65), // ChannelID length > 64
				ProviderName: "testProvider",
				Title:        "testTitle",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fields.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewEventsDB(t *testing.T) {
	type args struct {
		db *gorm.DB
	}
	tests := []struct {
		name string
		args args
		want *EventsDB
	}{
		{
			name: "valid DB",
			args: args{
				db: &gorm.DB{}, // Mocked gorm.DB instance
			},
			want: &EventsDB{
				Conn: &gorm.DB{}, // Expected to return EventsDB with the same gorm.DB instance
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewEventsDB(tt.args.db); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewEventsDB() = %v, want %v", got, tt.want)
			}
		})
	}
}
