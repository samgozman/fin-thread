package main

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
}

type Config struct {
	env                *Env     // Holds all the environment variables that are used in the app
	suspiciousKeywords []string // Used to "flag" suspicious news by the journalist.Journalist
	filterKeys         []string // Used to remove news by the journalist.Journalist if they don't contain any of these keys
}

// NewConfig creates a new Config object with the given Env and default values from DefaultConfig.
func NewConfig(env *Env) *Config {
	c := DefaultConfig()
	c.env = env
	return c
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
		},
		filterKeys: []string{
			"European Union",
			"United States",
			"United Kingdom",
			"China",
			"Germany",
			"France",
			"Japan",
			"Italy",
			"India",
		},
	}
}
