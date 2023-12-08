package main

import (
	"github.com/getsentry/sentry-go"
	"github.com/go-co-op/gocron"
	"github.com/spf13/viper"
	"log/slog"
	"os"
	"time"

	. "github.com/samgozman/fin-thread/archivist"
	. "github.com/samgozman/fin-thread/composer"
	. "github.com/samgozman/fin-thread/journalist"
	. "github.com/samgozman/fin-thread/publisher"
)

// Env is a structure that holds all the environment variables that are used in the app
type Env struct {
	TelegramChannelID string `mapstructure:"TELEGRAM_CHANNEL_ID"`
	TelegramBotToken  string `mapstructure:"TELEGRAM_BOT_TOKEN"`
	OpenAiToken       string `mapstructure:"OPENAI_TOKEN"`
	PostgresDSN       string `mapstructure:"POSTGRES_DSN"`
	SentryDSN         string `mapstructure:"SENTRY_DSN"`
}

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

	pub, err := NewTelegramPublisher(env.TelegramChannelID, env.TelegramBotToken)
	if err != nil {
		l.Error("[main] Error creating Telegram publisher:", err)
		os.Exit(1)
	}

	arch, err := NewArchivist(env.PostgresDSN)
	if err != nil {
		l.Error("[main] Error creating Archivist:", err)
		os.Exit(1)
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

	// TODO: move to config, this is just a test
	suspiciousKeywords := []string{"sign up", "buy now", "climate change", "activists", "advice", "covid-19", "study", "humanitarian", "award", "research"}

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
		NewRssProvider("trading-economics:food-inflation", "https://tradingeconomics.com/rss/news.aspx?i=food+inflation"),
		NewRssProvider("trading-economics:inflation-rate-mom", "https://tradingeconomics.com/rss/news.aspx?i=inflation+rate+mom"),
		NewRssProvider("trading-economics:core-inflation-rate-mom", "https://tradingeconomics.com/rss/news.aspx?i=core+inflation+rate+mom"),
		NewRssProvider("trading-economics:wholesale-prices-mom", "https://tradingeconomics.com/rss/news.aspx?i=wholesale+prices+mom"),
		NewRssProvider("trading-economics:weapons-sales", "https://tradingeconomics.com/rss/news.aspx?i=weapons+sales"),
		NewRssProvider("trading-economics:rent-inflation", "https://tradingeconomics.com/rss/news.aspx?i=rent+inflation"),
		NewRssProvider("trading-economics:housing-index", "https://tradingeconomics.com/rss/news.aspx?i=housing+index"),
		NewRssProvider("trading-economics:housing-starts", "https://tradingeconomics.com/rss/news.aspx?i=housing+starts"),
		NewRssProvider("trading-economics:households-debt-to-gdp", "https://tradingeconomics.com/rss/news.aspx?i=households+debt+to+gdp"),
		NewRssProvider("trading-economics:government-debt", "https://tradingeconomics.com/rss/news.aspx?i=government+debt"),
	}).FlagByKeys(suspiciousKeywords).Limit(1).FilterByKeys([]string{"European Union", "United States", "United Kingdom", "China", "Germany", "France", "Japan", "Italy", "India"})

	app := &App{
		composer:  NewComposer(env.OpenAiToken),
		publisher: pub,
		archivist: arch,
		logger:    slog.Default(),
	}

	s := gocron.NewScheduler(time.UTC)
	_, err = s.Every(60 * time.Second).Do(app.FinJob(marketJournalist, time.Now().Add(-60*time.Second)))
	if err != nil {
		return
	}

	_, err = s.Every(90 * time.Second).Do(app.FinJob(broadNews, time.Now().Add(-90*time.Second)))
	if err != nil {
		return
	}

	_, err = s.Every(5 * time.Minute).WaitForSchedule().Do(app.FinJob(teJournalist, time.Now().Add(-5*time.Minute)))
	if err != nil {
		return
	}

	defer s.Stop()
	s.StartAsync()

	l.Info("[main] Started fin-thread successfully")
	select {}
}
