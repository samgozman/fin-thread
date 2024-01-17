package main

import (
	"context"
	"github.com/avast/retry-go"
	"github.com/getsentry/sentry-go"
	"github.com/go-co-op/gocron/v2"
	"github.com/samgozman/fin-thread/archivist"
	"github.com/samgozman/fin-thread/composer"
	"github.com/samgozman/fin-thread/jobs"
	"github.com/samgozman/fin-thread/journalist"
	"github.com/samgozman/fin-thread/publisher"
	"github.com/samgozman/fin-thread/scavenger"
	"github.com/samgozman/fin-thread/scavenger/stocks"
	"github.com/samgozman/fin-thread/utils"
	"log/slog"
	"time"
)

type App struct {
	cnf *Config // App configuration
}

func (a *App) start() {
	telegramPublisher, err := publisher.NewTelegramPublisher(a.cnf.env.TelegramChannelID, a.cnf.env.TelegramBotToken)
	if err != nil {
		slog.Default().Error("[main] Error creating Telegram telegramPublisher:", err)
		panic(err)
	}

	archivistEntity, err := archivist.NewArchivist(a.cnf.env.PostgresDSN)
	if err != nil {
		slog.Default().Error("[main] Error creating Archivist:", err)
		panic(err)
	}

	composerEntity := composer.NewComposer(a.cnf.env.OpenAiToken, a.cnf.env.TogetherAIToken, a.cnf.env.GoogleGeminiToken)

	marketJournalist := journalist.NewJournalist("MarketNews", a.cnf.rssProviders.marketJournalists).
		FlagByKeys(a.cnf.suspiciousKeywords).
		Limit(2)

	broadNews := journalist.NewJournalist("BroadNews", a.cnf.rssProviders.broadJournalists).
		FlagByKeys(a.cnf.suspiciousKeywords).
		Limit(1)

	// get all stockMap and pass as a parameter to jobs
	scv := scavenger.Scavenger{}
	var stockMap *stocks.StockMap
	err = retry.Do(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		stockMap, err = scv.Screener.FetchFromNasdaq(ctx)
		return err
	}, retry.Attempts(2), retry.Delay(5*time.Second))
	if err != nil {
		slog.Default().Error("[main] Error fetching stockMap:", err)

		// TODO: Find a reliable API source for this sorts of data
		// try to fill the gaps with static data
		stockMap = scv.Screener.FetchFromString(a.cnf.env.StockSymbols)
		if stockMap == nil {
			slog.Default().Error("[main] Error fetching stockMap from env")
		}
	}

	marketJob := jobs.NewJob(composerEntity, telegramPublisher, archivistEntity, marketJournalist, stockMap).
		FetchUntil(time.Now().Add(-60 * time.Second)).
		OmitSuspicious().
		OmitIfAllKeysEmpty().
		OmitUnlistedStocks().
		RemoveClones().
		ComposeText().
		SaveToDB().
		Publish()

	broadJob := jobs.NewJob(composerEntity, telegramPublisher, archivistEntity, broadNews, stockMap).
		FetchUntil(time.Now().Add(-4 * time.Minute)).
		OmitSuspicious().
		OmitEmptyMeta(jobs.MetaTickers).
		OmitUnlistedStocks().
		RemoveClones().
		ComposeText().
		SaveToDB().
		Publish()

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
		utils.CaptureSentryException("createSchedulerError", hub, err)
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
		utils.CaptureSentryException("createScheduleJobError", hub, err)
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
		utils.CaptureSentryException("createScheduleJobError", hub, err)
		panic(err)
	}

	// Calendar job
	calJob := jobs.NewCalendarJob(
		scv.EconomicCalendar,
		telegramPublisher,
		archivistEntity,
		"mql5-calendar",
	).Publish()

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
		utils.CaptureSentryException("createScheduleJobError", hub, err)
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
		utils.CaptureSentryException("createScheduleJobError", hub, err)
		panic(err)
	}

	// Before market open job
	bmoJob := jobs.NewSummaryJob(
		composerEntity,
		telegramPublisher,
		archivistEntity,
	).Publish()
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
		utils.CaptureSentryException("createScheduleJobError", hub, err)
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
