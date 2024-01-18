package composer

import "fmt"

type promptConfig struct {
	ComposePrompt        string
	SummarisePrompt      summarisePromptFunc
	FilterPromptInstruct filterPromptFunc
}

const (
	maxWordsPerSentence = 10
)

func defaultPromptConfig() *promptConfig {
	return &promptConfig{
		ComposePrompt: `You need to fill some (or none) tickers, markets and hashtags arrays for each news.
		If news are mentioning some companies and stocks you need to find appropriate stocks 'tickers' (ONLY STOCKS, ignore ETFs and crypto). 
		If news are about some market events you need to fill 'markets' with some index tickers (like SPY, QQQ, or RUT etc.) based on the context.
		News context can be also related to some popular topics, we call it 'hashtags'.
		You only need to choose appropriate hashtag (0-3) only from this list: inflation, interestrates, crisis, unemployment, bankruptcy, dividends, IPO, debt, war, buybacks, fed, AI, crypto, bitcoin.
		It is OK if you don't find some tickers, markets or hashtags. It's also possible that you will find none.
		Next you need to create an informative, original 'text' based on the title and description.
		You need to write a 'text' that would be easy to read and understand, 1-2 sentences long.
		Always answer in the following JSON format: [{id:"", text:"", tickers:[], markets:[], hashtags:[]}]
		----------------------------------------
		ONLY JSON IS ALLOWED as an answer. No explanation or other text is allowed.
`,
		SummarisePrompt: func(headlinesLimit int) string {
			return fmt.Sprintf(`You will receive a JSON array of news with IDs.
				You need to create a short (%v words max) summary for the %v most important financial, 
				economical, stock market news what happened from the start of the day.
				Find the main verb in the string and put it into the result JSON.
				Always answer in the following JSON format: [{summary:"", verb:"", id:"", link:""}]
				----------------------------------------
				ONLY JSON IS ALLOWED as an answer. No explanation or other text is allowed.
`,
				maxWordsPerSentence,
				headlinesLimit,
			)
		},
		FilterPromptInstruct: func(newsJson string) string {
			return fmt.Sprintf(`[INST]You will be given a JSON array of financial news.
				You need to remove from array blank, purposeless, clickbait, advertising or non-financial news.
				Most important news right know is inflation, interest rates, war, elections, crisis, unemployment index etc.
				Always answer in the following JSON format: [{\"ID\":\"\",\"Title\":\"\",\"Description\":\"\"}] or [].
				----------------------------------------
				ONLY JSON IS ALLOWED as an answer. No explanation or other text is allowed.
				Input:\n%s[/INST]`, newsJson)
		},
	}
}

type summarisePromptFunc = func(headlinesLimit int) string

type filterPromptFunc = func(newsJson string) string
