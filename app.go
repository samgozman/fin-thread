package main

import (
	"github.com/getsentry/sentry-go"
	"github.com/go-co-op/gocron"
	. "github.com/samgozman/fin-thread/archivist"
	. "github.com/samgozman/fin-thread/composer"
	. "github.com/samgozman/fin-thread/journalist"
	. "github.com/samgozman/fin-thread/publisher"
	"log/slog"
	"time"
)

type App struct {
	composer  *Composer
	publisher *TelegramPublisher
	archivist *Archivist
	logger    *slog.Logger
}

func (a *App) start() {
	// TODO: move to config, this is just a test
	suspiciousKeywords := []string{
		"sign up",
		"buy now",
		"climate",
		"activists",
		"activism",
		"advice",
		"covid-19",
		"study",
		"humanitarian",
		"award",
		"research",
		"human rights",
		"united nations",
		"adult content",
		"pornography",
		"porn",
		"sexually",
		"gender",
		"sexuality",
		"class action lawsuit",
		"subscribe",
	}
	filterKeys := []string{
		"European Union",
		"United States",
		"United Kingdom",
		"China",
		"Germany",
		"France",
		"Japan",
		"Italy",
		"India",
	}

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
	}).FlagByKeys(suspiciousKeywords).Limit(2)

	broadNews := NewJournalist("BroadNews", []NewsProvider{
		NewRssProvider("finpost:news", "https://financialpost.com/feed"),
	}).FlagByKeys(suspiciousKeywords).Limit(1)

	teJournalist := NewJournalist("TradingEconomics", []NewsProvider{
		NewRssProvider("trading-economics:european-union", "https://tradingeconomics.com/european-union/rss"),
		NewRssProvider("trading-economics:core-inflation-rate-mom", "https://tradingeconomics.com/rss/news.aspx?i=core+inflation+rate+mom"),
		NewRssProvider("trading-economics:wholesale-prices-mom", "https://tradingeconomics.com/rss/news.aspx?i=wholesale+prices+mom"),
		NewRssProvider("trading-economics:weapons-sales", "https://tradingeconomics.com/rss/news.aspx?i=weapons+sales"),
		NewRssProvider("trading-economics:housing-index", "https://tradingeconomics.com/rss/news.aspx?i=housing+index"),
		NewRssProvider("trading-economics:housing-starts", "https://tradingeconomics.com/rss/news.aspx?i=housing+starts"),
		NewRssProvider("trading-economics:households-debt-to-gdp", "https://tradingeconomics.com/rss/news.aspx?i=households+debt+to+gdp"),
		NewRssProvider("trading-economics:government-debt", "https://tradingeconomics.com/rss/news.aspx?i=government+debt"),
	}).FlagByKeys(suspiciousKeywords).Limit(1).FilterByKeys(filterKeys)

	marketJob := NewJob(a, marketJournalist).
		FetchUntil(time.Now().Add(-60 * time.Second)).
		OmitSuspicious().
		OmitIfAllKeysEmpty().
		RemoveClones().
		ComposeText().
		SaveToDB()

	broadJob := NewJob(a, broadNews).
		FetchUntil(time.Now().Add(-4 * time.Minute)).
		OmitSuspicious().
		OmitEmptyMeta(MetaTickers).
		RemoveClones().
		ComposeText().
		SaveToDB()

	teJob := NewJob(a, teJournalist).
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

	s := gocron.NewScheduler(time.UTC)
	_, err := s.Every(60 * time.Second).Do(marketJob.Run())
	if err != nil {
		hub.AddBreadcrumb(&sentry.Breadcrumb{
			Category: "scheduler",
			Message:  "Error scheduling job for Market news",
			Level:    sentry.LevelFatal,
		}, nil)
		hub.CaptureException(err)
		panic(err)
	}

	_, err = s.Every(4 * time.Minute).Do(broadJob.Run())
	if err != nil {
		hub.AddBreadcrumb(&sentry.Breadcrumb{
			Category: "scheduler",
			Message:  "Error scheduling job for Broad news",
			Level:    sentry.LevelFatal,
		}, nil)
		hub.CaptureException(err)
		panic(err)
	}

	_, err = s.Every(5 * time.Minute).WaitForSchedule().Do(teJob.Run())
	if err != nil {
		sentry.AddBreadcrumb(&sentry.Breadcrumb{
			Category: "scheduler",
			Message:  "Error scheduling job for TradingEconomics",
			Level:    sentry.LevelFatal,
		})
		hub.CaptureException(err)
		panic(err)
	}

	defer s.Stop()
	s.StartAsync()

	a.logger.Info("Started fin-thread successfully")
	select {}
}
