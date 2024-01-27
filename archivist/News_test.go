package archivist

import (
	"crypto/md5"
	"encoding/hex"
	"github.com/google/uuid"
	"github.com/samgozman/fin-thread/composer"
	"gorm.io/gorm"
	"reflect"
	"testing"
	"time"
)

func TestNews_BeforeCreate(t *testing.T) {
	type args struct {
		in0 *gorm.DB
	}
	tests := []struct {
		name    string
		fields  News
		args    args
		wantErr bool
	}{
		{
			name: "Test News BeforeCreate",
			fields: News{
				ChannelID:     "testChannel",
				ProviderName:  "testProvider",
				URL:           "https://test.com",
				OriginalTitle: "Test Title",
				OriginalDesc:  "Test Description",
				OriginalDate:  time.Now(),
			},
			args: args{
				in0: &gorm.DB{},
			},
			wantErr: false,
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

func TestNews_GenerateHash(t *testing.T) {
	tests := []struct {
		name   string
		fields News
	}{
		{
			name: "Test News GenerateHash",
			fields: News{
				ChannelID:     "testChannel",
				ProviderName:  "testProvider",
				URL:           "https://test.com",
				OriginalTitle: "Test Title",
				OriginalDesc:  "Test Description",
				OriginalDate:  time.Now(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields.GenerateHash()

			hash := md5.Sum([]byte(tt.fields.OriginalTitle + tt.fields.OriginalDesc))

			if tt.fields.Hash != hex.EncodeToString(hash[:]) {
				t.Errorf("GenerateHash() error = %v, wantErr %v", tt.fields.Hash, hex.EncodeToString(hash[:]))
			}
		})
	}
}

func TestNews_ToHeadline(t *testing.T) {
	okID := uuid.New()

	tests := []struct {
		name   string
		fields News
		want   *composer.Headline
	}{
		{
			name: "Test News ToHeadline",
			fields: News{
				ID:            okID,
				ChannelID:     "testChannel",
				PublicationID: "3333",
				ProviderName:  "testProvider",
				URL:           "https://test.com",
				OriginalTitle: "Test Title",
				OriginalDesc:  "Test Description",
				OriginalDate:  time.Now(),
			},
			want: &composer.Headline{
				ID:   okID.String(),
				Text: "Test Title",
				Link: "https://t.me/testChannel/3333",
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

func TestNews_Validate(t *testing.T) {
	tests := []struct {
		name    string
		fields  News
		wantErr bool
	}{
		{
			name: "Test News Validate - Valid News",
			fields: News{
				ChannelID:     "testChannel",
				ProviderName:  "testProvider",
				URL:           "https://test.com",
				OriginalTitle: "Test Title",
				OriginalDesc:  "Test Description",
				OriginalDate:  time.Now(),
			},
			wantErr: false,
		},
		{
			name: "Test News Validate - Invalid News (ChannelID too long)",
			fields: News{
				ChannelID:     "testChanneltestChanneltestChanneltestChanneltestChanneltestChanneltestChanneltestChannel",
				ProviderName:  "testProvider",
				URL:           "https://test.com",
				OriginalTitle: "Test Title",
				OriginalDesc:  "Test Description",
				OriginalDate:  time.Now(),
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
