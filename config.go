package main

import (
	"encoding/json"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/samgozman/fin-thread/journalist"
)

// Env is a structure that holds all the environment variables that are used in the app.
type Env struct {
	TelegramChannelID string `mapstructure:"TELEGRAM_CHANNEL_ID" validate:"required"`
	TelegramBotToken  string `mapstructure:"TELEGRAM_BOT_TOKEN" validate:"required"`
	OpenAiToken       string `mapstructure:"OPENAI_TOKEN" validate:"required"`
	TogetherAIToken   string `mapstructure:"TOGETHER_AI_TOKEN" validate:"required"`
	GoogleGeminiToken string `mapstructure:"GOOGLE_GEMINI_TOKEN"`
	PostgresDSN       string `mapstructure:"POSTGRES_DSN" validate:"required"`
	SentryDSN         string `mapstructure:"SENTRY_DSN" validate:"required"`
	StockSymbols      string `mapstructure:"STOCK_SYMBOLS" validate:"required"`
	MarketJournalists string `mapstructure:"MARKET_JOURNALISTS" validate:"required,json"`
	BroadJournalists  string `mapstructure:"BROAD_JOURNALISTS" validate:"required,json"`
}

type Config struct {
	env                *Env     // Holds all the environment variables that are used in the app
	suspiciousKeywords []string // Used to "flag" suspicious news by the journalist.Journalist
	rssProviders       struct {
		marketJournalists []journalist.NewsProvider // Market news journalists
		broadJournalists  []journalist.NewsProvider // Broad news journalists
	}
}

// NewConfig creates a new Config object with the given Env and default values from DefaultConfig.
func NewConfig(env *Env) (*Config, error) {
	c := DefaultConfig()
	c.env = env

	// unmarshal rss providers and validate them
	marketJournalists, err := unmarshalRssProviders(env.MarketJournalists)
	if err != nil {
		return nil, fmt.Errorf("marketJournalists: %w", err)
	}

	broadJournalists, err := unmarshalRssProviders(env.BroadJournalists)
	if err != nil {
		return nil, fmt.Errorf("broadJournalists: %w", err)
	}

	c.rssProviders.marketJournalists = marketJournalists
	c.rssProviders.broadJournalists = broadJournalists

	return c, nil
}

// DefaultConfig creates a new Config object with default values.
func DefaultConfig() *Config {
	return &Config{
		env: &Env{},
		suspiciousKeywords: []string{
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
			"class-action lawsuit",
			"subscribe",
			"ark invest",
			"cathie wood",
			"bitcoin",
			"ethereum",
			"btc",
			"eth",
			"dogecoin",
			"technical charts",
			"technical chart",
			"technical analysis",
			"technical indicator",
			"technical pattern",
			"golden cross",
			"death cross",
			"50-day",
			"200-day",
			"50 day",
			"200 day",
			"moving average",
			"rsi",
			"analysts say",
			"analyst says",
		},
	}
}

type rssProvider struct {
	Name string `validate:"required"`
	URL  string `validate:"required,url"`
}

// unmarshalRssProviders unmarshal a JSON string into a slice of rssProvider objects.
func unmarshalRssProviders(str string) ([]journalist.NewsProvider, error) {
	var rssProviderList []rssProvider
	err := json.Unmarshal([]byte(str), &rssProviderList)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling journalists: %w", err)
	}
	for _, item := range rssProviderList {
		err := validator.New().Struct(item)
		if err != nil {
			return nil, fmt.Errorf("error validating journalist: %w", err)
		}
	}

	result := make([]journalist.NewsProvider, 0, len(rssProviderList))
	for _, item := range rssProviderList {
		result = append(result, journalist.NewRssProvider(item.Name, item.URL))
	}

	return result, nil
}
