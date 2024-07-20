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

func (m *MockOpenAiClient) CreateChatCompletion(ctx context.Context, req openai.ChatCompletionRequest) (response openai.ChatCompletionResponse, err error) {
	args := m.Called(ctx, req)
	return args.Get(0).(openai.ChatCompletionResponse), args.Error(1) //nolint:wrapcheck
}

type MockTogetherAIClient struct {
	mock.Mock
}

func (m *MockTogetherAIClient) CreateChatCompletion(ctx context.Context, options togetherAIRequest) (*TogetherAIResponse, error) {
	args := m.Called(ctx, options)
	return args.Get(0).(*TogetherAIResponse), args.Error(1) //nolint:wrapcheck
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
			IsSuspicious: true,
			IsFiltered:   false,
		},
		{
			ID:           "2",
			Title:        "The market thinks the Fed is going to start cutting rates aggressively. Investors could be in for a letdown",
			Description:  "Markets may be at least a tad optimistic, particularly considering the cautious approach central bank officials have taken.",
			Link:         "https://www.cnbc.com/",
			Date:         time.Now().UTC(),
			ProviderName: "cnbc",
			IsSuspicious: false,
			IsFiltered:   false,
		},
		{
			ID:           "3",
			Title:        "Wholesale prices fell 0.5% in October for biggest monthly drop since April 2020",
			Description:  "Wholesale prices fell 0.5% in October for biggest monthly drop since April 2020",
			Link:         "https://www.cnbc.com/",
			Date:         time.Now().UTC(),
			ProviderName: "cnbc",
			IsSuspicious: false,
			IsFiltered:   false,
		},
	}

	type args struct {
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
				news: news,
			},
			expectedFilteredNews: journalist.NewsList{news[0], news[1], news[2]},
			want: []*ComposedNews{
				{
					ID:       "1",
					Text:     "Ray Dalio warns about the soaring U.S. government debt reaching a critical inflection point, potentially leading to larger problems.",
					Tickers:  []string{"AAPL"},
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
				{
					ID:       "3",
					Text:     "Wholesale prices fell 0.5% in October for biggest monthly drop since April 2020",
					Tickers:  []string{},
					Markets:  []string{},
					Hashtags: []string{},
				},
			},
			wantErr: false,
		},
		{
			name: "Should pass and return empty array correctly",
			args: args{
				news: journalist.NewsList{},
			},
			want:    []*ComposedNews{},
			wantErr: false,
		},
		{
			name: "Should return error if OpenAI client returns error",
			args: args{
				news: news,
			},
			expectedFilteredNews: journalist.NewsList{news[0], news[1]},
			want:                 nil,
			wantErr:              true,
		},
	}
	for _, tt := range tests {
		mockClient := new(MockOpenAiClient)
		defConf := defaultPromptConfig()

		// Set expectations for the mock client
		if tt.wantErr {
			mockError := errors.New("some error")
			mockClient.On("CreateChatCompletion", mock.Anything, mock.Anything).Return(openai.ChatCompletionResponse{}, mockError)
		} else {
			jsonNews, _ := tt.expectedFilteredNews.RemoveFlagged().ToContentJSON()

			// Break the JSON to test the fix for OpenAI frequent bug (with extra closing bracket and some other stuff)
			wantNewsJSON, _ := json.MarshalIndent(tt.want, "", "  ")
			wantNewsJSON = []byte(fmt.Sprintf("```{%s}```", wantNewsJSON))

			mockClient.On("CreateChatCompletion", mock.Anything, openai.ChatCompletionRequest{
				Model: openai.GPT4oMini,
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
				Stop:             []string{"#"},
			}).Return(openai.ChatCompletionResponse{
				Choices: []openai.ChatCompletionChoice{
					{
						Message: openai.ChatCompletionMessage{
							Content: string(wantNewsJSON),
						},
					},
				},
			}, nil)
		}

		t.Run(tt.name, func(t *testing.T) {
			c := &Composer{
				OpenAiClient: mockClient,
				Config:       defaultPromptConfig(),
			}
			got, err := c.Compose(context.Background(), tt.args.news)
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
		OpenAiClient openAiClientInterface
		Config       *promptConfig
	}
	type args struct {
		headlines      []*Headline
		headlinesLimit int
		maxTokens      int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*SummarisedHeadline
		wantErr bool
	}{
		{
			name: "Should pass and return correct composed jsonNews",
			args: args{
				headlines: []*Headline{
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
			want: []*SummarisedHeadline{
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
		defConf := defaultPromptConfig()

		c := &Composer{
			OpenAiClient: mockClient,
			Config:       defaultPromptConfig(),
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
				Model: openai.GPT4oMini,
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
			got, err := c.Summarise(context.Background(), tt.args.headlines, tt.args.headlinesLimit, tt.args.maxTokens)
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

func TestComposer_Filter(t *testing.T) {
	type args struct {
		news journalist.NewsList
	}
	tests := []struct {
		name    string
		args    args
		want    journalist.NewsList
		wantErr bool
	}{
		{
			name: "Should pass and return correct filtered news",
			args: args{
				news: journalist.NewsList{
					{
						ID:           "1",
						Title:        "Ray Dalio says U.S. reaching an inflection point where the debt problem quickly gets even worse",
						Description:  "Soaring U.S. government debt is reaching a point where it will begin creating larger problems, the hedge fund titan said Friday.",
						Link:         "https://www.cnbc.com/",
						Date:         time.Now().UTC(),
						ProviderName: "cnbc",
						IsFiltered:   false,
						IsSuspicious: true,
					},
					{
						ID:           "2",
						Title:        "The market thinks the Fed is going to start cutting rates aggressively. Investors could be in for a letdown",
						Description:  "Markets may be at least a tad optimistic, particularly considering the cautious approach central bank officials have taken.",
						Link:         "https://www.cnbc.com/",
						Date:         time.Now().UTC(),
						ProviderName: "cnbc",
						IsFiltered:   false,
						IsSuspicious: false,
					},
					{
						ID:           "3",
						Title:        "Wholesale prices fell 0.5% in October for biggest monthly drop since April 2020",
						Description:  "Wholesale prices fell 0.5% in October for biggest monthly drop since April 2020",
						Link:         "https://www.cnbc.com/",
						Date:         time.Now().UTC(),
						ProviderName: "cnbc",
						IsFiltered:   false,
						IsSuspicious: false,
					},
				},
			},
			want: journalist.NewsList{
				{
					ID:           "1",
					Title:        "Ray Dalio says U.S. reaching an inflection point where the debt problem quickly gets even worse",
					Description:  "Soaring U.S. government debt is reaching a point where it will begin creating larger problems, the hedge fund titan said Friday.",
					Link:         "https://www.cnbc.com/",
					Date:         time.Now().UTC(),
					ProviderName: "cnbc",
					IsFiltered:   false,
					IsSuspicious: true,
				},
				{
					ID:           "2",
					Title:        "The market thinks the Fed is going to start cutting rates aggressively. Investors could be in for a letdown",
					Description:  "Markets may be at least a tad optimistic, particularly considering the cautious approach central bank officials have taken.",
					Link:         "https://www.cnbc.com/",
					Date:         time.Now().UTC(),
					ProviderName: "cnbc",
					IsFiltered:   true,
					IsSuspicious: false,
				},
				{
					ID:           "3",
					Title:        "Wholesale prices fell 0.5% in October for biggest monthly drop since April 2020",
					Description:  "Wholesale prices fell 0.5% in October for biggest monthly drop since April 2020",
					Link:         "https://www.cnbc.com/",
					Date:         time.Now().UTC(),
					ProviderName: "cnbc",
					IsFiltered:   false,
					IsSuspicious: false,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := new(MockTogetherAIClient)
			defConf := defaultPromptConfig()

			// Set expectations for the mock client
			if tt.wantErr {
				mockError := errors.New("some error")
				mockClient.On("CreateChatCompletion", mock.Anything, mock.Anything).Return(&TogetherAIResponse{}, mockError)
			} else {
				jsonNews, _ := tt.args.news.RemoveFlagged().ToContentJSON()
				expectedJSONNews, _ := tt.want.RemoveFlagged().ToContentJSON()

				mockClient.On("CreateChatCompletion",
					mock.Anything,
					togetherAIRequest{
						Model:             "mistralai/Mixtral-8x7B-Instruct-v0.1",
						Prompt:            defConf.FilterPromptInstruct(jsonNews),
						MaxTokens:         2048,
						Temperature:       0.7,
						TopP:              0.7,
						TopK:              50,
						RepetitionPenalty: 1,
						Stop:              []string{"[/INST]", "</s>"},
					},
				).Return(&TogetherAIResponse{
					Choices: []struct {
						Text string `json:"text"`
					}{
						{
							Text: expectedJSONNews,
						},
					},
				}, nil)
			}

			c := &Composer{
				TogetherAIClient: mockClient,
				Config:           defaultPromptConfig(),
			}
			got, err := c.Filter(context.Background(), tt.args.news)
			if (err != nil) != tt.wantErr {
				t.Errorf("Filter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("Filter() wrong len = %v, want %v", len(got), len(tt.want))
			}
		})
	}
}
