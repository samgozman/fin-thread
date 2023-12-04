package main

import (
	"github.com/go-co-op/gocron"
	"github.com/spf13/viper"
	"log"
	"os"
	"time"

	. "github.com/samgozman/go-fin-feed/archivist"
	. "github.com/samgozman/go-fin-feed/composer"
	. "github.com/samgozman/go-fin-feed/journalist"
	. "github.com/samgozman/go-fin-feed/publisher"
)

// Env is a structure that holds all the environment variables that are used in the app
type Env struct {
	TelegramChannelID string `mapstructure:"TELEGRAM_CHANNEL_ID"`
	TelegramBotToken  string `mapstructure:"TELEGRAM_BOT_TOKEN"`
	OpenAiToken       string `mapstructure:"OPENAI_TOKEN"`
	PostgresDSN       string `mapstructure:"POSTGRES_DSN"`
}

func main() {
	// Initialize viper
	viper.AddConfigPath(".")
	viper.SetConfigFile(".env")

	l := log.Default()
	l.SetPrefix("[main]: ")

	var env Env
	// Read the config file, if present
	err := viper.ReadInConfig()
	if err != nil {
		log.Println("No config file found, reading from the system env")
		// TODO: fetch with viper, add validation
		env = Env{
			TelegramChannelID: os.Getenv("TELEGRAM_CHANNEL_ID"),
			TelegramBotToken:  os.Getenv("TELEGRAM_BOT_TOKEN"),
			OpenAiToken:       os.Getenv("OPENAI_TOKEN"),
			PostgresDSN:       os.Getenv("POSTGRES_DSN"),
		}
	} else {
		err = viper.Unmarshal(&env)
		if err != nil {
			l.Fatal("Error unmarshalling config:", err)
		}
	}

	pub, err := NewTelegramPublisher(env.TelegramChannelID, env.TelegramBotToken)
	if err != nil {
		l.Fatal("Error creating Telegram publisher:", err)
	}

	arch, err := NewArchivist(env.PostgresDSN)
	if err != nil {
		l.Fatal("Error creating Archivist:", err)
	}

	app := &App{
		staff: &Staff{
			marketJournalist: NewJournalist([]NewsProvider{
				NewRssProvider("bloomberg:markets", "https://feeds.bloomberg.com/markets/news.rss"),
				NewRssProvider("benzinga:news", "https://www.benzinga.com/news/feed"),
				NewRssProvider("benzinga:m&a", "https://www.benzinga.com/news/m-a/feed"),
				NewRssProvider("benzinga:buybacks", "https://www.benzinga.com/news/buybacks/feed"),
				NewRssProvider("wsj:markets", "https://feeds.a.dj.com/rss/RSSMarketsMain.xml"),
			}),
			teJournalist: NewJournalist([]NewsProvider{
				NewRssProvider("trading-economics:repo-rate", "https://tradingeconomics.com/rss/news.aspx?i=repo+rate"),
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
			}),
		},
		composer:  NewComposer(env.OpenAiToken),
		publisher: pub,
		archivist: arch,
	}

	s := gocron.NewScheduler(time.UTC)
	_, err = s.Every(60 * time.Second).Do(app.CreateMarketNewsJob(time.Now().Add(-60 * time.Second)))
	if err != nil {
		return
	}

	_, err = s.Every(5 * time.Minute).WaitForSchedule().Do(app.CreateTradingEconomicsNewsJob(time.Now().Add(-5 * time.Minute)))
	if err != nil {
		return
	}

	defer s.Stop()
	s.StartAsync()

	l.Println("Started fin-feed successfully")
	select {}
}
