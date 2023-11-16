package journalist

import (
	"context"
	"fmt"
	"sync"
)

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
func (j *Journalist) GetLatestNews(ctx context.Context) ([]*News, error) {
	// Create channels to collect results and errors
	// TODO: Create a structure to collect results, like {source: "name", result: []*News}
	resultCh := make(chan []*News, len(j.providers))
	errorCh := make(chan error, len(j.providers))

	// Use WaitGroup to wait for all goroutines to finish
	var wg sync.WaitGroup

	for i := 0; i < len(j.providers); i++ {
		wg.Add(1)
		go fetchWorker(&ctx, &wg, &resultCh, &errorCh, &j.providers[i])
	}

	wg.Wait()
	close(resultCh)
	close(errorCh)

	// Collect results and errors from channels
	var results []*News
	var errors []error

	for result := range resultCh {
		results = append(results, result...)
	}

	for err := range errorCh {
		errors = append(errors, err)
	}

	// print results
	for _, result := range results {
		fmt.Println(result)
	}

	// TODO: Return real results
	return nil, nil
}

func fetchWorker(ctx *context.Context, wg *sync.WaitGroup, resCh *chan []*News, errCh *chan error, p *NewsProvider) {
	defer wg.Done()

	result, err := (*p).Fetch(*ctx)
	if err != nil {
		*errCh <- err
		return
	}

	// TODO: Return a structure with source name and result
	*resCh <- result
	return
}
