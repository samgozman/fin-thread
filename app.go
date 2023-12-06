package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/samgozman/fin-thread/archivist/models"
	"log/slog"
	"slices"
	"time"

	. "github.com/samgozman/fin-thread/archivist"
	. "github.com/samgozman/fin-thread/composer"
	. "github.com/samgozman/fin-thread/journalist"
	. "github.com/samgozman/fin-thread/publisher"
)

type App struct {
	staff     *Staff
	composer  *Composer
	publisher *TelegramPublisher
	archivist *Archivist
	logger    *slog.Logger
}

func (a *App) CreateMarketNewsJob(until time.Time) JobFunc {
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		l := a.logger.WithGroup("[CreateMarketNewsJob]")

		news, e := a.staff.marketJournalist.GetLatestNews(ctx, until)
		if e != nil {
			l.Info("[GetLatestNews]", "error", e)
		}
		if len(news) == 0 {
			return
		}

		uniqueNews, err := a.RemoveDuplicates(ctx, news)
		if err != nil {
			l.Info("[RemoveDuplicates]", "error", err)
			return
		}
		if len(uniqueNews) == 0 {
			return
		}

		err = a.ComposeAndPostNews(ctx, uniqueNews)
		if err != nil {
			l.Warn("[ComposeAndPostNews]", "error", err)
			return
		}
	}
}

func (a *App) CreateTradingEconomicsNewsJob(until time.Time) JobFunc {
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		l := a.logger.WithGroup("[CreateTradingEconomicsNewsJob]")

		news, e := a.staff.teJournalist.GetLatestNews(ctx, until)
		if e != nil {
			l.Info("[GetLatestNews]", "error", e)
		}
		if len(news) == 0 {
			return
		}

		// RemoveDuplicates news by keywords, if Title do not contain any of the keywords - skip it
		filteredNews := news.FilterByKeywords(
			[]string{"European Union", "United States", "United Kingdom", "China", "Germany", "France", "Japan", "Italy", "India"},
		)

		if len(news) == 0 {
			return
		}

		uniqueNews, err := a.RemoveDuplicates(ctx, filteredNews)
		if err != nil {
			l.Info("[RemoveDuplicates]", "error", err)
			return
		}
		if len(uniqueNews) == 0 {
			return
		}

		err = a.ComposeAndPostNews(ctx, uniqueNews)
		if err != nil {
			l.Warn("[ComposeAndPostNews]", "error", err)
			return
		}
	}
}

func (a *App) ComposeAndPostNews(ctx context.Context, news NewsList) error {
	composedNews, err := a.composer.Compose(ctx, news)
	if err != nil {
		return errors.New(fmt.Sprintf("[composer.Compose]: %v", err))
	}
	if len(composedNews) == 0 {
		return nil
	}

	dbNews := make([]models.News, len(composedNews))

	for i, n := range composedNews {
		// composedNews and news are not the same length because of filtering
		// so, we need to use the original news by hash
		originalNews := news.FindById(n.ID)
		if originalNews == nil {
			return errors.New(fmt.Sprintf("cannot find original news %v", n))
		}

		f := formatNews(n, originalNews.ProviderName)
		id, err := a.publisher.Publish(f)
		if err != nil {
			return errors.New(fmt.Sprintf("[publisher.Publish]: %v", err))
		}

		meta, err := json.Marshal(struct {
			Tickers  []string
			Markets  []string
			Hashtags []string
		}{
			Tickers:  n.Tickers,
			Markets:  n.Markets,
			Hashtags: n.Hashtags,
		})
		if err != nil {
			return errors.New(fmt.Sprintf("[json.Marshal] meta: %v", err))
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
			return errors.New(fmt.Sprintf("[News.Create]: %v", err))
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
		return nil, errors.New(fmt.Sprintf("[News.FindAllByHashes]: %v", err))
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

// formatNews formats the news to be posted to the channel
func formatNews(n *ComposedNews, provider string) string {
	return fmt.Sprintf(
		"ID: %s\nProvider: %s\nHashtags: %s\nTickers: %s\nMarkets: %s\n%s",
		n.ID, provider, n.Hashtags, n.Tickers, n.Markets, n.Text,
	)
}

// Staff is the structure that holds all the journalists
type Staff struct {
	// Journalist for common market news
	marketJournalist *Journalist
	// Specialized journalist for Trading Economics news RSS feed only
	teJournalist *Journalist
}

type JobFunc func()
