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
			name: "valid rss feed with 2 providers",
			fields: fields{
				providers: []NewsProvider{
					NewRssProvider("nasdaq:stocks", "https://www.nasdaq.com/feed/rssoutbound?category=Stocks"),
					NewRssProvider("nasdaq:markets", "https://www.nasdaq.com/feed/rssoutbound?category=Markets"),
				},
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := &Journalist{
				providers: tt.fields.providers,
			}
			ctx, cancel := context.WithTimeout(tt.args.ctx, 10*time.Second)
			defer cancel()
			got, err := j.GetLatestNews(ctx)
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
