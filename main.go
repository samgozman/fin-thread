package main

import (
	"github.com/getsentry/sentry-go"
	"github.com/go-playground/validator/v10"
	"log/slog"
	"os"
	"time"
)

func main() {
	l := slog.Default()

	env := Env{
		TelegramChannelID: os.Getenv("TELEGRAM_CHANNEL_ID"),
		TelegramBotToken:  os.Getenv("TELEGRAM_BOT_TOKEN"),
		OpenAiToken:       os.Getenv("OPENAI_TOKEN"),
		TogetherAIToken:   os.Getenv("TOGETHER_AI_TOKEN"),
		GoogleGeminiToken: os.Getenv("GOOGLE_GEMINI_TOKEN"),
		PostgresDSN:       os.Getenv("POSTGRES_DSN"),
		SentryDSN:         os.Getenv("SENTRY_DSN"),
		StockSymbols:      os.Getenv("STOCK_SYMBOLS"),
		MarketJournalists: os.Getenv("MARKET_JOURNALISTS"),
		BroadJournalists:  os.Getenv("BROAD_JOURNALISTS"),
		ServerName:        os.Getenv("SERVER_NAME"),
		ShouldPublish:     os.Getenv("SHOULD_PUBLISH") == "true",
	}
	validate := validator.New()
	if err := validate.Struct(env); err != nil {
		l.Error("[main] Error validating environment variables:", err)
		return
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:                env.SentryDSN,
		EnableTracing:      true,
		TracesSampleRate:   1.0, // There are not many transactions, so we can afford to send all of them
		ProfilesSampleRate: 1.0, // Same here
		ServerName:         env.ServerName,
	})
	if err != nil {
		l.Error("[main] Error initializing Sentry:", err)
		os.Exit(1)
	}
	defer sentry.Flush(2 * time.Second)
	defer sentry.Recover()

	cnf, err := NewConfig(&env)
	if err != nil {
		l.Error("[main] Error creating Config:", err)
		return
	}

	app := &App{
		cnf,
	}

	app.start()
}
