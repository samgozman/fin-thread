package composer

type Config struct {
	MetaPrompt       string
	ComposePrompt    string
	ImportancePrompt string
}

func DefaultConfig() *Config {
	return &Config{
		MetaPrompt: `You will be given a JSON array of financial news with ID. 
			Your job is to find meta data in those messages and response with string JSON array of format:
			[{id:"", tickers:[], markets:[], hashtags:[]}]
			If news are mentioning some companies and stocks you need to find appropriate stocks 'tickers'. 
			If news are about some market events you need to fill 'markets' with some index tickers (like SPY, QQQ, or RUT etc.) based on the context.
			News context can be also related to some popular topics, we call it 'hashtags'.
			You only need to choose appropriate hashtag (0-3) from this list: inflation, interestrates, crisis, unemployment, bankruptcy, dividends, IPO, debt, war, buybacks, fed.
			It is OK if you don't find find some tickers, markets or hashtags. It's also possible that you will find none.`,
		ComposePrompt: `You will be given a JSON array of financial news with ID. 
			Your job is to work with news feeds from users (financial, investments, market topics).
			Each news has a title and description. You need to combine the title and description
			and rewrite it so it would be more straight to the point and look more original.
			Response with string JSON array of format:
			[{news_id:"", text:""}]`,
		ImportancePrompt: `You will be given a JSON array of financial news.
			You need to remove from array blank, purposeless, clickbait, advertising or non-financial news.
			Most  important news right know is inflation, interest rates, war, elections, crisis, unemployment index etc.
			Return the response in the same JSON format. If none of the news are important, return empty array [].`,
	}
}
