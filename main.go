package main

import (
	"github.com/getsentry/sentry-go"
	"github.com/spf13/viper"
	"log/slog"
	"os"
	"time"
)

func main() {
	// Initialize viper
	viper.AddConfigPath(".")
	viper.SetConfigFile(".env")

	l := slog.Default()

	var env Env
	// Read the config file, if present
	err := viper.ReadInConfig()
	if err != nil {
		l.Info("[main] No config file found, reading from the system env")
		// TODO: fetch with viper, add validation
		env = Env{
			TelegramChannelID: os.Getenv("TELEGRAM_CHANNEL_ID"),
			TelegramBotToken:  os.Getenv("TELEGRAM_BOT_TOKEN"),
			OpenAiToken:       os.Getenv("OPENAI_TOKEN"),
			TogetherAIToken:   os.Getenv("TOGETHER_AI_TOKEN"),
			PostgresDSN:       os.Getenv("POSTGRES_DSN"),
			SentryDSN:         os.Getenv("SENTRY_DSN"),
		}
	} else {
		err = viper.Unmarshal(&env)
		if err != nil {
			l.Error("[main] Error unmarshalling config:", err)
			os.Exit(1)
		}
	}

	err = sentry.Init(sentry.ClientOptions{
		Dsn:                env.SentryDSN,
		EnableTracing:      true,
		TracesSampleRate:   1.0, // There are not many transactions, so we can afford to send all of them
		ProfilesSampleRate: 1.0, // Same here
	})
	if err != nil {
		l.Error("[main] Error initializing Sentry:", err)
		os.Exit(1)
	}
	defer sentry.Flush(2 * time.Second)

	app := &App{
		cnf: NewConfig(&env),
	}

	app.start()
}
