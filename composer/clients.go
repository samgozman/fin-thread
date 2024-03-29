package composer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/generative-ai-go/genai"
	"github.com/samgozman/fin-thread/pkg/errlvl"
	"github.com/sashabaranov/go-openai"
	"google.golang.org/api/option"
	"io"
	"net/http"
)

// openAiClientInterface is an interface for OpenAI API client.
type openAiClientInterface interface {
	CreateChatCompletion(ctx context.Context, req openai.ChatCompletionRequest) (response openai.ChatCompletionResponse, err error)
}

// togetherAIClientInterface is an interface for TogetherAI API client.
type togetherAIClientInterface interface {
	CreateChatCompletion(ctx context.Context, options togetherAIRequest) (*TogetherAIResponse, error)
}

// togetherAIRequest is a struct that contains options for TogetherAI API requests.
type togetherAIRequest struct {
	Model             string   `json:"model"`
	Prompt            string   `json:"prompt"`
	MaxTokens         int      `json:"max_tokens"`
	Temperature       float64  `json:"temperature"`
	TopP              float64  `json:"top_p"`
	TopK              int      `json:"top_k"`
	RepetitionPenalty float64  `json:"repetition_penalty"`
	Stop              []string `json:"stop"`
}

// TogetherAIResponse is a struct that contains response from TogetherAI API.
//
//goland:noinspection GoUnnecessarilyExportedIdentifiers
type TogetherAIResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Text string `json:"text"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	}
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Object  string `json:"object"`
}

// TogetherAI client to interact with TogetherAI API (replacement for OpenAI API in some cases).
type TogetherAI struct {
	APIKey string
	URL    string
}

// CreateChatCompletion creates a new chat completion request to TogetherAI API.
func (t *TogetherAI) CreateChatCompletion(ctx context.Context, options togetherAIRequest) (*TogetherAIResponse, error) {
	bodyJSON, err := json.Marshal(options)
	if err != nil {
		return nil, fmt.Errorf("error marshalling JSON: %w with value %v", err, options)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", t.URL, bytes.NewBuffer(bodyJSON))
	if err != nil {
		return nil, newError(
			fmt.Errorf("error creating request: %w", err),
			errlvl.ERROR,
			"TogetherAI.CreateChatCompletion",
			"NewRequestWithContext",
		)
	}

	req.Header.Set("Authorization", "Bearer "+t.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req) //nolint:bodyclose
	if err != nil {
		return nil, newError(
			fmt.Errorf("error sending request: %w", err),
			errlvl.ERROR,
			"TogetherAI.CreateChatCompletion",
			"client.Do",
		)
	}

	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			return
		}
	}(resp.Body)

	var response TogetherAIResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, newError(
			fmt.Errorf("error decoding response: %w", err),
			errlvl.ERROR,
			"TogetherAI.CreateChatCompletion",
			"json.NewDecoder",
		)
	}

	return &response, nil
}

// NewTogetherAI creates new TogetherAI client.
func NewTogetherAI(apiKey string) *TogetherAI {
	return &TogetherAI{
		APIKey: apiKey,
		URL:    "https://api.together.xyz/completions",
	}
}

type GoogleGeminiClientInterface interface {
	CreateChatCompletion(ctx context.Context, req GoogleGeminiRequest) (response *genai.GenerateContentResponse, err error)
}

// GoogleGeminiRequest is a struct that contains options for Google Gemini API requests.
type GoogleGeminiRequest struct {
	Prompt      string  `json:"prompt"`
	MaxTokens   int32   `json:"max_tokens"`
	Temperature float32 `json:"temperature"`
	TopP        float32 `json:"top_p"`
	TopK        int32   `json:"top_k"`
}

// GoogleGemini is a structure for Google Gemini AI API client.
// ! https://ai.google.dev/available_regions#available_regions
// ! Gemini is not available in EU region yet.
type GoogleGemini struct {
	APIKey string
}

// NewGoogleGemini creates new Google Gemini client.
func NewGoogleGemini(apiKey string) *GoogleGemini {
	return &GoogleGemini{
		APIKey: apiKey,
	}
}

// CreateChatCompletion creates a new chat completion request to Google Gemini API.
func (g *GoogleGemini) CreateChatCompletion(ctx context.Context, req GoogleGeminiRequest) (response *genai.GenerateContentResponse, err error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(g.APIKey))
	if err != nil {
		return nil, fmt.Errorf("error creating Google Gemini client: %w", err)
	}
	defer func(client *genai.Client) {
		err = client.Close()
		if err != nil {
			return
		}
	}(client)

	model := client.GenerativeModel("gemini-pro")
	model.SetTemperature(req.Temperature)
	model.SetTopP(req.TopP)
	model.SetTopK(req.TopK)
	model.SetMaxOutputTokens(req.MaxTokens)

	resp, err := model.GenerateContent(ctx, genai.Text(req.Prompt))
	if err != nil {
		return nil, newError(
			fmt.Errorf("error generating content: %w", err),
			errlvl.ERROR,
			"GoogleGemini.CreateChatCompletion",
			"model.GenerateContent",
		)
	}

	return resp, nil
}
