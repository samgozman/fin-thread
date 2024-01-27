package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/getsentry/sentry-go"
	"github.com/samgozman/fin-thread/archivist"
	"github.com/samgozman/fin-thread/composer"
	"github.com/samgozman/fin-thread/internal/utils"
	"github.com/samgozman/fin-thread/journalist"
	"github.com/samgozman/fin-thread/publisher"
	"github.com/samgozman/fin-thread/scavenger/stocks"
	"log/slog"
	"slices"
	"strings"
	"time"
)

// Job will be executed by the scheduler and will fetch, compose, publish and save news to the database.
type Job struct {
	composer   *composer.Composer           // composer that will compose text for the article using OpenAI
	publisher  *publisher.TelegramPublisher // publisher that will publish news to the channel
	archivist  *archivist.Archivist         // archivist that will save news to the database
	journalist *journalist.Journalist       // journalist that will fetch news
	stocks     *stocks.StockMap             // stocks that will be used to filter news and compose meta (optional). TODO: use more fields from Stock struct
	logger     *slog.Logger                 // special logger for the job
	options    *jobOptions                  // job options
}

// jobOptions holds job options needed for the job execution.
type jobOptions struct {
	until              time.Time       // fetch articles until this date
	omitSuspicious     bool            // if true, will not publish suspicious articles
	omitEmptyMetaKeys  *omitKeyOptions // holds keys that will omit news if empty. Note: requires shouldComposeText to be true
	omitIfAllKeysEmpty bool            // if true, will omit articles with empty meta for all keys. Note: requires shouldComposeText to be set
	omitUnlistedStocks bool            // if true, will omit articles with stocks unlisted in the Job.stocks
	shouldComposeText  bool            // if true, will compose text for the article using OpenAI. If false, will use original title and description
	shouldSaveToDB     bool            // if true, will save all news to the database
	shouldRemoveClones bool            // if true, will remove duplicated news found in the DB. Note: requires shouldSaveToDB to be true
}

// NewJob creates a new Job instance.
func NewJob(
	composer *composer.Composer,
	publisher *publisher.TelegramPublisher,
	archivist *archivist.Archivist,
	journalist *journalist.Journalist,
	stocks *stocks.StockMap,
) *Job {
	return &Job{
		composer:   composer,
		publisher:  publisher,
		archivist:  archivist,
		journalist: journalist,
		stocks:     stocks,
		logger:     slog.Default(),
		options:    &jobOptions{},
	}
}

// FetchUntil sets the date until which the articles will be fetched.
func (job *Job) FetchUntil(until time.Time) *Job {
	job.options.until = until
	return job
}

// OmitSuspicious sets the flag that will omit suspicious articles.
func (job *Job) OmitSuspicious() *Job {
	job.options.omitSuspicious = true
	return job
}

// OmitEmptyMeta will omit news with empty meta for the given key from composer.ComposedMeta.
// Note: requires ComposeText to be set.
func (job *Job) OmitEmptyMeta(key metaKey) *Job {
	if job.options.omitEmptyMetaKeys == nil {
		job.options.omitEmptyMetaKeys = &omitKeyOptions{}
	}
	switch key {
	case MetaTickers:
		job.options.omitEmptyMetaKeys.emptyTickers = true
	case MetaMarkets:
		job.options.omitEmptyMetaKeys.emptyMarkets = true
	case MetaHashtags:
		job.options.omitEmptyMetaKeys.emptyHashtags = true
	default:
		panic(fmt.Errorf("unknown meta key: %s", key))
	}
	return job
}

// OmitIfAllKeysEmpty will omit articles with empty meta for all keys from composer.ComposedMeta.
//
// Example:
// "{"Markets": [], "Tickers": [], "Hashtags": []}" will be omitted,
// but "{"Markets": ["SPY"], "Tickers": [], "Hashtags": []}" will not.
func (job *Job) OmitIfAllKeysEmpty() *Job {
	job.options.omitIfAllKeysEmpty = true
	return job
}

// ComposeText sets the flag that will compose text for the article using OpenAI.
func (job *Job) ComposeText() *Job {
	job.options.shouldComposeText = true
	return job
}

// RemoveClones sets the flag that will remove duplicated news found in the DB.
func (job *Job) RemoveClones() *Job {
	job.options.shouldRemoveClones = true
	return job
}

// SaveToDB sets the flag that will save all news to the database.
func (job *Job) SaveToDB() *Job {
	job.options.shouldSaveToDB = true
	return job
}

// OmitUnlistedStocks sets the flag that will omit articles publishing with stocks unlisted in the Job.stocks.
func (job *Job) OmitUnlistedStocks() *Job {
	job.options.omitUnlistedStocks = true
	return job
}

// Run return job function that will be executed by the scheduler.
func (job *Job) Run() JobFunc {
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()

		jobName := fmt.Sprintf("Run.%s", job.journalist.Name)

		tx := sentry.StartTransaction(ctx, fmt.Sprintf("Job.%s", jobName))
		tx.Op = "job"

		// Sentry performance monitoring
		hub := sentry.GetHubFromContext(ctx)
		if hub == nil {
			hub = sentry.CurrentHub().Clone()
			ctx = sentry.SetHubOnContext(ctx, hub)
		}

		defer func() {
			tx.Finish()
			hub.Flush(2 * time.Second)
		}()

		// Note: all functions are wrapped with sentry.Span for performance monitoring.
		// Placing them in separate functions for better readability (experimental).

		getLatestNews := func() journalist.NewsList {
			span := tx.StartChild("GetLatestNews")
			news, err := job.journalist.GetLatestNews(ctx, job.options.until)
			span.Finish()
			if err != nil {
				e := fmt.Errorf("[%s][GetLatestNews]: %w", jobName, err)
				job.logger.Info(e.Error())
				utils.CaptureSentryException("jobGetLatestNewsError", hub, e)
			}

			hub.AddBreadcrumb(&sentry.Breadcrumb{
				Category: "successful",
				Message:  fmt.Sprintf("GetLatestNews returned %d news", len(news)),
				Level:    sentry.LevelInfo,
			}, nil)

			return news
		}
		removeDuplicates := func(news journalist.NewsList) (journalist.NewsList, error) {
			span := tx.StartChild("removeDuplicates")
			news, err := job.removeDuplicates(ctx, news)
			span.Finish()
			if err != nil {
				e := fmt.Errorf("[%s][removeDuplicates]: %w", jobName, err)
				job.logger.Info(e.Error())
				utils.CaptureSentryException("jobRemoveDuplicatesError", hub, e)
				return nil, e
			}
			hub.AddBreadcrumb(&sentry.Breadcrumb{
				Category: "successful",
				Message:  fmt.Sprintf("removeDuplicates returned %d news", len(news)),
				Level:    sentry.LevelInfo,
			}, nil)

			return news, nil
		}
		filterByComposer := func(news journalist.NewsList) (journalist.NewsList, error) {
			span := tx.StartChild("filter")
			news, err := job.composer.Filter(ctx, news)
			span.Finish()
			if err != nil {
				e := fmt.Errorf("[%s][Filter]: %w", jobName, err)
				job.logger.Info(e.Error())
				utils.CaptureSentryException("jobComposerFilterError", hub, e)
				return nil, e
			}
			hub.AddBreadcrumb(&sentry.Breadcrumb{
				Category: "successful",
				Message:  fmt.Sprintf("filter returned %d news", len(news)),
				Level:    sentry.LevelInfo,
			}, nil)

			return news, nil
		}
		composeNews := func(news journalist.NewsList) ([]*composer.ComposedNews, error) {
			span := tx.StartChild("composeNews")
			composedNews, err := job.composeNews(ctx, news)
			span.Finish()
			if err != nil {
				job.logger.Warn(fmt.Sprintf("[%s][composeNews]", jobName), "error", err)
				utils.CaptureSentryException("jobComposeNewsError", hub, err)
				return nil, err
			}
			hub.AddBreadcrumb(&sentry.Breadcrumb{
				Category: "successful",
				Message:  fmt.Sprintf("composeNews returned %d news", len(composedNews)),
				Level:    sentry.LevelInfo,
			}, nil)

			return composedNews, nil
		}
		saveNews := func(news journalist.NewsList, composedNews []*composer.ComposedNews) ([]*archivist.News, error) {
			span := tx.StartChild("saveNews")
			dbNews, err := job.saveNews(ctx, news, composedNews)
			span.Finish()
			if err != nil {
				job.logger.Warn(fmt.Sprintf("[%s][saveNews]", jobName), "error", err)
				utils.CaptureSentryException("jobSaveNewsError", hub, err)
				return nil, err
			}
			hub.AddBreadcrumb(&sentry.Breadcrumb{
				Category: "successful",
				Message:  fmt.Sprintf("saveNews returned %d news", len(dbNews)),
				Level:    sentry.LevelInfo,
			}, nil)

			return dbNews, nil
		}
		prepublishFilter := func(dbNews []*archivist.News) ([]*archivist.News, error) {
			span := tx.StartChild("prepublishFilter")
			filteredNews, err := job.prepublishFilter(dbNews)
			span.Finish()
			if err != nil {
				job.logger.Warn(fmt.Sprintf("[%s][prepublishFilter]", jobName), "error", err)
				utils.CaptureSentryException("jobPrepublishFilterError", hub, err)
				return nil, err
			}
			hub.AddBreadcrumb(&sentry.Breadcrumb{
				Category: "successful",
				Message:  fmt.Sprintf("prepublishFilter returned %d news", len(filteredNews)),
				Level:    sentry.LevelInfo,
			}, nil)

			return filteredNews, nil
		}
		publishNews := func(filteredNews []*archivist.News) ([]*archivist.News, error) {
			span := tx.StartChild("publishNews")
			publishedNews, err := job.publish(ctx, filteredNews)
			span.Finish()
			if err != nil {
				job.logger.Warn(fmt.Sprintf("[%s][publishNews]", jobName), "error", err)
				utils.CaptureSentryException("jobPublishError", hub, err)
				return nil, err
			}
			hub.AddBreadcrumb(&sentry.Breadcrumb{
				Category: "successful",
				Message:  fmt.Sprintf("publishNews returned %d news", len(publishedNews)),
				Level:    sentry.LevelInfo,
			}, nil)

			return publishedNews, nil
		}
		updateNews := func(publishedNews []*archivist.News) error {
			span := tx.StartChild("updateNews")
			err := job.updateNews(ctx, publishedNews)
			span.Finish()
			if err != nil {
				job.logger.Warn(fmt.Sprintf("[%s][updateNews]", jobName), "error", err)
				utils.CaptureSentryException("jobUpdateNewsError", hub, err)
				return err
			}
			hub.AddBreadcrumb(&sentry.Breadcrumb{
				Category: "successful",
				Message:  "updateNews finished",
				Level:    sentry.LevelInfo,
			}, nil)

			return nil
		}

		news := getLatestNews()
		if len(news) == 0 {
			return
		}

		news, err := removeDuplicates(news)
		if err != nil || len(news) == 0 {
			return
		}

		news, err = filterByComposer(news)
		if err != nil || len(news) == 0 {
			return
		}

		composedNews, err := composeNews(news)
		if err != nil || len(composedNews) == 0 {
			return
		}

		dbNews, err := saveNews(news, composedNews)
		if err != nil || len(dbNews) == 0 {
			return
		}

		filteredNews, err := prepublishFilter(dbNews)
		if err != nil || len(filteredNews) == 0 {
			return
		}

		publishedNews, err := publishNews(filteredNews)
		if err != nil || len(publishedNews) == 0 {
			return
		}

		err = updateNews(publishedNews)
		if err != nil {
			return
		}
	}
}

// removeDuplicates removes duplicated news in place found in the DB.
func (job *Job) removeDuplicates(ctx context.Context, news journalist.NewsList) (journalist.NewsList, error) {
	if !job.options.shouldRemoveClones || !job.options.shouldSaveToDB {
		return nil, nil
	}

	txName := sentry.WithTransactionName("Job.removeDuplicates")
	span := sentry.StartSpan(ctx, "FindAllByHashes", txName)

	hashes := make([]string, len(news))
	for i, n := range news {
		hashes[i] = n.ID
	}

	// TODO: Replace with ExistsByHashes
	existsByHash, err := job.archivist.Entities.News.FindAllByHashes(ctx, hashes)
	span.Finish()
	if err != nil {
		return nil, fmt.Errorf("[Job.removeDuplicates][News.FindAllByHashes]: %w", err)
	}

	span = sentry.StartSpan(ctx, "FindAllByUrls", txName)

	urls := make([]string, len(news))
	for i, n := range news {
		urls[i] = n.Link
	}

	existsByURL, err := job.archivist.Entities.News.FindAllByUrls(ctx, urls)
	if err != nil {
		return nil, fmt.Errorf("[Job.removeDuplicates][News.FindAllByUrls]: %w", err)
	}

	span.Finish()

	// Create array of hashes and urls of existed news for convenience
	existedHashes := make([]string, len(existsByHash))
	for i, n := range existsByHash {
		existedHashes[i] = n.Hash
	}
	existedUrls := make([]string, len(existsByURL))
	for i, n := range existsByURL {
		existedUrls[i] = n.Hash
	}

	var result journalist.NewsList

	// create array without duplicates
	for _, n := range news {
		if !slices.Contains(existedHashes, n.ID) && !slices.Contains(existedUrls, n.Link) {
			result = append(result, n)
		}
	}

	return result, nil
}

// composeNews composes text for the article using OpenAI and finds meta.
func (job *Job) composeNews(ctx context.Context, news journalist.NewsList) ([]*composer.ComposedNews, error) {
	if !job.options.shouldComposeText {
		return nil, nil
	}

	// TODO: Split openai jobs - 1: remove unnecessary news, 2: compose text
	span := sentry.StartSpan(ctx, "Compose", sentry.WithTransactionName("Job.composeNews"))
	composedNews, err := job.composer.Compose(ctx, news)
	span.Finish()
	if err != nil {
		return nil, fmt.Errorf("[Job.composeNews][composer.Compose]: %w", err)
	}

	return composedNews, nil
}

func (job *Job) saveNews(
	ctx context.Context,
	news journalist.NewsList,
	composedNews []*composer.ComposedNews,
) ([]*archivist.News, error) {
	if !job.options.shouldSaveToDB {
		return nil, nil
	}

	if len(news) < len(composedNews) {
		return nil, errors.New("[Job.saveNews]: Composed news count is more than original news count")
	}

	// Map composed news by hash for convenience
	composedNewsMap := make(map[string]*composer.ComposedNews, len(composedNews))
	for _, n := range composedNews {
		composedNewsMap[n.ID] = n
	}

	dbNews := make([]*archivist.News, len(news))
	for i, n := range news {
		dbNews[i] = &archivist.News{
			Hash:          n.ID,
			ChannelID:     job.publisher.ChannelID,
			ProviderName:  n.ProviderName,
			OriginalTitle: n.Title,
			OriginalDesc:  n.Description,
			OriginalDate:  n.Date,
			URL:           n.Link,
			IsSuspicious:  n.IsSuspicious,
		}

		// Save composed text and meta if found in the map
		if val, ok := composedNewsMap[n.ID]; ok {
			meta, err := json.Marshal(composer.ComposedMeta{
				Tickers:  val.Tickers,
				Markets:  val.Markets,
				Hashtags: val.Hashtags,
			})
			if err != nil {
				return nil, fmt.Errorf("[Job.saveNews][json.Marshal] meta: %w", err)
			}

			dbNews[i].ComposedText = val.Text
			dbNews[i].MetaData = meta
		}
	}

	span := sentry.StartSpan(ctx, "News.Create", sentry.WithTransactionName("Job.saveNews"))
	err := job.archivist.Entities.News.Create(ctx, dbNews)
	span.Finish()
	if err != nil {
		return nil, fmt.Errorf("[Job.saveNews][News.Create]: %w", err)
	}

	return dbNews, nil
}

// prepublishFilter final filter before publishing which will use all options and gathered info from previous steps.
func (job *Job) prepublishFilter(news []*archivist.News) ([]*archivist.News, error) {
	filteredNews := make([]*archivist.News, 0, len(news))

NewsRange:
	for _, n := range news {
		// Skip suspicious news if needed
		if n.IsSuspicious && job.options.omitSuspicious {
			continue
		}

		// TODO: Change Unmarshal with find method among ComposedNews
		var meta composer.ComposedMeta
		err := json.Unmarshal(n.MetaData, &meta)
		if err != nil {
			return nil, fmt.Errorf("[Job.publish][json.Unmarshal] meta: %w. Value: %v", err, n.MetaData)
		}

		// Skip news with empty meta if needed
		if job.options.omitEmptyMetaKeys != nil {
			if job.options.omitEmptyMetaKeys.emptyTickers && len(meta.Tickers) == 0 {
				continue
			}
			if job.options.omitEmptyMetaKeys.emptyMarkets && len(meta.Markets) == 0 {
				continue
			}
			if job.options.omitEmptyMetaKeys.emptyHashtags && len(meta.Hashtags) == 0 {
				continue
			}
		}

		// Skip news with unlisted stocks if needed
		if job.options.omitUnlistedStocks && job.stocks != nil && len(meta.Tickers) > 0 {
			for _, t := range meta.Tickers {
				if _, ok := (*job.stocks)[t]; !ok {
					continue NewsRange
				}
			}
		}

		// Omit if all keys are empty and omitIfAllKeysEmpty is set
		if job.options.omitIfAllKeysEmpty &&
			len(meta.Tickers) == 0 &&
			len(meta.Markets) == 0 &&
			len(meta.Hashtags) == 0 {
			continue
		}

		filteredNews = append(filteredNews, n)
	}

	return filteredNews, nil
}

// publish publishes the news to the channel and updates dbNews with PublicationID and PublishedAt fields.
func (job *Job) publish(ctx context.Context, news []*archivist.News) ([]*archivist.News, error) {
	updatedNews := make([]*archivist.News, 0, len(news))

	for _, n := range news {
		// Format news
		var formattedText string
		if job.options.shouldComposeText {
			formattedText = formatNewsWithComposedMeta(*n)
		} else {
			formattedText = n.OriginalTitle + "\n" + n.OriginalDesc
		}

		span := sentry.StartSpan(ctx, "Publish", sentry.WithTransactionName("Job.publish"))
		span.SetTag("news_hash", n.Hash)
		id, err := job.publisher.Publish(formattedText)
		span.Finish()

		if err != nil {
			return nil, fmt.Errorf("[Job.publish][publisher.Publish]: %w", err)
		}

		// Save publication data to the entity
		n.PublicationID = id
		n.PublishedAt = time.Now()

		updatedNews = append(updatedNews, n)
	}

	return updatedNews, nil
}

// updateNews updates news in the database.
func (job *Job) updateNews(ctx context.Context, dbNews []*archivist.News) error {
	if !job.options.shouldSaveToDB {
		return nil
	}

	for _, n := range dbNews {
		// TODO: add update many method to archivist with transaction
		span := sentry.StartSpan(ctx, "News.Update", sentry.WithTransactionName("Job.updateNews"))
		span.SetTag("news_hash", n.Hash)
		err := job.archivist.Entities.News.Update(ctx, n)
		span.Finish()
		if err != nil {
			return fmt.Errorf("[Job.updateNews][News.Update]: %w", err)
		}
	}

	return nil
}

func formatNewsWithComposedMeta(n archivist.News) string {
	if n.MetaData == nil {
		return n.ComposedText
	}

	var meta composer.ComposedMeta
	err := json.Unmarshal(n.MetaData, &meta)
	if err != nil {
		return n.ComposedText
	}

	result := n.ComposedText
	for _, t := range meta.Tickers {
		result = strings.Replace(result, t, fmt.Sprintf("[%s](https://short-fork.extr.app/en/%s?utm_source=finthread)", t, t), 1)
	}

	// TODO: Decide what to do with markets and hashtags

	return result
}

// JobFunc is a type for job function that will be executed by the scheduler.
type JobFunc func()

// metaKey is a type for meta keys based on the keys from composer.ComposedMeta struct.
type metaKey string

// Based on the composer.ComposedMeta struct keys.
const (
	MetaTickers  metaKey = "Tickers"
	MetaMarkets  metaKey = "Markets"
	MetaHashtags metaKey = "Hashtags"
)

// omitKeyOptions holds keys that will omit news if empty. Note: requires jobOptions.shouldComposeText to be true.
type omitKeyOptions struct {
	emptyTickers  bool // if true, will omit articles with empty tickers meta from composer.ComposedMeta
	emptyMarkets  bool // if true, will omit articles with empty markets meta from composer.ComposedMeta
	emptyHashtags bool // if true, will omit articles with empty hashtags meta from composer.ComposedMeta
}
