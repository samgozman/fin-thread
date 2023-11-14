package journalist

import "context"

type Journalist struct {
	providers []NewsProvider
}

func NewJournalist(providers []NewsProvider) *Journalist {
	return &Journalist{
		providers: providers,
	}
}

func (*Journalist) GetLatestNews(ctx context.Context) ([]News, error) {
	// TODO: run all providers in parallel and merge the results
	return nil, nil
}
