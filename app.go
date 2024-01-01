package main

import (
	"github.com/getsentry/sentry-go"
	"github.com/go-co-op/gocron/v2"
	. "github.com/samgozman/fin-thread/archivist"
	. "github.com/samgozman/fin-thread/composer"
	. "github.com/samgozman/fin-thread/jobs"
	. "github.com/samgozman/fin-thread/journalist"
	. "github.com/samgozman/fin-thread/publisher"
	"github.com/samgozman/fin-thread/scavenger"
	"log/slog"
	"time"
)

type App struct {
	cnf *Config // App configuration
}

func (a *App) start() {
	publisher, err := NewTelegramPublisher(a.cnf.env.TelegramChannelID, a.cnf.env.TelegramBotToken)
	if err != nil {
		slog.Default().Error("[main] Error creating Telegram publisher:", err)
		panic(err)
	}

	archivist, err := NewArchivist(a.cnf.env.PostgresDSN)
	if err != nil {
		slog.Default().Error("[main] Error creating Archivist:", err)
		panic(err)
	}

	composer := NewComposer(a.cnf.env.OpenAiToken, a.cnf.env.TogetherAIToken)

	marketJournalist := NewJournalist("MarketNews", []NewsProvider{
		NewRssProvider("benzinga:large-cap", "https://www.benzinga.com/news/large-cap/feed"),
		NewRssProvider("benzinga:mid-cap", "https://www.benzinga.com/news/mid-cap/feed"),
		NewRssProvider("benzinga:m&a", "https://www.benzinga.com/news/m-a/feed"),
		NewRssProvider("benzinga:buybacks", "https://www.benzinga.com/news/buybacks/feed"),
		NewRssProvider("benzinga:global", "https://www.benzinga.com/news/global/feed"),
		NewRssProvider("benzinga:sec", "https://www.benzinga.com/sec/feed"),
		NewRssProvider("benzinga:bonds", "https://www.benzinga.com/markets/bonds/feed"),
		NewRssProvider("benzinga:analyst:upgrades", "https://www.benzinga.com/analyst-ratings/upgrades/feed"),
		NewRssProvider("benzinga:analyst:downgrades", "https://www.benzinga.com/analyst-ratings/downgrades/feed"),
		NewRssProvider("benzinga:etfs", "https://www.benzinga.com/etfs/feed"),
	}).FlagByKeys(a.cnf.suspiciousKeywords).Limit(2)

	broadNews := NewJournalist("BroadNews", []NewsProvider{
		NewRssProvider("finpost:news", "https://financialpost.com/feed"),
	}).FlagByKeys(a.cnf.suspiciousKeywords).Limit(1)

	teJournalist := NewJournalist("TradingEconomics", []NewsProvider{
		NewRssProvider("trading-economics:european-union", "https://tradingeconomics.com/european-union/rss"),
		NewRssProvider("trading-economics:core-inflation-rate-mom", "https://tradingeconomics.com/rss/news.aspx?i=core+inflation+rate+mom"),
		NewRssProvider("trading-economics:wholesale-prices-mom", "https://tradingeconomics.com/rss/news.aspx?i=wholesale+prices+mom"),
		NewRssProvider("trading-economics:weapons-sales", "https://tradingeconomics.com/rss/news.aspx?i=weapons+sales"),
		NewRssProvider("trading-economics:housing-index", "https://tradingeconomics.com/rss/news.aspx?i=housing+index"),
		NewRssProvider("trading-economics:housing-starts", "https://tradingeconomics.com/rss/news.aspx?i=housing+starts"),
		NewRssProvider("trading-economics:households-debt-to-gdp", "https://tradingeconomics.com/rss/news.aspx?i=households+debt+to+gdp"),
		NewRssProvider("trading-economics:government-debt", "https://tradingeconomics.com/rss/news.aspx?i=government+debt"),
	}).FlagByKeys(a.cnf.suspiciousKeywords).Limit(1).FilterByKeys(a.cnf.filterKeys)

	marketJob := NewJob(composer, publisher, archivist, marketJournalist).
		FetchUntil(time.Now().Add(-60 * time.Second)).
		OmitSuspicious().
		OmitIfAllKeysEmpty().
		RemoveClones().
		ComposeText().
		SaveToDB()

	broadJob := NewJob(composer, publisher, archivist, broadNews).
		FetchUntil(time.Now().Add(-4 * time.Minute)).
		OmitSuspicious().
		OmitEmptyMeta(MetaTickers).
		RemoveClones().
		ComposeText().
		SaveToDB()

	teJob := NewJob(composer, publisher, archivist, teJournalist).
		FetchUntil(time.Now().Add(-5 * time.Minute)).
		OmitEmptyMeta(MetaHashtags).
		RemoveClones().
		ComposeText().
		SaveToDB()

	// Sentry hub for fatal errors
	hub := sentry.CurrentHub().Clone()
	hub.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetLevel(sentry.LevelFatal)
	})
	defer hub.Flush(2 * time.Second)

	s, err := gocron.NewScheduler()
	if err != nil {
		hub.AddBreadcrumb(&sentry.Breadcrumb{
			Category: "scheduler",
			Message:  "Error creating scheduler",
			Level:    sentry.LevelFatal,
		}, nil)
		hub.CaptureException(err)
		panic(err)
	}

	_, err = s.NewJob(
		gocron.DurationJob(60*time.Second),
		gocron.NewTask(marketJob.Run()),
		gocron.WithSingletonMode(gocron.LimitModeReschedule), // for often jobs
		gocron.WithName("scheduler for Market news"),
	)

	if err != nil {
		hub.AddBreadcrumb(&sentry.Breadcrumb{
			Category: "scheduler",
			Message:  "Error scheduling job for Market news",
			Level:    sentry.LevelFatal,
		}, nil)
		hub.CaptureException(err)
		panic(err)
	}

	_, err = s.NewJob(
		gocron.DurationJob(4*time.Minute),
		gocron.NewTask(broadJob.Run()),
		gocron.WithName("scheduler for Broad market news"),
	)
	if err != nil {
		hub.AddBreadcrumb(&sentry.Breadcrumb{
			Category: "scheduler",
			Message:  "Error scheduling job for Broad news",
			Level:    sentry.LevelFatal,
		}, nil)
		hub.CaptureException(err)
		panic(err)
	}

	_, err = s.NewJob(
		gocron.DurationJob(5*time.Minute),
		gocron.NewTask(teJob.Run()),
		gocron.WithName("scheduler for TradingEconomics news"),
	)
	if err != nil {
		sentry.AddBreadcrumb(&sentry.Breadcrumb{
			Category: "scheduler",
			Message:  "Error scheduling job for TradingEconomics",
			Level:    sentry.LevelFatal,
		})
		hub.CaptureException(err)
		panic(err)
	}

	// Calendar job
	scv := scavenger.Scavenger{}
	calJob := NewCalendarJob(
		scv.EconomicCalendar,
		publisher,
		archivist,
		"mql5-calendar",
	)
	_, err = s.NewJob(
		gocron.CronJob("0 6 * * 1", false), // every Monday at 6:00
		gocron.NewTask(calJob.RunWeeklyCalendarJob()),
		gocron.WithName("scheduler for Calendar"),
	)
	if err != nil {
		sentry.AddBreadcrumb(&sentry.Breadcrumb{
			Category: "scheduler",
			Message:  "Error scheduling job for Calendar",
			Level:    sentry.LevelFatal,
		})
		hub.CaptureException(err)
		panic(err)
	}

	_, err = s.NewJob(
		gocron.DurationJob(90*time.Second),
		gocron.NewTask(calJob.RunCalendarUpdatesJob()),
		gocron.WithName("scheduler for Calendar updates"),
	)
	if err != nil {
		sentry.AddBreadcrumb(&sentry.Breadcrumb{
			Category: "scheduler",
			Message:  "Error scheduling job for Calendar updates",
			Level:    sentry.LevelFatal,
		})
		hub.CaptureException(err)
		panic(err)
	}

	// Before market open job
	bmoJob := NewSummaryJob(
		composer,
		publisher,
		archivist,
	)
	_, err = s.NewJob(
		// TODO: Use holidays calendar to avoid unnecessary runs
		gocron.CronJob("0 14 * * 1-5", false), // every weekday at 14:00 UTC (market opens at 14:30 UTC)
		gocron.NewTask(bmoJob.Run(time.Now().Truncate(24*time.Hour))),
		gocron.WithName("scheduler for Before Market Open summary job"),
	)
	if err != nil {
		sentry.AddBreadcrumb(&sentry.Breadcrumb{
			Category: "scheduler",
			Message:  "Error scheduling job for Before Market Open",
			Level:    sentry.LevelFatal,
		})
		hub.CaptureException(err)
		panic(err)
	}

	defer func(s gocron.Scheduler) {
		err := s.Shutdown()
		if err != nil {
			panic(err)
		}
	}(s)
	s.Start()

	slog.Default().Info("Started fin-thread successfully")
	select {}
}
