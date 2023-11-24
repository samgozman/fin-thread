package composer

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/sashabaranov/go-openai"
	"reflect"
	"testing"
	"time"

	"github.com/samgozman/go-fin-feed/journalist"
	"github.com/stretchr/testify/mock"
)

type MockOpenAiClient struct {
	mock.Mock
}

func (m *MockOpenAiClient) CreateChatCompletion(ctx context.Context, req openai.ChatCompletionRequest) (response openai.ChatCompletionResponse, error error) {
	args := m.Called(ctx, req)
	return args.Get(0).(openai.ChatCompletionResponse), args.Error(1)
}

func TestComposer_ChooseMostImportantNews(t *testing.T) {
	type args struct {
		ctx  context.Context
		news []*journalist.News
	}

	news := []*journalist.News{
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
			Date:         time.Now().Add(-24 * time.Hour * 2).UTC(),
			ProviderName: "cnbc",
		},
	}

	tests := []struct {
		name    string
		args    args
		want    []*journalist.News
		wantErr bool
	}{
		{
			name: "Should pass and return 2 news",
			args: args{
				ctx:  context.Background(),
				news: []*journalist.News{news[0], news[1], news[2]},
			},
			want:    []*journalist.News{news[0], news[1]},
			wantErr: false,
		},
		{
			name: "Should pass and return 0 news",
			args: args{
				ctx:  context.Background(),
				news: []*journalist.News{news[2]},
			},
			want:    []*journalist.News{},
			wantErr: false,
		},
		{
			name: "Should return original news (except overdue) and error if OpenAI returns fails",
			args: args{
				ctx:  context.Background(),
				news: []*journalist.News{news[0], news[1], news[2]},
			},
			want:    []*journalist.News{news[0], news[1]},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		mockClient := new(MockOpenAiClient)

		// Set expectations for the mock client
		if tt.wantErr {
			mockError := errors.New("some error")
			mockClient.On("CreateChatCompletion", mock.Anything, mock.Anything).Return(openai.ChatCompletionResponse{}, mockError)
		} else {
			wantNewsJson, _ := json.Marshal(tt.want)
			mockClient.On("CreateChatCompletion", mock.Anything, mock.Anything).Return(openai.ChatCompletionResponse{
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
			c := NewComposer(mockClient)
			got, err := c.ChooseMostImportantNews(tt.args.ctx, tt.args.news)
			if (err != nil) != tt.wantErr {
				t.Errorf("Composer.ChooseMostImportantNews() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("Composer.ChooseMostImportantNews() wrong news len = %v, want %v", len(got), len(tt.want))
			}

			for i, n := range got {
				if !reflect.DeepEqual(n, tt.want[i]) {
					t.Errorf("Composer.ChooseMostImportantNews() = %v, want %v", n, tt.want[i])
				}
			}
		})
	}
}

func TestComposer_findNewsMetaData(t *testing.T) {
	type args struct {
		ctx  context.Context
		news []*journalist.News
	}
	tests := []struct {
		name    string
		args    args
		mockRes string
		want    map[string]*NewsMeta
		wantErr bool
	}{
		{
			name: "Should pass and return correct meta data",
			args: args{
				ctx: context.Background(),
				news: []*journalist.News{
					{
						ID:          "1234",
						Title:       "Up 10% In The Last One Month, What's Next For Morgan Stanley Stock?",
						Description: "Morgan Stanley&amp;rsquo;s stock&amp;nbsp;(NYSE: MS) has lost roughly 6% YTD, as compared to the 18% rise in the S&amp;amp;P500 over the same period. Further, the stock is currently trading at $80 per share, which is 11% below its fair value of $90 &amp;ndash; Trefis&amp;rsquo; estimate for&amp;nbsp;Mor",
					},
				},
			},
			mockRes: "[{\n  \"id\": \"1234\",\n  \"tickers\": [\"MS\"],\n  \"markets\": [\"SPY\"],\n  \"hashtags\": []\n}]",
			want: map[string]*NewsMeta{
				"1234": {
					Tickers:  []string{"MS"},
					Markets:  []string{"SPY"},
					Hashtags: []string{},
				},
			},
			wantErr: false,
		},
		{
			name: "Should pass and return empty meta data",
			args: args{
				ctx: context.Background(),
				news: []*journalist.News{
					{
						ID:          "1",
						Title:       "Blah blah blah",
						Description: "Blah blah blah",
					},
				},
			},
			mockRes: "[{\n  \"id\": \"1\",\n  \"tickers\": [],\n  \"markets\": [],\n  \"hashtags\": []\n}]",
			want: map[string]*NewsMeta{
				"1": {
					Tickers:  []string{},
					Markets:  []string{},
					Hashtags: []string{},
				},
			},
			wantErr: false,
		},
		{
			name: "Should return error if OpenAI fails",
			args: args{
				ctx: context.Background(),
				news: []*journalist.News{
					{
						ID:          "1",
						Title:       "Blah blah blah",
						Description: "Blah blah blah",
					},
				},
			},
			mockRes: "",
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		mockClient := new(MockOpenAiClient)

		if tt.wantErr {
			mockError := errors.New("some error")
			mockClient.On("CreateChatCompletion", mock.Anything, mock.Anything).Return(openai.ChatCompletionResponse{}, mockError)
		} else {
			mockClient.On("CreateChatCompletion", mock.Anything, mock.Anything).Return(openai.ChatCompletionResponse{
				Choices: []openai.ChatCompletionChoice{
					{
						Message: openai.ChatCompletionMessage{
							Content: tt.mockRes,
						},
					},
				},
			}, nil)
		}

		t.Run(tt.name, func(t *testing.T) {
			c := NewComposer(mockClient)
			got, err := c.findNewsMetaData(tt.args.ctx, tt.args.news)
			if (err != nil) != tt.wantErr {
				t.Errorf("findNewsMetaData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("findNewsMetaData() got = %v, want %v", got, tt.want)
			}
		})
	}
}
