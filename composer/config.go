package composer

type Config struct {
	ComposePrompt string
}

func DefaultConfig() *Config {
	return &Config{
		ComposePrompt: `You will be answering only in JSON array format: [{id:"", text:"", tickers:[], markets:[], hashtags:[]}]
		You need to remove from array blank, spam, purposeless, clickbait, tabloid, advertising, unspecified, anonymous or non-financial news.
		Most important news right know is inflation, interest rates, war, elections, crisis, unemployment index, regulations.
		If none of the news are important, return empty array [].
		Next you need to fill some (or none) tickers, markets and hashtags arrays for each news.
		If news are mentioning some companies and stocks you need to find appropriate stocks 'tickers'. 
		If news are about some market events you need to fill 'markets' with some index tickers (like SPY, QQQ, or RUT etc.) based on the context.
		News context can be also related to some popular topics, we call it 'hashtags'.
		You only need to choose appropriate hashtag (0-3) only from this list: inflation, interestrates, crisis, unemployment, bankruptcy, dividends, IPO, debt, war, buybacks, fed, AI, crypto, bitcoin.
		It is OK if you don't find find some tickers, markets or hashtags. It's also possible that you will find none.
		Next you need to create an informative, original 'text' based on the title and description.
		You need to write a 'text' that would be easy to read and understand, 1-2 sentences long.
`,
	}
}
