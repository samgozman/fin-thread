package journalist

import (
	"context"
	"testing"
	"time"
)

func TestJournalist_GetLatestNews(t *testing.T) {
	type fields struct {
		providers []NewsProvider
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "valid rss feed with 2 providers",
			fields: fields{
				providers: []NewsProvider{
					NewRssProvider("nasdaq:stocks", "https://www.nasdaq.com/feed/rssoutbound?category=Stocks"),
					NewRssProvider("nasdaq:markets", "https://www.nasdaq.com/feed/rssoutbound?category=Markets"),
				},
			},
			wantErr: false,
		},
		{
			name: "valid rss feed with 3 providers if one of them is invalid",
			fields: fields{
				providers: []NewsProvider{
					NewRssProvider("nasdaq:stocks", "https://www.nasdaq.com/feed/rssoutbound?category=Stocks"),
					NewRssProvider("nasdaq:markets", "https://www.nasdaq.com/feed/rssoutbound?category=Markets"),
					NewRssProvider("nasdaq:invalid", "https://httpbin.org/delay/15"),
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := &Journalist{
				providers: tt.fields.providers,
			}
			ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
			defer cancel()
			got, err := j.GetLatestNews(ctx, time.Now().AddDate(0, 0, -3))
			if (err != nil) != tt.wantErr {
				t.Errorf("Journalist.GetLatestNews() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			var nasdaqStocksCount, nasdaqMarketsCount int
			for _, news := range got {
				if news.ID == "" {
					t.Error("Journalist.GetLatestNews() news.ID is empty")
					return
				}
				if news.Title == "" {
					t.Error("Journalist.GetLatestNews() news.Title is empty")
					return
				}
				if news.Description == "" {
					t.Error("Journalist.GetLatestNews() news.Description is empty")
					return
				}
				if news.Link == "" {
					t.Error("Journalist.GetLatestNews() news.Link is empty")
					return
				}
				if news.Date.IsZero() {
					t.Error("Journalist.GetLatestNews() news.Date is empty")
					return
				}
				if news.ProviderName == "nasdaq:stocks" {
					nasdaqStocksCount++
				}
				if news.ProviderName == "nasdaq:markets" {
					nasdaqMarketsCount++
				}
			}

			if nasdaqStocksCount == 0 {
				t.Error("Journalist.GetLatestNews() nasdaqStocksCount is 0")
			}
			if nasdaqMarketsCount == 0 {
				t.Error("Journalist.GetLatestNews() nasdaqMarketsCount is 0")
			}
		})
	}
}
