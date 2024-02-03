package composer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/samgozman/fin-thread/internal/utils"
	"github.com/samgozman/fin-thread/pkg/errlvl"
	"time"

	"github.com/samber/lo"
	"github.com/samgozman/fin-thread/journalist"
	"github.com/sashabaranov/go-openai"
)

// TODO: refactor Composer to be able to choose provider for each method

// Composer is used to compose (rephrase) news and events, find some meta information about them,
// filter out some unnecessary stuff, summarise them and so on.
type Composer struct {
	OpenAiClient       openAiClientInterface
	TogetherAIClient   togetherAIClientInterface
	GoogleGeminiClient GoogleGeminiClientInterface
	Config             *promptConfig
}

// NewComposer creates a new Composer instance with OpenAI and TogetherAI clients and default config.
func NewComposer(oaiToken, tgrAiToken, geminiToken string) *Composer {
	return &Composer{
		OpenAiClient:       openai.NewClient(oaiToken),
		TogetherAIClient:   NewTogetherAI(tgrAiToken),
		GoogleGeminiClient: NewGoogleGemini(geminiToken),
		Config:             defaultPromptConfig(),
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
	preFilteredNews := todayNews.RemoveFlagged()
	jsonNews, err := preFilteredNews.ToContentJSON()
	if err != nil {
		return nil, newError(err, errlvl.ERROR, "Compose", "NewsList.ToContentJSON")
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
			Stop:             []string{"#"}, // Stop on hashtags in text
		},
	)
	if err != nil {
		return nil, newError(err, errlvl.WARN, "Compose", "OpenAiClient.CreateChatCompletion")
	}

	matches, err := aiJSONStringFixer(resp.Choices[0].Message.Content)
	if err != nil {
		return nil, newError(err, errlvl.ERROR, "Compose", "aiJSONStringFixer")
	}

	var fullComposedNews []*ComposedNews
	err = json.Unmarshal([]byte(matches), &fullComposedNews)
	if err != nil {
		return nil, newError(err, errlvl.ERROR, "Compose", "json.Unmarshal").WithValue(matches)
	}

	for _, n := range fullComposedNews {
		// Fix unicode symbols in tickers
		for i, t := range n.Tickers {
			n.Tickers[i] = utils.ReplaceUnicodeSymbols(t)
		}
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
		return nil, newError(err, errlvl.ERROR, "Summarise", "json.Marshal headlines").WithValue(fmt.Sprintf("%+v", headlines))
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
		return nil, newError(err, errlvl.WARN, "Summarise", "OpenAiClient.CreateChatCompletion")
	}

	matches, err := aiJSONStringFixer(resp.Choices[0].Message.Content)
	if err != nil {
		return nil, newError(err, errlvl.ERROR, "Summarise", "aiJSONStringFixer")
	}

	var h []*SummarisedHeadline
	err = json.Unmarshal([]byte(matches), &h)
	if err != nil {
		return nil, newError(err, errlvl.ERROR, "Summarise", "json.Unmarshal").WithValue(resp.Choices[0].Message.Content)
	}

	return h, nil
}

// Filter removes unnecessary news from the given news list using GoogleGemini API
// and returns the same news list with IsFiltered flag set to true for filtered out news.
func (c *Composer) Filter(ctx context.Context, news journalist.NewsList) (journalist.NewsList, error) {
	if len(news) == 0 {
		return nil, nil
	}

	preFilteredNews := news.RemoveFlagged()
	jsonNews, err := preFilteredNews.ToContentJSON()
	if err != nil {
		return nil, newError(err, errlvl.ERROR, "Filter", "ToContentJSON").WithValue(fmt.Sprintf("%+v", news))
	}

	resp, err := c.GoogleGeminiClient.CreateChatCompletion(
		ctx,
		GoogleGeminiRequest{
			Prompt:      c.Config.FilterPromptInstruct(jsonNews),
			MaxTokens:   2048,
			Temperature: 0.9,
			TopP:        1,
			TopK:        1,
		},
	)
	if err != nil {
		return nil, newError(err, errlvl.WARN, "Filter", "GoogleGeminiClient.CreateChatCompletion")
	}

	matches, err := aiJSONStringFixer(
		fmt.Sprintf("%s", resp.Candidates[0].Content.Parts[0]),
	)
	if err != nil {
		return nil, newError(err, errlvl.ERROR, "Filter", "aiJSONStringFixer")
	}

	var chosenByAi journalist.NewsList
	err = json.Unmarshal([]byte(matches), &chosenByAi)
	if err != nil {
		return nil, newError(err, errlvl.ERROR, "Filter", "json.Unmarshal").WithValue(matches)
	}

	// Create a map of chosenByAi news IDs to quickly find them
	chosenMap := make(map[string]*journalist.News)
	for _, n := range chosenByAi {
		chosenMap[n.ID] = n
	}

	preFilteredMap := make(map[string]*journalist.News)
	for _, n := range preFilteredNews {
		preFilteredMap[n.ID] = n
	}

	// Add IsFiltered flag to the original news list if it is NOT chosen by AI (filtered out)
	for _, n := range news {
		_, isChosen := chosenMap[n.ID]
		_, isPreFiltered := preFilteredMap[n.ID]

		// Mark news as filtered only if it wasn't removed by pre-filtering before
		if !isChosen && isPreFiltered {
			n.IsFiltered = true
		}
	}

	return news, nil
}

// Headline is the base data structure for the data to summarise.
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
