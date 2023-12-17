package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/getsentry/sentry-go"
	"github.com/samgozman/fin-thread/archivist"
	"github.com/samgozman/fin-thread/archivist/models"
	"github.com/samgozman/fin-thread/composer"
	"github.com/samgozman/fin-thread/journalist"
	"github.com/samgozman/fin-thread/publisher"
	"log/slog"
	"slices"
	"time"
)

// Job will be executed by the scheduler and will fetch, compose, publish and save news to the database
type Job struct {
	composer   *composer.Composer           // composer that will compose text for the article using OpenAI
	publisher  *publisher.TelegramPublisher // publisher that will publish news to the channel
	archivist  *archivist.Archivist         // archivist that will save news to the database
	journalist *journalist.Journalist       // journalist that will fetch news
	logger     *slog.Logger                 // special logger for the job
	options    *JobOptions                  // job options
}

// JobOptions holds job options needed for the job execution
type JobOptions struct {
	until              time.Time       // fetch articles until this date
	omitSuspicious     bool            // if true, will not publish suspicious articles
	omitEmptyMetaKeys  *omitKeyOptions // holds keys that will omit news if empty. Note: requires shouldComposeText to be true
	omitIfAllKeysEmpty bool            // if true, will omit articles with empty meta for all keys. Note: requires shouldComposeText to be set
	shouldComposeText  bool            // if true, will compose text for the article using OpenAI. If false, will use original title and description
	shouldSaveToDB     bool            // if true, will save all news to the database
	shouldRemoveClones bool            // if true, will remove duplicated news found in the DB. Note: requires shouldSaveToDB to be true
}

// NewJob creates a new Job instance
func NewJob(
	composer *composer.Composer,
	publisher *publisher.TelegramPublisher,
	archivist *archivist.Archivist,
	journalist *journalist.Journalist,
) *Job {
	return &Job{
		composer:   composer,
		publisher:  publisher,
		archivist:  archivist,
		journalist: journalist,
		logger:     slog.Default(),
		options:    &JobOptions{},
	}
}

// FetchUntil sets the date until which the articles will be fetched
func (job *Job) FetchUntil(until time.Time) *Job {
	job.options.until = until
	return job
}

// OmitSuspicious sets the flag that will omit suspicious articles
func (job *Job) OmitSuspicious() *Job {
	job.options.omitSuspicious = true
	return job
}

// OmitEmptyMeta will omit news with empty meta for the given key from composer.ComposedMeta
// Note: requires ComposeText to be set
func (job *Job) OmitEmptyMeta(key MetaKey) *Job {
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
		panic(errors.New(fmt.Sprintf("Unknown meta key: %s", key)))
	}
	return job
}

// OmitIfAllKeysEmpty will omit articles with empty meta for all keys from composer.ComposedMeta
//
// Example:
// "{"Markets": [], "Tickers": [], "Hashtags": []}" will be omitted,
// but "{"Markets": ["SPY"], "Tickers": [], "Hashtags": []}" will not
func (job *Job) OmitIfAllKeysEmpty() *Job {
	job.options.omitIfAllKeysEmpty = true
	return job
}

// ComposeText sets the flag that will compose text for the article using OpenAI
func (job *Job) ComposeText() *Job {
	job.options.shouldComposeText = true
	return job
}

// RemoveClones sets the flag that will remove duplicated news found in the DB
func (job *Job) RemoveClones() *Job {
	job.options.shouldRemoveClones = true
	return job
}

// SaveToDB sets the flag that will save all news to the database
func (job *Job) SaveToDB() *Job {
	job.options.shouldSaveToDB = true
	return job
}

// Run return job function that will be executed by the scheduler
func (job *Job) Run() JobFunc {
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
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

		span := tx.StartChild("GetLatestNews")
		news, err := job.journalist.GetLatestNews(ctx, job.options.until)
		span.Finish()
		if err != nil {
			job.logger.Info(fmt.Sprintf("[%s][GetLatestNews]", jobName), "error", err)
			hub.CaptureException(err)
		}

		hub.AddBreadcrumb(&sentry.Breadcrumb{
			Category: "successful",
			Message:  fmt.Sprintf("GetLatestNews returned %d news", len(news)),
			Level:    sentry.LevelInfo,
		}, nil)
		if len(news) == 0 {
			return
		}

		jobData := &JobData{
			News: news,
		}

		span = tx.StartChild("removeDuplicates")
		jobData.News, err = job.removeDuplicates(ctx, news)
		span.Finish()
		if err != nil {
			job.logger.Info(fmt.Sprintf("[%s][removeDuplicates]", jobName), "error", err)
			hub.CaptureException(err)
			return
		}
		hub.AddBreadcrumb(&sentry.Breadcrumb{
			Category: "successful",
			Message:  fmt.Sprintf("removeDuplicates returned %d news", len(jobData.News)),
			Level:    sentry.LevelInfo,
		}, nil)
		if len(jobData.News) == 0 {
			return
		}

		span = tx.StartChild("composeNews")
		jobData.ComposedNews, err = job.composeNews(ctx, jobData.News)
		span.Finish()
		if err != nil {
			job.logger.Warn(fmt.Sprintf("[%s][composeNews]", jobName), "error", err)
			hub.CaptureException(err)
			return
		}
		hub.AddBreadcrumb(&sentry.Breadcrumb{
			Category: "successful",
			Message:  fmt.Sprintf("composeNews returned %d news", len(jobData.ComposedNews)),
			Level:    sentry.LevelInfo,
		}, nil)

		span = tx.StartChild("saveNews")
		jobData.DBNews, err = job.saveNews(ctx, jobData)
		span.Finish()
		if err != nil {
			job.logger.Warn(fmt.Sprintf("[%s][saveNews]", jobName), "error", err)
			hub.CaptureException(err)
			return
		}
		hub.AddBreadcrumb(&sentry.Breadcrumb{
			Category: "successful",
			Message:  fmt.Sprintf("saveNews returned %d news", len(jobData.DBNews)),
			Level:    sentry.LevelInfo,
		}, nil)

		span = tx.StartChild("publish")
		jobData.DBNews, err = job.publish(ctx, jobData.DBNews)
		span.Finish()
		if err != nil {
			job.logger.Warn(fmt.Sprintf("[%s][publish]", jobName), "error", err)
			hub.CaptureException(err)
			return
		}
		hub.AddBreadcrumb(&sentry.Breadcrumb{
			Category: "successful",
			Message:  fmt.Sprintf("publish returned %d news", len(jobData.DBNews)),
			Level:    sentry.LevelInfo,
		}, nil)

		span = tx.StartChild("updateNews")
		err = job.updateNews(ctx, jobData.DBNews)
		span.Finish()
		if err != nil {
			job.logger.Warn(fmt.Sprintf("[%s][updateNews]", jobName), "error", err)
			hub.CaptureException(err)
			return
		}
		hub.AddBreadcrumb(&sentry.Breadcrumb{
			Category: "successful",
			Message:  "updateNews finished",
			Level:    sentry.LevelInfo,
		}, nil)
	}
}

// removeDuplicates removes duplicated news found in the DB
func (job *Job) removeDuplicates(ctx context.Context, news journalist.NewsList) (journalist.NewsList, error) {
	if !job.options.shouldRemoveClones || !job.options.shouldSaveToDB {
		return news, nil
	}

	hashes := make([]string, len(news))
	for i, n := range news {
		hashes[i] = n.ID
	}

	span := sentry.StartSpan(ctx, "FindAllByHashes", sentry.WithTransactionName("Job.removeDuplicates"))
	// TODO: Replace with ExistsByHashes
	exists, err := job.archivist.Entities.News.FindAllByHashes(ctx, hashes)
	span.Finish()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("[Job.removeDuplicates][News.FindAllByHashes]: %v", err))
	}
	existedHashes := make([]string, len(exists))
	for i, n := range exists {
		existedHashes[i] = n.Hash
	}

	var uniqueNews journalist.NewsList
	for _, n := range news {
		if !slices.Contains(existedHashes, n.ID) {
			uniqueNews = append(uniqueNews, n)
		}
	}
	return uniqueNews, nil
}

// composeNews composes text for the article using OpenAI and finds meta
func (job *Job) composeNews(ctx context.Context, news journalist.NewsList) ([]*composer.ComposedNews, error) {
	if !job.options.shouldComposeText {
		return nil, nil
	}

	// TODO: Split openai jobs - 1: remove unnecessary news, 2: compose text
	span := sentry.StartSpan(ctx, "Compose", sentry.WithTransactionName("Job.composeNews"))
	composedNews, err := job.composer.Compose(ctx, news)
	span.Finish()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("[Job.composeNews][composer.Compose]: %v", err))
	}

	return composedNews, nil
}

func (job *Job) saveNews(ctx context.Context, data *JobData) ([]*models.News, error) {
	if !job.options.shouldSaveToDB {
		return nil, nil
	}

	if len(data.News) < len(data.ComposedNews) {
		return nil, errors.New(fmt.Sprintf("[Job.saveNews]: Composed news count is more than original news count"))
	}

	// Map composed news by hash for convenience
	composedNewsMap := make(map[string]*composer.ComposedNews, len(data.ComposedNews))
	for _, n := range data.ComposedNews {
		composedNewsMap[n.ID] = n
	}

	dbNews := make([]*models.News, len(data.News))
	for i, n := range data.News {

		dbNews[i] = &models.News{
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
				return nil, errors.New(fmt.Sprintf("[Job.saveNews][json.Marshal] meta: %v", err))
			}

			dbNews[i].ComposedText = val.Text
			dbNews[i].MetaData = meta
		}
	}

	// TODO: add create many method to archivist with transaction
	for _, n := range dbNews {
		span := sentry.StartSpan(ctx, "News.Create", sentry.WithTransactionName("Job.saveNews"))
		err := job.archivist.Entities.News.Create(ctx, n)
		span.SetTag("news_id", n.ID.String())
		span.SetTag("news_hash", n.Hash)
		span.Finish()
		if err != nil {
			return nil, errors.New(fmt.Sprintf("[Job.saveNews][News.Create]: %v", err))
		}
	}

	return dbNews, nil
}

// publish publishes the news to the channel
func (job *Job) publish(ctx context.Context, dbNews []*models.News) ([]*models.News, error) {
	for _, n := range dbNews {
		// Skip suspicious news if needed
		if n.IsSuspicious && job.options.omitSuspicious {
			continue
		}

		// TODO: Change Unmarshal with find method among ComposedNews
		var meta composer.ComposedMeta
		err := json.Unmarshal(n.MetaData, &meta)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("[Job.publish][json.Unmarshal] meta: %v. Value: %s", err, n.MetaData))
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

		// Omit if all keys are empty and omitIfAllKeysEmpty is set
		if job.options.omitIfAllKeysEmpty {
			if len(meta.Tickers) == 0 && len(meta.Markets) == 0 && len(meta.Hashtags) == 0 {
				continue
			}
		}

		// Format news
		var formattedText string
		if job.options.shouldComposeText {
			formattedText = fmt.Sprintf(
				"Hash: %s\nProvider: %s\nMeta: %s\n %s",
				n.Hash, n.ProviderName, n.MetaData.String(), n.ComposedText,
			)
		} else {
			formattedText = n.OriginalTitle + "\n" + n.OriginalDesc
		}

		span := sentry.StartSpan(ctx, "Publish", sentry.WithTransactionName("Job.publish"))
		span.SetTag("news_hash", n.Hash)
		id, err := job.publisher.Publish(formattedText)
		span.Finish()

		if err != nil {
			return nil, errors.New(fmt.Sprintf("[Job.publish][publisher.Publish]: %v", err))
		}

		// Save publication data to the entity
		n.PublicationID = id
		n.PublishedAt = time.Now()
	}

	return dbNews, nil
}

// updateNews updates news in the database
func (job *Job) updateNews(ctx context.Context, dbNews []*models.News) error {
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
			return errors.New(fmt.Sprintf("[Job.updateNews][News.Update]: %v", err))
		}
	}

	return nil
}

// JobData holds different types of news data passed between the job functions just for convenience
type JobData struct {
	News         journalist.NewsList      // Original news fetched from the journalist
	ComposedNews []*composer.ComposedNews // Composed news with custom text and meta
	DBNews       []*models.News           // News entities from/for the database
}

// JobFunc is a type for job function that will be executed by the scheduler
type JobFunc func()

// MetaKey is a type for meta keys based on the keys from composer.ComposedMeta struct
type MetaKey string

// Based on the composer.ComposedMeta struct keys
const (
	MetaTickers  MetaKey = "Tickers"
	MetaMarkets  MetaKey = "Markets"
	MetaHashtags MetaKey = "Hashtags"
)

// omitKeyOptions holds keys that will omit news if empty. Note: requires JobOptions.shouldComposeText to be true
type omitKeyOptions struct {
	emptyTickers  bool // if true, will omit articles with empty tickers meta from composer.ComposedMeta
	emptyMarkets  bool // if true, will omit articles with empty markets meta from composer.ComposedMeta
	emptyHashtags bool // if true, will omit articles with empty hashtags meta from composer.ComposedMeta
}
