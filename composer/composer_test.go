package composer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"reflect"
	"testing"
	"time"

	"github.com/samgozman/fin-thread/journalist"
	"github.com/stretchr/testify/mock"
)

type MockOpenAiClient struct {
	mock.Mock
}

func (m *MockOpenAiClient) CreateChatCompletion(ctx context.Context, req openai.ChatCompletionRequest) (response openai.ChatCompletionResponse, error error) {
	args := m.Called(ctx, req)
	return args.Get(0).(openai.ChatCompletionResponse), args.Error(1)
}

func TestComposer_Compose(t *testing.T) {
	news := journalist.NewsList{
		{
			ID:           "1",
			Title:        "Ray Dalio says U.S. reaching an inflection point where the debt problem quickly gets even worse",
			Description:  "Soaring U.S. government debt is reaching a point where it will begin creating larger problems, the hedge fund titan said Friday.",
			Link:         "https://www.cnbc.com/",
			Date:         time.Now().UTC(),
			ProviderName: "cnbc",
		},
		{
			ID:           "2",
			Title:        "The market thinks the Fed is going to start cutting rates aggressively. Investors could be in for a letdown",
			Description:  "Markets may be at least a tad optimistic, particularly considering the cautious approach central bank officials have taken.",
			Link:         "https://www.cnbc.com/",
			Date:         time.Now().UTC(),
			ProviderName: "cnbc",
		},
		{
			ID:           "3",
			Title:        "Wholesale prices fell 0.5% in October for biggest monthly drop since April 2020",
			Description:  "Wholesale prices fell 0.5% in October for biggest monthly drop since April 2020",
			Link:         "https://www.cnbc.com/",
			Date:         time.Now().Add(-24 * time.Hour * 2).UTC(), // Should be filtered out
			ProviderName: "cnbc",
		},
	}

	type args struct {
		ctx  context.Context
		news journalist.NewsList
	}
	tests := []struct {
		name                 string
		args                 args
		expectedFilteredNews journalist.NewsList
		want                 []*ComposedNews
		wantErr              bool
	}{
		{
			name: "Should pass and return correct composed jsonNews",
			args: args{
				ctx:  context.Background(),
				news: news,
			},
			expectedFilteredNews: journalist.NewsList{news[0], news[1]},
			want: []*ComposedNews{
				{
					ID:       "1",
					Text:     "Ray Dalio warns about the soaring U.S. government debt reaching a critical inflection point, potentially leading to larger problems.",
					Tickers:  []string{},
					Markets:  []string{},
					Hashtags: []string{"debt"},
				},
				{
					ID:       "2",
					Text:     "The market anticipates aggressive rate cuts by the Fed, despite the cautious approach of central bank officials. Investors may face disappointment.",
					Tickers:  []string{},
					Markets:  []string{},
					Hashtags: []string{"interestrates"},
				},
			},
			wantErr: false,
		},
		{
			name: "Should pass and return empty array correctly",
			args: args{
				ctx:  context.Background(),
				news: journalist.NewsList{},
			},
			want:    []*ComposedNews{},
			wantErr: false,
		},
		{
			name: "Should return error if OpenAI fails",
			args: args{
				ctx:  context.Background(),
				news: news,
			},
			expectedFilteredNews: journalist.NewsList{news[0], news[1]},
			want:                 nil,
			wantErr:              true,
		},
	}
	for _, tt := range tests {
		mockClient := new(MockOpenAiClient)
		defConf := DefaultPromptConfig()

		// Set expectations for the mock client
		if tt.wantErr {
			mockError := errors.New("some error")
			mockClient.On("CreateChatCompletion", mock.Anything, mock.Anything).Return(openai.ChatCompletionResponse{}, mockError)
		} else {
			jsonNews, _ := tt.expectedFilteredNews.ToContentJSON()

			// Break the JSON to test the fix for OpenAI frequent bug (with extra closing bracket and some other stuff)
			wantNewsJson, _ := json.MarshalIndent(tt.want, "", "  ")
			wantNewsJson = []byte(fmt.Sprintf("```{%s}```", wantNewsJson))

			mockClient.On("CreateChatCompletion", mock.Anything, openai.ChatCompletionRequest{
				Model: openai.GPT3Dot5Turbo1106,
				Messages: []openai.ChatCompletionMessage{
					{
						Role:    openai.ChatMessageRoleSystem,
						Content: defConf.ComposePrompt,
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
			}).Return(openai.ChatCompletionResponse{
				Choices: []openai.ChatCompletionChoice{
					{
						Message: openai.ChatCompletionMessage{
							Content: string(wantNewsJson),
						},
					},
				},
			}, nil)
		}

		t.Run(tt.name, func(t *testing.T) {
			c := &Composer{
				OpenAiClient: mockClient,
				Config:       DefaultPromptConfig(),
			}
			got, err := c.Compose(tt.args.ctx, tt.args.news)
			if (err != nil) != tt.wantErr {
				t.Errorf("Compose() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("Compose() wrong len = %v, want %v", len(got), len(tt.want))
			}

			for i, n := range got {
				if !reflect.DeepEqual(n, tt.want[i]) {
					t.Errorf("Compose() = %v, want %v", n, tt.want[i])
				}
			}
		})
	}
}

func TestComposer_Summarise(t *testing.T) {
	type fields struct {
		OpenAiClient OpenAiClientInterface
		Config       *PromptConfig
	}
	type args struct {
		ctx            context.Context
		headlines      []Headline
		headlinesLimit int
		maxTokens      int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []SummarisedHeadline
		wantErr bool
	}{
		{
			name: "Should pass and return correct composed jsonNews",
			args: args{
				ctx: context.Background(),
				headlines: []Headline{
					{
						ID:   "1",
						Text: "Ray Dalio warns about the soaring U.S. government debt reaching a critical inflection point, potentially leading to larger problems.",
						Link: "https://t.me/fin_thread/1",
					},
					{
						ID:   "2",
						Text: "The market anticipates aggressive rate cuts by the Fed, despite the cautious approach of central bank officials. Investors may face disappointment.",
						Link: "https://t.me/fin_thread/2",
					},
				},
				headlinesLimit: 2,
				maxTokens:      512,
			},
			want: []SummarisedHeadline{
				{
					ID:      "1",
					Summary: "Some warns summary",
					Link:    "https://t.me/fin_thread/1",
					Verb:    "warns",
				},
				{
					ID:      "2",
					Summary: "Some anticipates summary",
					Link:    "https://t.me/fin_thread/2",
					Verb:    "anticipates",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		mockClient := new(MockOpenAiClient)
		defConf := DefaultPromptConfig()

		c := &Composer{
			OpenAiClient: mockClient,
			Config:       DefaultPromptConfig(),
		}

		// Set expectations for the mock client
		if tt.wantErr {
			mockError := errors.New("some error")
			mockClient.On("CreateChatCompletion", mock.Anything, mock.Anything).Return(openai.ChatCompletionResponse{}, mockError)
		} else {
			jsonHeadlines, err := json.Marshal(tt.args.headlines)
			if err != nil {
				return
			}

			// Break the JSON to test the fix for OpenAI frequent bug (with extra closing bracket and some other stuff)
			wantHeadlines, _ := json.MarshalIndent(tt.want, "", "  ")
			wantHeadlines = []byte(fmt.Sprintf("```{%s}```", wantHeadlines))

			mockClient.On("CreateChatCompletion", mock.Anything, openai.ChatCompletionRequest{
				Model: openai.GPT3Dot5Turbo1106,
				Messages: []openai.ChatCompletionMessage{
					{
						Role:    openai.ChatMessageRoleSystem,
						Content: defConf.SummarisePrompt(tt.args.headlinesLimit),
					},
					{
						Role:    openai.ChatMessageRoleUser,
						Content: string(jsonHeadlines),
					},
				},
				Temperature:      1,
				MaxTokens:        tt.args.maxTokens,
				TopP:             0.7,
				FrequencyPenalty: 0,
				PresencePenalty:  0,
			}).Return(openai.ChatCompletionResponse{
				Choices: []openai.ChatCompletionChoice{
					{
						Message: openai.ChatCompletionMessage{
							Content: string(wantHeadlines),
						},
					},
				},
			}, nil)
		}

		t.Run(tt.name, func(t *testing.T) {
			got, err := c.Summarise(tt.args.ctx, tt.args.headlines, tt.args.headlinesLimit, tt.args.maxTokens)
			if (err != nil) != tt.wantErr {
				t.Errorf("Summarise() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Summarise() got = %v, want %v", got, tt.want)
			}
		})
	}
}
