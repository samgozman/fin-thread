package composer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
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

func (c *Composer) ComposeNews(ctx context.Context, news []*journalist.News) ([]*ComposedNews, []error) {
	j, err := json.Marshal(news)
	if err != nil {
		return nil, []error{err}
	}
	jsonNews := string(j)

	composedCh := make(chan []*ComposedNews, 1)
	metaCh := make(chan map[string]*NewsMeta, 1)
	errorCh := make(chan error, 2)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		meta, err := c.findNewsMetaData(ctx, jsonNews)
		if err != nil {
			errorCh <- errors.New(fmt.Sprintf("[ComposeNews] error in findNewsMetaData: %s", err))
			return
		}

		metaCh <- meta
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		resp, err := c.OpenAiClient.CreateChatCompletion(
			ctx,
			openai.ChatCompletionRequest{
				Model: openai.GPT3Dot5Turbo1106,
				Messages: []openai.ChatCompletionMessage{
					{
						Role: openai.ChatMessageRoleSystem,
						Content: `You will be given a JSON array of financial news with ID. 
						Your job is to work with news feeds from users (financial, investments, market topics).
						Each news has a title and description. You need to combine the title and description
						and rewrite it so it would be more straight to the point and look more original.
						Response with string JSON array of format:
						[{news_id:"", text:""}]`,
					},
					{
						Role:    openai.ChatMessageRoleUser,
						Content: jsonNews,
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
			errorCh <- errors.New(fmt.Sprintf("[ComposeNews] error in OpenAiClient.CreateChatCompletion: %s", err))
			return
		}

		var composedNews []*ComposedNews
		err = json.Unmarshal([]byte(resp.Choices[0].Message.Content), &composedNews)
		if err != nil {
			errorCh <- errors.New(fmt.Sprintf("[ComposeNews] error in json.Unmarshal: %s", err))
			return
		}

		composedCh <- composedNews
	}()

	wg.Wait()
	close(composedCh)
	close(metaCh)
	close(errorCh)

	var r []*ComposedNews
	var e []error
	var m map[string]*NewsMeta

	for result := range composedCh {
		r = append(r, result...)
	}

	for err := range errorCh {
		e = append(e, err)
	}

	for meta := range metaCh {
		m = meta
	}

	for _, n := range r {
		n.MetaData = m[n.NewsID]
	}

	return r, e
}

// findNewsMetaData finds tickers, markets and hashtags mentioned in the news.
//
// Returns map of NewsID -> NewsMeta
func (c *Composer) findNewsMetaData(ctx context.Context, jsonNews string) (map[string]*NewsMeta, error) {
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
					Content: jsonNews,
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
	NewsID   string    `json:"news_id"`
	Text     string    `json:"text"`
	MetaData *NewsMeta `json:"meta_data"`
}

type newsMetaParsed struct {
	ID       string   `json:"id"`
	Tickers  []string `json:"tickers"`
	Markets  []string `json:"markets"`
	Hashtags []string `json:"hashtags"`
}
