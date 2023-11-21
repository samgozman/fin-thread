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
func (c *Composer) findNewsMetaData(ctx context.Context, news []*journalist.News) map[string]*NewsMeta {
	// TODO: implement (use OpenAiClient)
	return nil
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
