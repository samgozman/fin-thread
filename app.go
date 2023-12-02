package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/samgozman/go-fin-feed/archivist/models"
	"slices"
	"strings"
	"time"

	. "github.com/samgozman/go-fin-feed/archivist"
	. "github.com/samgozman/go-fin-feed/composer"
	. "github.com/samgozman/go-fin-feed/journalist"
	. "github.com/samgozman/go-fin-feed/publisher"
)

type App struct {
	staff     *Staff
	composer  *Composer
	publisher *TelegramPublisher
	archivist *Archivist
}

func (a *App) CreateMarketNewsJob(until time.Time) JobFunc {
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		news, e := a.staff.marketJournalist.GetLatestNews(ctx, until)
		if e != nil {
			fmt.Println(e)
			return
		}

		if len(news) == 0 {
			return
		}

		err := a.ComposeAndPostNews(ctx, news)
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}

func (a *App) CreateTradingEconomicsNewsJob(until time.Time) JobFunc {
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		news, e := a.staff.teJournalist.GetLatestNews(ctx, until)
		if e != nil {
			fmt.Println(e)
			return
		}

		// Filter news by keywords, if Title do not contain any of the keywords - skip it
		filterKeywords := []string{"European Union", "United States", "United Kingdom", "China", "Germany", "France", "Japan", "Italy", "India"}
		var filteredNews NewsList
		for _, n := range news {
			c := false
			// Check if any keyword is present in the title
			for _, k := range filterKeywords {
				if strings.Contains(n.Title, k) {
					c = true
					break
				}
			}
			if c {
				filteredNews = append(filteredNews, n)
			}
		}

		if len(news) == 0 {
			return
		}

		err := a.ComposeAndPostNews(ctx, news)
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}

func (a *App) ComposeAndPostNews(ctx context.Context, news NewsList) error {
	composedNews, err := a.PrepareNews(ctx, news)
	if err != nil {
		return err
	}
	if len(composedNews) == 0 {
		return nil
	}

	dbNews := make([]models.News, len(composedNews))

	for i, n := range composedNews {
		f := a.FormatNews(n)
		id, err := a.publisher.Publish(f)
		if err != nil {
			return err
		}

		meta, err := json.Marshal(n.MetaData)
		if err != nil {
			return err
		}

		// composedNews and news are not the same length because of filtering
		// so, we need to use the original news by hash
		originalNews := news.FindById(n.NewsID)
		if originalNews == nil {
			return errors.New(fmt.Sprintf("cannot find original news %v", n))
		}

		dbNews[i] = models.News{
			ChannelID:     a.publisher.ChannelID,
			PublicationID: id,
			OriginalTitle: originalNews.Title,
			OriginalDesc:  originalNews.Description,
			OriginalDate:  originalNews.Date,
			URL:           originalNews.Link,
			PublishedAt:   time.Now(),
			ComposedText:  n.Text,
			MetaData:      meta,
		}
	}

	// TODO: add create many method to archivist with transaction
	for _, n := range dbNews {
		err := a.archivist.Entities.News.Create(ctx, &n)
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *App) RemoveDuplicates(ctx context.Context, news NewsList) (NewsList, error) {
	hashes := make([]string, len(news))
	for i, n := range news {
		hashes[i] = n.ID
	}

	exists, err := a.archivist.Entities.News.FindAllByHashes(ctx, hashes)
	if err != nil {
		return nil, err
	}
	existedHashes := make([]string, len(exists))
	for i, n := range exists {
		existedHashes[i] = n.Hash
	}

	var uniqueNews NewsList
	for _, n := range news {
		if !slices.Contains(existedHashes, n.ID) {
			uniqueNews = append(uniqueNews, n)
		}
	}
	return uniqueNews, nil
}

func (a *App) PrepareNews(ctx context.Context, news NewsList) ([]*ComposedNews, error) {
	importantNews, err := a.composer.ChooseMostImportantNews(ctx, news)
	if err != nil {
		return nil, err
	}

	if len(importantNews) == 0 {
		return nil, nil
	}

	composedNews, err := a.composer.ComposeNews(ctx, importantNews)
	if err != nil {
		return nil, err
	}

	return composedNews, nil
}

func (a *App) FormatNews(n *ComposedNews) string {
	return fmt.Sprintf("ID: %s\nHashtags: %s\nTickers: %s\nMarkets: %s\n%s", n.NewsID, n.MetaData.Hashtags, n.MetaData.Tickers, n.MetaData.Markets, n.Text)
}

// Staff is the structure that holds all the journalists
type Staff struct {
	// Journalist for common market news
	marketJournalist *Journalist
	// Specialized journalist for Trading Economics news RSS feed only
	teJournalist *Journalist
}

type JobFunc func()
