package jobs

import (
	"context"
	"fmt"
	"github.com/avast/retry-go"
	"github.com/getsentry/sentry-go"
	"github.com/samgozman/fin-thread/archivist"
	"github.com/samgozman/fin-thread/composer"
	"github.com/samgozman/fin-thread/publisher"
	"github.com/samgozman/fin-thread/utils"
	"log/slog"
	"strings"
	"time"
)

type SummaryJob struct {
	composer  *composer.Composer           // composer that will compose text for the article using OpenAI
	publisher *publisher.TelegramPublisher // publisher that will publish news to the channel
	archivist *archivist.Archivist         // archivist that will save news to the database
	logger    *slog.Logger                 // special logger for the job
	options   *summaryJobOptions           // options for the job
}

func NewSummaryJob(
	composer *composer.Composer,
	publisher *publisher.TelegramPublisher,
	archivist *archivist.Archivist,
) *SummaryJob {
	return &SummaryJob{
		composer:  composer,
		publisher: publisher,
		archivist: archivist,
		logger:    slog.Default(),
		options:   &summaryJobOptions{},
	}
}

// Publish sets the flag that will publish summary to the channel. Else: will just print them to the console (for development).
func (j *SummaryJob) Publish() *SummaryJob {
	j.options.shouldPublish = true
	return j
}

type summaryJobOptions struct {
	shouldPublish bool // if true, will publish news to the channel. Else: will just print them to the console (for development)
}

// Run runs the Summary job. From if the time from which events should be processed.
func (j *SummaryJob) Run(from time.Time) JobFunc {
	return func() {
		_ = retry.Do(func() error {
			ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
			defer cancel()

			tx := sentry.StartTransaction(ctx, "RunSummaryJob")
			tx.Op = "job-summary"

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

			// Fetch news from the database
			span := sentry.StartSpan(ctx, "News.FindAllUntilDate", sentry.WithTransactionName("SummaryJob.Run"))
			news, err := j.archivist.Entities.News.FindAllUntilDate(ctx, from)
			span.Finish()
			if err != nil {
				e := fmt.Errorf("error fetching news from the database: %w", err)
				j.logger.Error(e.Error())
				hub.AddBreadcrumb(&sentry.Breadcrumb{
					Category: "database",
					Message:  "Error fetching news from the database",
					Level:    sentry.LevelError,
				}, nil)
				utils.CaptureSentryException("jobSummaryNewsFindAllError", hub, e)
				return e
			}
			hub.AddBreadcrumb(&sentry.Breadcrumb{
				Category: "successful",
				Message:  fmt.Sprintf("News.FindAllUntilDate returned %d news", len(news)),
				Level:    sentry.LevelInfo,
			}, nil)

			// Find all events
			span = sentry.StartSpan(ctx, "Events.FindAllUntilDate", sentry.WithTransactionName("SummaryJob.Run"))
			events, err := j.archivist.Entities.Events.FindAllUntilDate(ctx, from)
			span.Finish()
			if err != nil {
				e := fmt.Errorf("error fetching events from the database: %w", err)
				j.logger.Error(e.Error())
				hub.AddBreadcrumb(&sentry.Breadcrumb{
					Category: "database",
					Message:  "Error fetching events from the database",
					Level:    sentry.LevelError,
				}, nil)
				utils.CaptureSentryException("jobSummaryEventsFindAllError", hub, e)
				return e
			}

			hub.AddBreadcrumb(&sentry.Breadcrumb{
				Category: "successful",
				Message:  fmt.Sprintf("Events.FindAllUntilDate returned %d events", len(events)),
				Level:    sentry.LevelInfo,
			}, nil)

			if sum := len(events) + len(news); sum < 5 {
				j.logger.Info("No news or events to process (or total < 5)")
				hub.AddBreadcrumb(&sentry.Breadcrumb{
					Category: "successful",
					Message:  fmt.Sprintf("Sum of news & events = %d, which is below summary threshold (5). ", sum),
					Level:    sentry.LevelDebug,
				}, nil)
				return nil
			}

			var headlines []*composer.Headline
			for _, e := range events {
				headlines = append(headlines, e.ToHeadline())
			}
			for _, n := range news {
				headlines = append(headlines, n.ToHeadline())
			}

			span = sentry.StartSpan(ctx, "Summarise", sentry.WithTransactionName("SummaryJob.Run"))
			summarised, err := j.composer.Summarise(ctx, headlines, 20, 2048)
			span.Finish()
			if err != nil {
				e := fmt.Errorf("error summarising news: %w", err)
				j.logger.Error(e.Error())
				hub.AddBreadcrumb(&sentry.Breadcrumb{
					Category: "composer",
					Message:  "Error composing summary",
					Level:    sentry.LevelError,
				}, nil)
				utils.CaptureSentryException("jobSummaryComposerSummariseError", hub, e)
				return e
			}
			if len(summarised) == 0 {
				j.logger.Info("No summarised news")
				hub.AddBreadcrumb(&sentry.Breadcrumb{
					Category: "debug",
					Message:  "No summarised news",
					Level:    sentry.LevelDebug,
				}, nil)
				return nil
			}

			hub.AddBreadcrumb(&sentry.Breadcrumb{
				Category: "successful",
				Message:  fmt.Sprintf("composer.Summarise returned %d headlines", len(summarised)),
				Level:    sentry.LevelInfo,
			}, nil)

			message := formatSummary(summarised, from)
			if message == "" {
				j.logger.Info("No summary message")
				hub.AddBreadcrumb(&sentry.Breadcrumb{
					Category: "debug",
					Message:  "No summary message",
					Level:    sentry.LevelDebug,
				}, nil)
				return nil
			}

			if !j.options.shouldPublish {
				fmt.Println(message)
				return nil
			}

			// Publish summary to the channel
			span = sentry.StartSpan(ctx, "Publish", sentry.WithTransactionName("SummaryJob.Run"))
			_, err = j.publisher.Publish(message)
			span.Finish()
			if err != nil {
				e := fmt.Errorf("error publishing summary: %w", err)
				j.logger.Error(e.Error())
				hub.AddBreadcrumb(&sentry.Breadcrumb{
					Category: "publisher",
					Message:  "Error publishing summary",
					Level:    sentry.LevelError,
				}, nil)
				utils.CaptureSentryException("jobSummaryPublishError", hub, e)
				// Note: Unrecoverable error, because Telegram API often hangs up, but somehow publishes the message
				return retry.Unrecoverable(e) //nolint:wrapcheck
			}

			hub.AddBreadcrumb(&sentry.Breadcrumb{
				Category: "successful",
				Message:  "Summary published successfully",
				Level:    sentry.LevelInfo,
			}, nil)

			// TODO: Save or not to save summary to db?
			return nil
		},
			retry.Attempts(5),
			retry.Delay(10*time.Minute),
		)
	}
}

func formatSummary(headlines []*composer.SummarisedHeadline, from time.Time) string {
	if len(headlines) == 0 {
		return ""
	}

	hours := int(time.Since(from).Hours())

	message := fmt.Sprintf("📓 #summary\nWhat happened in the last %d hours:\n", hours)

	for _, h := range headlines {
		m := fmt.Sprintf("- %s\n", h.Summary)
		if h.Link != "" && h.Verb != "" {
			m = strings.Replace(m, h.Verb, fmt.Sprintf("[%s](%s)", h.Verb, h.Link), 1)
		}
		message += m
	}

	return message
}
