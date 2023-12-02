package journalist

import (
	"context"
	"sync"
	"time"
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

// GetLatestNews fetches the latest news (until date) from all providers and merges them into unified list.
// Errors are collected and returned in array of ProviderErr
func (j *Journalist) GetLatestNews(ctx context.Context, until time.Time) (NewsList, []error) {
	// Create channels to collect results and errors
	resultCh := make(chan NewsList, len(j.providers))
	errorCh := make(chan error, len(j.providers))

	// Use WaitGroup to wait for all goroutines to finish
	var wg sync.WaitGroup

	for i := 0; i < len(j.providers); i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			result, err := j.providers[id].Fetch(ctx, until)
			if err != nil {
				errorCh <- err
				return
			}

			resultCh <- result
		}(i)
	}

	wg.Wait()
	close(resultCh)
	close(errorCh)

	// Collect results and errors from channels
	var results NewsList
	var errors []error

	for result := range resultCh {
		results = append(results, result...)
	}

	// TODO: join errors into one errors.Join
	for err := range errorCh {
		errors = append(errors, err)
	}

	return results, errors
}
