package journalist

import (
	"reflect"
	"testing"
	"time"
)

func TestNewNews(t *testing.T) {
	type args struct {
		title        string
		description  string
		link         string
		date         string
		providerName string
	}
	tests := []struct {
		name    string
		args    args
		want    *News
		wantErr bool
	}{
		{
			name: "valid news",
			args: args{
				title:        "title",
				description:  "description",
				link:         "link",
				date:         "Mon, 02 Jan 2006 15:04:05 MST",
				providerName: "provider",
			},
			want: &News{
				ID:           "cbd261a703d9f7f5bf08f8ede0a1e99b",
				Title:        "title",
				Description:  "description",
				Link:         "link",
				Date:         time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC),
				ProviderName: "provider",
			},
			wantErr: false,
		},
		{
			name: "invalid date",
			args: args{
				title:       "title",
				description: "description",
				link:        "link",
				date:        "invalid date",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewNews(tt.args.title, tt.args.description, tt.args.link, tt.args.date, tt.args.providerName)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewNews() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewNews() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TODO: Add tests for ToJSON and ToContentJSON
