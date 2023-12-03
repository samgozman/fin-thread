package journalist

import (
	"context"
	"errors"
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
func (j *Journalist) GetLatestNews(ctx context.Context, until time.Time) (NewsList, error) {
	// Create channels to collect results and e
	resultCh := make(chan NewsList, len(j.providers))
	errorCh := make(chan error, len(j.providers))

	// Use WaitGroup to wait for all goroutines to finish
	var wg sync.WaitGroup

	for i := 0; i < len(j.providers); i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

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

	// Collect results and e from channels
	var results NewsList
	var e []error

	for result := range resultCh {
		results = append(results, result...)
	}

	for err := range errorCh {
		e = append(e, err)
	}

	return results, errors.Join(e...)
}
