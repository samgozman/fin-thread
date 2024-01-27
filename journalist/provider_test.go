package journalist

import (
	"context"
	"testing"
	"time"
)

func TestRssProvider_Fetch(t *testing.T) {
	type fields struct {
		Name string
		URL  string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "valid rss feed",
			fields: fields{
				Name: "test",
				URL:  "https://www.nasdaq.com/feed/rssoutbound?category=Dividends",
			},
			wantErr: false,
		},
		{
			name: "error feed",
			fields: fields{
				Name: "test",
				URL:  "https://google.com/",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &RssProvider{
				Name: tt.fields.Name,
				URL:  tt.fields.URL,
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			got, err := r.Fetch(ctx, time.Now().AddDate(0, 0, -3))
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
				if news.ProviderName != tt.fields.Name {
					t.Errorf("RssProvider.Fetch() news.providerName = %v, want %v", news.ProviderName, tt.fields.Name)
					return
				}
			}
		})
	}
}
