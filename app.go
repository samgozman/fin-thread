package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/getsentry/sentry-go"
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
	composer  *Composer
	publisher *TelegramPublisher
	archivist *Archivist
	logger    *slog.Logger
}

func (a *App) FinJob(j *Journalist, until time.Time) JobFunc {
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		jobName := fmt.Sprintf("FinJob.%s", j.Name)

		// Sentry performance monitoring
		hub := sentry.GetHubFromContext(ctx)
		if hub == nil {
			hub = sentry.CurrentHub().Clone()
			ctx = sentry.SetHubOnContext(ctx, hub)
		}
		defer hub.Flush(2 * time.Second)
		transaction := sentry.StartTransaction(ctx, fmt.Sprintf("App.%s", jobName))
		defer transaction.Finish()

		news, err := j.GetLatestNews(ctx, until)
		if err != nil {
			a.logger.Info(fmt.Sprintf("[%s][GetLatestNews]", jobName), "error", err)
			hub.CaptureException(err)
		}
		if len(news) == 0 {
			return
		}

		uniqueNews, err := a.RemoveDuplicates(ctx, news)
		if err != nil {
			a.logger.Info(fmt.Sprintf("[%s][RemoveDuplicates]", jobName), "error", err)
			hub.CaptureException(err)
			return
		}
		if len(uniqueNews) == 0 {
			return
		}

		err = a.ComposeAndPostNews(ctx, uniqueNews)
		if err != nil {
			a.logger.Warn(fmt.Sprintf("[%s][ComposeAndPostNews]", jobName), "error", err)
			hub.CaptureException(err)
			return
		}
	}
}

func (a *App) ComposeAndPostNews(ctx context.Context, news NewsList) error {
	span := sentry.StartSpan(ctx, "Compose", sentry.WithTransactionName("App.ComposeAndPostNews"))
	composedNews, err := a.composer.Compose(ctx, news)
	span.Finish()
	if err != nil {
		return errors.New(fmt.Sprintf("[composer.Compose]: %v", err))
	}
	if len(composedNews) == 0 {
		return nil
	}

	dbNews := make([]models.News, len(composedNews))

	// Create news in the database
	for i, n := range composedNews {
		// composedNews and news are not the same length because of filtering
		// so, we need to use the original news by hash
		originalNews := news.FindById(n.ID)
		if originalNews == nil {
			return errors.New(fmt.Sprintf("cannot find original news %v", n))
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
			Hash:          n.ID,
			ChannelID:     a.publisher.ChannelID,
			OriginalTitle: originalNews.Title,
			OriginalDesc:  originalNews.Description,
			OriginalDate:  originalNews.Date,
			URL:           originalNews.Link,
			IsSuspicious:  originalNews.IsSuspicious,
			ComposedText:  n.Text,
			MetaData:      meta,
		}
	}

	// TODO: add create many method to archivist with transaction
	for _, n := range dbNews {
		span = sentry.StartSpan(ctx, "News.Create", sentry.WithTransactionName("App.ComposeAndPostNews"))
		err := a.archivist.Entities.News.Create(ctx, &n)
		span.SetTag("news_id", n.ID.String())
		span.SetTag("news_hash", n.Hash)
		span.Finish()
		if err != nil {
			return errors.New(fmt.Sprintf("[News.Create]: %v", err))
		}
	}

	// Publish news to the channel and update news in the database
	for _, n := range dbNews {
		originalNews := news.FindById(n.Hash)
		if originalNews == nil {
			return errors.New(fmt.Sprintf("[publisher.Publish]: %v", err))
		}
		f := formatNews(&n, originalNews.ProviderName)

		span := sentry.StartSpan(ctx, "Publish", sentry.WithTransactionName("App.ComposeAndPostNews"))
		span.SetTag("news_hash", n.Hash)
		id, err := a.publisher.Publish(f)
		span.Finish()
		if err != nil {
			return err
		}

		// Update news with publication id
		n.PublicationID = id
		n.PublishedAt = time.Now()

		// TODO: add update many method to archivist with transaction
		span = sentry.StartSpan(ctx, "News.Update", sentry.WithTransactionName("App.ComposeAndPostNews"))
		span.SetTag("news_hash", n.Hash)
		err = a.archivist.Entities.News.Update(ctx, &n)
		span.Finish()
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

	span := sentry.StartSpan(ctx, "FindAllByHashes", sentry.WithTransactionName("App.RemoveDuplicates"))
	// TODO: Replace with ExistsByHashes
	exists, err := a.archivist.Entities.News.FindAllByHashes(ctx, hashes)
	span.Finish()
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
func formatNews(n *models.News, provider string) string {
	return fmt.Sprintf(
		"Hash: %s\nProvider: %s\nMeta: %s\nIsSuspicious:%v\n %s",
		n.Hash, provider, n.MetaData.String(), n.IsSuspicious, n.ComposedText,
	)
}

type JobFunc func()
