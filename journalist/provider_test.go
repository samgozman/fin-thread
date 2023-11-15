package journalist

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestNewNews(t *testing.T) {
	type args struct {
		title       string
		description string
		link        string
		date        string
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
				title:       "title",
				description: "description",
				link:        "link",
				date:        "Mon, 02 Jan 2006 15:04:05 MST",
			},
			want: &News{
				ID:          "fe7fba6bc0c72172a7b007cf8ee4adac",
				Title:       "title",
				Description: "description",
				Link:        "link",
				Date:        time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC),
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
			got, err := NewNews(tt.args.title, tt.args.description, tt.args.link, tt.args.date)
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

func TestRssProvider_Fetch(t *testing.T) {
	type fields struct {
		Name string
		Url  string
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "valid rss feed",
			fields: fields{
				Name: "test",
				Url:  "https://www.nasdaq.com/feed/rssoutbound?category=Dividends",
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: false,
		},
		{
			name: "error feed",
			fields: fields{
				Name: "test",
				Url:  "https://google.com/",
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RssProvider{
				Name: tt.fields.Name,
				Url:  tt.fields.Url,
			}
			ctx, cancel := context.WithTimeout(tt.args.ctx, 10*time.Second)
			defer cancel()
			got, err := r.Fetch(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("RssProvider.Fetch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			for _, news := range got {
				if news.ID == "" {
					t.Error("RssProvider.Fetch() news.ID is empty")
					return
				}
				if news.Title == "" {
					t.Error("RssProvider.Fetch() news.Title is empty")
					return
				}
				if news.Description == "" {
					t.Error("RssProvider.Fetch() news.Description is empty")
					return
				}
				if news.Link == "" {
					t.Error("RssProvider.Fetch() news.Link is empty")
					return
				}
				if news.Date.IsZero() {
					t.Error("RssProvider.Fetch() news.Date is empty")
					return
				}
			}
		})
	}
}
