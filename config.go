package main

// Env is a structure that holds all the environment variables that are used in the app
type Env struct {
	TelegramChannelID string `mapstructure:"TELEGRAM_CHANNEL_ID"`
	TelegramBotToken  string `mapstructure:"TELEGRAM_BOT_TOKEN"`
	OpenAiToken       string `mapstructure:"OPENAI_TOKEN"`
	PostgresDSN       string `mapstructure:"POSTGRES_DSN"`
	SentryDSN         string `mapstructure:"SENTRY_DSN"`
}
