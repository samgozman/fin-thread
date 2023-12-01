package journalist

import (
	"context"
	"fmt"
	"time"

	"github.com/mmcdole/gofeed"
)

// NewsProvider is the interface for the data fetcher (via RSS, API, etc.)
type NewsProvider interface {
	Fetch(ctx context.Context, until time.Time) (NewsList, error)
}

// RssProvider is the RSS provider implementation
type RssProvider struct {
	Name string // Name is used for logging purposes
	Url  string
}

// NewRssProvider creates a new RssProvider instance
func NewRssProvider(name, url string) *RssProvider {
	return &RssProvider{
		Name: name,
		Url:  url,
	}
}

// Fetch fetches the news from the RSS feed until the given date
func (r *RssProvider) Fetch(ctx context.Context, until time.Time) (NewsList, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURLWithContext(r.Url, ctx)
	if err != nil {
		return nil, NewProviderErr(r.Name, err.Error())
	}

	var news NewsList
	for _, item := range feed.Items {
		newsItem, err := NewNews(item.Title, item.Description, item.Link, item.Published, r.Name)
		if err != nil {
			return nil, NewProviderErr(r.Name, err.Error())
		}
		news = append(news, newsItem)
	}

	// Filter news by date
	for i, n := range news {
		if n.Date.Before(until) {
			news = news[:i]
			break
		}
	}

	return news, nil
}

// ProviderErr is the error type for the NewsProvider
type ProviderErr struct {
	Err          string
	ProviderName string
}

func (e *ProviderErr) Error() string {
	return fmt.Sprintf("Provider %s error: %s", e.ProviderName, e.Err)
}

func NewProviderErr(providerName, err string) *ProviderErr {
	return &ProviderErr{
		Err:          err,
		ProviderName: providerName,
	}
}
