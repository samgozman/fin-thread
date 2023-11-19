package composer

import (
	"context"

	"github.com/samgozman/go-fin-feed/journalist"
	"github.com/sashabaranov/go-openai"
)

type Composer struct {
	OpenAiClient *openai.Client
}

func NewComposer(openAiClient *openai.Client) *Composer {
	return &Composer{OpenAiClient: openAiClient}
}

func (c *Composer) ChooseMostImportantNews(ctx context.Context, news []*journalist.News) []*journalist.News {
	// TODO: implement (use OpenAiClient)
	return news
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
