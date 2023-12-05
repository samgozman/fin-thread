package composer

import (
	"context"
	"encoding/json"
	"errors"
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
		name                string
		args                args
		expectedFiltredNews journalist.NewsList
		want                []*ComposedNews
		wantErr             bool
	}{
		{
			name: "Should pass and return correct composed jsonNews",
			args: args{
				ctx:  context.Background(),
				news: news,
			},
			expectedFiltredNews: journalist.NewsList{news[0], news[1]},
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
			expectedFiltredNews: journalist.NewsList{news[0], news[1]},
			want:                nil,
			wantErr:             true,
		},
	}
	for _, tt := range tests {
		mockClient := new(MockOpenAiClient)
		defConf := DefaultConfig()

		// Set expectations for the mock client
		if tt.wantErr {
			mockError := errors.New("some error")
			mockClient.On("CreateChatCompletion", mock.Anything, mock.Anything).Return(openai.ChatCompletionResponse{}, mockError)
		} else {
			jsonNews, _ := tt.expectedFiltredNews.ToContentJSON()
			wantNewsJson, _ := json.Marshal(tt.want)
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
				Config:       DefaultConfig(),
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
