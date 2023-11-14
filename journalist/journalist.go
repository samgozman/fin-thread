package journalist

import "context"

// Journalist is the main struct that fetches the news from all providers and merges them into unified list
type Journalist struct {
	providers []NewsProvider
}

// NewJournalist creates a new Journalist instance
func NewJournalist(providers []NewsProvider) *Journalist {
	return &Journalist{
		providers: providers,
	}
}

// GetLatestNews fetches the latest news from all providers and merges them into unified list
func (*Journalist) GetLatestNews(ctx context.Context) ([]News, error) {
	// TODO: run all providers in parallel and merge the results
	return nil, nil
}
