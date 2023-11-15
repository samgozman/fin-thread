package journalist

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"time"

	"github.com/mmcdole/gofeed"
)

type News struct {
	ID          string    // ID is the md5 hash of link + date
	Title       string    // Title is the title of the news
	Description string    // Description is the description of the news
	Link        string    // Link is the link to the news
	Date        time.Time // Date is the date of the news
}

func NewNews(title, description, link, date string) (*News, error) {
	dateTime, err := parseDate(date)
	if err != nil {
		return nil, err
	}

	hash := md5.Sum([]byte(link + dateTime.String()))

	return &News{
		ID:          hex.EncodeToString(hash[:]),
		Title:       title,
		Description: description,
		Link:        link,
		Date:        dateTime,
	}, nil
}

// NewsProvider is the interface for the data fetcher (via RSS, API, etc.)
type NewsProvider interface {
	Fetch(ctx context.Context) ([]News, error)
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

// Fetch fetches the news from the RSS feed
func (r *RssProvider) Fetch(ctx context.Context) ([]*News, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURLWithContext(r.Url, ctx)
	if err != nil {
		return nil, err
	}

	var news []*News
	for _, item := range feed.Items {
		newsItem, err := NewNews(item.Title, item.Description, item.Link, item.Published)
		if err != nil {
			return nil, err
		}
		news = append(news, newsItem)
	}

	return news, nil
}
