package jobs

import (
	"context"
	"errors"
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
				j.logger.Error("Error fetching news from the database", err)
				hub.AddBreadcrumb(&sentry.Breadcrumb{
					Category: "database",
					Message:  "Error fetching news from the database",
					Level:    sentry.LevelError,
				}, nil)
				utils.CaptureSentryException("jobSummaryNewsFindAllError", hub, err)
				return err
			}

			// Find all events
			span = sentry.StartSpan(ctx, "Events.FindAllUntilDate", sentry.WithTransactionName("SummaryJob.Run"))
			events, err := j.archivist.Entities.Events.FindAllUntilDate(ctx, from)
			span.Finish()
			if err != nil {
				j.logger.Error("Error fetching events from the database", err)
				hub.AddBreadcrumb(&sentry.Breadcrumb{
					Category: "database",
					Message:  "Error fetching events from the database",
					Level:    sentry.LevelError,
				}, nil)
				utils.CaptureSentryException("jobSummaryEventsFindAllError", hub, err)
				return err
			}

			if len(events)+len(news) < 5 {
				j.logger.Info("No news or events to process (or total < 5)")
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
				j.logger.Error("Error composing summary", err)
				hub.AddBreadcrumb(&sentry.Breadcrumb{
					Category: "composer",
					Message:  "Error composing summary",
					Level:    sentry.LevelError,
				}, nil)
				utils.CaptureSentryException("jobSummaryComposerSummariseError", hub, err)
				return err
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
				j.logger.Error("Error publishing summary", err)
				hub.AddBreadcrumb(&sentry.Breadcrumb{
					Category: "publisher",
					Message:  "Error publishing summary",
					Level:    sentry.LevelError,
				}, nil)
				utils.CaptureSentryException("jobSummaryPublishError", hub, err)
				// Note: Unrecoverable error, because Telegram API often hangs up, but somehow publishes the message
				return retry.Unrecoverable(errors.New("publishing error"))
			}

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
