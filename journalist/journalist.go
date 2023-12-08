package journalist

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Journalist is the main struct that fetches the news from all providers and merges them into unified list
type Journalist struct {
	Name       string // Name of the journalist (for logging purposes)
	providers  []NewsProvider
	flagKeys   []string // Keys that will "flag" the news as something that should be double-checked by human
	filterKeys []string // Keys that will remove the news from the list if they do not contain them
	limitNews  int      // Limit the number of news to fetch from each provider
}

// NewJournalist creates a new Journalist instance
func NewJournalist(name string, providers []NewsProvider) *Journalist {
	return &Journalist{
		Name:      name,
		providers: providers,
	}
}

// FlagByKeys sets the keys that will "flag" news that contain them by setting News.IsSuspicious to true
func (j *Journalist) FlagByKeys(flagKeys []string) *Journalist {
	j.flagKeys = flagKeys
	return j
}

// FilterByKeys sets the keys that will remove news that do not contain them
func (j *Journalist) FilterByKeys(filterKeys []string) *Journalist {
	j.filterKeys = filterKeys
	return j
}

// Limit sets the limit of news to fetch from each provider
func (j *Journalist) Limit(limit int) *Journalist {
	j.limitNews = limit
	return j
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

			// Limit the number of news to fetch from each provider if limitNews > 0
			if j.limitNews > 0 && len(result) > j.limitNews {
				result = result[:j.limitNews]
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

	results = results.MapIDs()

	if len(j.filterKeys) > 0 {
		results = results.FilterByKeywords(j.filterKeys)
	}

	if len(j.flagKeys) > 0 {
		results.FlagByKeywords(j.flagKeys)
	}

	for err := range errorCh {
		e = append(e, err)
	}

	return results, errors.Join(e...)
}
