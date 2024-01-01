package composer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/samber/lo"
	"github.com/samgozman/fin-thread/journalist"
	"github.com/sashabaranov/go-openai"
)

// Composer is used to compose (rephrase) news and events, find some meta information about them,
// filter out some unnecessary stuff, summarise them and so on.
type Composer struct {
	OpenAiClient     OpenAiClientInterface
	TogetherAIClient TogetherAIClientInterface
	Config           *PromptConfig
}

// NewComposer creates a new Composer instance with OpenAI and TogetherAI clients and default config
func NewComposer(oaiToken, tgrAiToken string) *Composer {
	return &Composer{
		OpenAiClient:     openai.NewClient(oaiToken),
		TogetherAIClient: NewTogetherAI(tgrAiToken),
		Config:           DefaultPromptConfig(),
	}
}

// Compose creates a new AI-composed news from the given news list.
// It will also find some meta information about the news and events (markets, tickers, hashtags).
func (c *Composer) Compose(ctx context.Context, news journalist.NewsList) ([]*ComposedNews, error) {
	// RemoveDuplicates out news that are not from today
	var todayNews journalist.NewsList = lo.Filter(news, func(n *journalist.News, _ int) bool {
		return n.Date.Day() == time.Now().Day()
	})

	if len(todayNews) == 0 {
		return nil, nil
	}

	// Convert news to JSON
	jsonNews, err := todayNews.ToContentJSON()
	if err != nil {
		return nil, newErr(err, "Compose", "NewsList.ToContentJSON")
	}

	// Compose news
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
		return nil, newErr(err, "Compose", "OpenAiClient.CreateChatCompletion")
	}

	matches, err := openaiJSONStringFixer(resp.Choices[0].Message.Content)
	if err != nil {
		return nil, newErr(err, "Compose", "openaiJSONStringFixer")
	}

	var fullComposedNews []*ComposedNews
	err = json.Unmarshal([]byte(matches), &fullComposedNews)
	if err != nil {
		return nil, newErr(err, "Compose", "json.Unmarshal").WithValue(matches)
	}

	return fullComposedNews, nil
}

// Summarise create a short AI summary for the Headline array of any kind.
// It will also add Markdown links in summary.
//
// `headlinesLimit` is used to tell AI to use only top N Headlines from the batch for summary (AI will decide).
//
// `maxTokens` is used to limit summary size in tokens. It is the hard limit for AI and also used
// for dynamically decide how many sentences AI should produce.
func (c *Composer) Summarise(ctx context.Context, headlines []*Headline, headlinesLimit, maxTokens int) ([]*SummarisedHeadline, error) {
	if len(headlines) == 0 {
		return nil, nil
	}

	if maxTokens == 0 {
		return nil, errors.New("maxTokens can't be 0")
	}

	if headlinesLimit == 0 {
		return nil, errors.New("headlinesLimit can't be 0")
	}

	jsonHeadlines, err := json.Marshal(headlines)
	if err != nil {
		return nil, newErr(err, "Summarise", "json.Marshal headlines").WithValue(fmt.Sprintf("%+v", headlines))
	}

	resp, err := c.OpenAiClient.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo1106,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: c.Config.SummarisePrompt(headlinesLimit),
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: string(jsonHeadlines),
				},
			},
			Temperature:      1,
			MaxTokens:        maxTokens,
			TopP:             0.7,
			FrequencyPenalty: 0,
			PresencePenalty:  0,
		},
	)
	if err != nil {
		return nil, newErr(err, "Summarise", "OpenAiClient.CreateChatCompletion")
	}

	matches, err := openaiJSONStringFixer(resp.Choices[0].Message.Content)
	if err != nil {
		return nil, newErr(err, "Summarise", "openaiJSONStringFixer")
	}

	var h []*SummarisedHeadline
	err = json.Unmarshal([]byte(matches), &h)
	if err != nil {
		return nil, newErr(err, "Summarise", "json.Unmarshal").WithValue(resp.Choices[0].Message.Content)
	}

	return h, nil
}

// Filter removes unnecessary news from the given news list using TogetherAI API.
func (c *Composer) Filter(ctx context.Context, news journalist.NewsList) (journalist.NewsList, error) {
	if len(news) == 0 {
		return nil, nil
	}

	jsonNews, err := news.ToJSON()
	if err != nil {
		return nil, newErr(err, "Filter", "json.Marshal news").WithValue(fmt.Sprintf("%+v", news))
	}

	resp, err := c.TogetherAIClient.CreateChatCompletion(
		ctx,
		TogetherAIRequest{
			Model:             "mistralai/Mistral-7B-Instruct-v0.2",
			Prompt:            c.Config.FilterPromptInstruct(jsonNews),
			MaxTokens:         2048,
			Temperature:       0.7,
			TopP:              0.7,
			TopK:              50,
			RepetitionPenalty: 1,
		},
	)
	if err != nil {
		return nil, newErr(err, "Filter", "TogetherAIClient.CreateChatCompletion")
	}

	matches, err := openaiJSONStringFixer(resp.Choices[0].Text)
	if err != nil {
		return nil, newErr(err, "Filter", "openaiJSONStringFixer")
	}

	var filtered journalist.NewsList
	err = json.Unmarshal([]byte(matches), &filtered)
	if err != nil {
		return nil, newErr(err, "Filter", "json.Unmarshal").WithValue(resp.Choices[0].Text)
	}

	return filtered, nil
}

// Headline is the base data structure for the data to summarise
type Headline struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	Link string `json:"link"`
}

// SummarisedHeadline is the base data structure of summarised news or events.
//
// OpenAI fails to apply markdown on selected verbs, but it's good at finding them.
type SummarisedHeadline struct {
	ID      string `json:"id"`      // ID of the news or event
	Verb    string `json:"verb"`    // Main verb of the news or event to be marked in summary
	Summary string `json:"summary"` // Summary of the news or event
	Link    string `json:"link"`    // Link to the publication to use in string Markdown
}

type ComposedNews struct {
	ID       string   `json:"id"`
	Text     string   `json:"text"`
	Tickers  []string `json:"tickers"`  // tickers mentioned or/and related to the news
	Markets  []string `json:"markets"`  // US/EU/Asia stocks, bonds, commodities, housing, etc.
	Hashtags []string `json:"hashtags"` // hashtags related to the news (#inflation, #fed, #buybacks, etc.)
}

type ComposedMeta struct {
	Tickers  []string `json:"tickers"`
	Markets  []string `json:"markets"`
	Hashtags []string `json:"hashtags"`
}
