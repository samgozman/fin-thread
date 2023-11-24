package composer

import (
	"context"
	"encoding/json"
	"time"

	"github.com/samber/lo"
	"github.com/samgozman/go-fin-feed/journalist"
	"github.com/sashabaranov/go-openai"
)

type OpenAiClientInterface interface {
	CreateChatCompletion(ctx context.Context, req openai.ChatCompletionRequest) (response openai.ChatCompletionResponse, error error)
}

type Composer struct {
	OpenAiClient OpenAiClientInterface
}

func NewComposer(openAiClient OpenAiClientInterface) *Composer {
	return &Composer{OpenAiClient: openAiClient}
}

func (c *Composer) ChooseMostImportantNews(ctx context.Context, news []*journalist.News) ([]*journalist.News, error) {
	// Filter out news that are not from today
	todayNews := lo.Filter(news, func(n *journalist.News, _ int) bool {
		return n.Date.Day() == time.Now().Day()
	})

	// Convert news to JSON
	jsonNews, err := json.Marshal(todayNews)
	if err != nil {
		return todayNews, err
	}

	resp, err := c.OpenAiClient.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo1106,
			Messages: []openai.ChatCompletionMessage{
				{
					Role: openai.ChatMessageRoleSystem,
					Content: "You will be given a JSON array of financial news. " +
						"You need to remove from array blank, purposeless, clickbait, advertising or non-financial news. " +
						"Most  important news right know is inflation, interest rates, war, elections, crisis, unemployment index etc. " +
						"Return the response in the same JSON format. If none of the news are important, return empty array [].",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: string(jsonNews),
				},
			},
			Temperature:      1,
			MaxTokens:        2048,
			TopP:             1,
			FrequencyPenalty: 0,
			PresencePenalty:  0,
		},
	)
	if err != nil {
		return todayNews, err
	}

	var filteredNews []*journalist.News
	err = json.Unmarshal([]byte(resp.Choices[0].Message.Content), &filteredNews)
	if err != nil {
		return todayNews, err
	}

	return filteredNews, nil
}

func (c *Composer) ComposeNews(ctx context.Context, news []*journalist.News) []*ComposedNews {
	// TODO: implement (use OpenAiClient)
	// Call findNewsMetaData
	return nil
}

// findNewsMetaData finds tickers, markets and hashtags mentioned in the news.
//
// Returns map of NewsID -> NewsMeta
func (c *Composer) findNewsMetaData(ctx context.Context, news []*journalist.News) (map[string]*NewsMeta, error) {
	jsonNews, err := json.Marshal(news)
	if err != nil {
		return nil, err
	}

	resp, err := c.OpenAiClient.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo1106,
			Messages: []openai.ChatCompletionMessage{
				{
					Role: openai.ChatMessageRoleSystem,
					Content: `You will be given a JSON array of financial news with ID. 
					Your job is to find meta data in those messages and response with string JSON array of format:
					[{id:"", tickers:[], markets:[], hashtags:[]}]
					If news are mentioning some companies and stocks you need to find appropriate stocks 'tickers'. 
					If news are about some market events you need to fill 'markets' with some index tickers (like SPY, QQQ, or RUT etc.) based on the context.
					News context can be also related to some popular topics, we call it 'hashtags'.
					You only need to choose appropriate hashtag (0-3) from this list: inflation, interestrates, crisis, unemployment, bankruptcy, dividends, IPO, debt, war, buybacks, fed.
					It is OK if you don't find find some tickers, markets or hashtags. It's also possible that you will find none.`,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: string(jsonNews),
				},
			},
			Temperature:      1,
			MaxTokens:        2048,
			TopP:             1,
			FrequencyPenalty: 0,
			PresencePenalty:  0,
		},
	)
	if err != nil {
		return nil, err
	}

	var newsMeta []*newsMetaParsed
	err = json.Unmarshal([]byte(resp.Choices[0].Message.Content), &newsMeta)
	if err != nil {
		return nil, err
	}

	// Convert newsMeta to map
	newsMetaMap := make(map[string]*NewsMeta)
	for _, meta := range newsMeta {
		newsMetaMap[meta.ID] = &NewsMeta{
			Tickers:  meta.Tickers,
			Markets:  meta.Markets,
			Hashtags: meta.Hashtags,
		}
	}

	return newsMetaMap, nil
}

type NewsMeta struct {
	Tickers  []string // tickers mentioned or/and related to the news
	Markets  []string // US/EU/Asia stocks, bonds, commodities, housing, etc.
	Hashtags []string // hashtags related to the news (#inflation, #fed, #buybacks, etc.)
}

type ComposedNews struct {
	NewsID   string
	Text     string
	MetaData *NewsMeta
}

type newsMetaParsed struct {
	ID       string   `json:"id"`
	Tickers  []string `json:"tickers"`
	Markets  []string `json:"markets"`
	Hashtags []string `json:"hashtags"`
}
