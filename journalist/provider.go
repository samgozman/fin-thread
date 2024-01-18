package journalist

import (
	"context"
	"time"

	"github.com/mmcdole/gofeed"
)

// NewsProvider is the interface for the data fetcher (via RSS, API, etc.).
type NewsProvider interface {
	Fetch(ctx context.Context, until time.Time) (NewsList, error)
}

// RssProvider is the RSS provider implementation.
type RssProvider struct {
	Name string // Name is used for logging purposes
	URL  string
}

// NewRssProvider creates a new RssProvider instance.
func NewRssProvider(name, url string) *RssProvider {
	return &RssProvider{
		Name: name,
		URL:  url,
	}
}

// Fetch fetches the news from the RSS feed until the given date.
func (r *RssProvider) Fetch(ctx context.Context, until time.Time) (NewsList, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURLWithContext(r.URL, ctx)
	if err != nil {
		return nil, newErrProvider(r.Name, err.Error())
	}

	var news NewsList
	for _, item := range feed.Items {
		newsItem, err := newNews(item.Title, item.Description, item.Link, item.Published, r.Name)
		if err != nil {
			return nil, newErrProvider(r.Name, err.Error())
		}
		news = append(news, newsItem)
	}

	for i, n := range news {
		// Remove duplicated news by date
		if n.Date.Before(until) {
			news = news[:i]
			break
		}
	}

	return news, nil
}
