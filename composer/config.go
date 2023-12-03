package composer

type Config struct {
	ComposePrompt string
}

func DefaultConfig() *Config {
	return &Config{
		ComposePrompt: `You will be answering only in JSON array format: [{id:"", text:"", tickers:[], markets:[], hashtags:[]}]
		You need to remove from array blank, purposeless, clickbait, advertising or non-financial news.
		Most  important news right know is inflation, interest rates, war, elections, crisis, unemployment index etc.
		If none of the news are important, return empty array [].
		Next you need to fill some (or none) tickers, markets and hashtags arrays for each news.
		If news are mentioning some companies and stocks you need to find appropriate stocks 'tickers'. 
		If news are about some market events you need to fill 'markets' with some index tickers (like SPY, QQQ, or RUT etc.) based on the context.
		News context can be also related to some popular topics, we call it 'hashtags'.
		You only need to choose appropriate hashtag (0-3) only from this list: inflation, interestrates, crisis, unemployment, bankruptcy, dividends, IPO, debt, war, buybacks, fed.
		It is OK if you don't find find some tickers, markets or hashtags. It's also possible that you will find none.
		Next you need to combine the title and description into one sentence and rewrite it
		so it would be more straight to the point and look more original and easy to read.`,
	}
}
