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
	Config       *Config
}

func NewComposer(oaiToken string) *Composer {
	return &Composer{OpenAiClient: openai.NewClient(oaiToken), Config: DefaultConfig()}
}

func (c *Composer) ChooseMostImportantNews(ctx context.Context, news journalist.NewsList) (journalist.NewsList, error) {
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
					Role:    openai.ChatMessageRoleSystem,
					Content: c.Config.ImportancePrompt,
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

	var filteredNews journalist.NewsList
	err = json.Unmarshal([]byte(resp.Choices[0].Message.Content), &filteredNews)
	if err != nil {
		return todayNews, err
	}

	return filteredNews, nil
}

func (c *Composer) ComposeNews(ctx context.Context, news journalist.NewsList) ([]*ComposedNews, error) {
	j, err := json.Marshal(news)
	if err != nil {
		return nil, err
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
						Role:    openai.ChatMessageRoleSystem,
						Content: c.Config.ComposePrompt,
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
			errorCh <- errors.New(fmt.Sprintf("[ComposeNews] error in json.Unmarshal: %s for object: %s", err, resp.Choices[0].Message.Content))
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

	return r, errors.Join(e...)
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
					Role:    openai.ChatMessageRoleSystem,
					Content: c.Config.MetaPrompt,
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
