package journalist

import (
	"context"
	"testing"
	"time"
)

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
			got, err := r.Fetch(ctx, time.Now().AddDate(0, 0, -3))
			if (err != nil) != tt.wantErr {
				t.Errorf("RssProvider.Fetch() error = %v, wantErr %w", err, tt.wantErr)
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
				if news.ProviderName != tt.fields.Name {
					t.Errorf("RssProvider.Fetch() news.ProviderName = %v, want %v", news.ProviderName, tt.fields.Name)
					return
				}
			}
		})
	}
}
