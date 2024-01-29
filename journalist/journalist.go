package journalist

import (
	"context"
	"errors"
	"github.com/samgozman/fin-thread/pkg/errlvl"
	"golang.org/x/sync/errgroup"
	"sync"
	"time"
)

// Journalist is the main struct that fetches the news from all providers and merges them into unified list.
type Journalist struct {
	Name      string // Name of the journalist (for logging purposes)
	providers []NewsProvider
	flagKeys  []string // Keys that will "flag" the news as something that should be double-checked by human
	limitNews int      // Limit the number of news to fetch from each provider
}

// NewJournalist creates a new Journalist instance.
func NewJournalist(name string, providers []NewsProvider) *Journalist {
	return &Journalist{
		Name:      name,
		providers: providers,
	}
}

// FlagByKeys sets the keys that will "flag" news that contain them by setting News.IsSuspicious to true.
func (j *Journalist) FlagByKeys(flagKeys []string) *Journalist {
	j.flagKeys = flagKeys
	return j
}

// Limit sets the limit of news to fetch from each provider.
func (j *Journalist) Limit(limit int) *Journalist {
	j.limitNews = limit
	return j
}

// GetLatestNews fetches the latest news (until date) from all providers and merges them into unified list.
func (j *Journalist) GetLatestNews(ctx context.Context, until time.Time) (NewsList, error) {
	// Manage goroutines and errors
	var eg errgroup.Group

	// Use a mutex to safely access shared data (results and errors)
	var mu sync.Mutex
	var results NewsList
	var e []error

	for i := 0; i < len(j.providers); i++ {
		// Capture loop variable
		id := i

		eg.Go(func() error {
			c, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			defer func() {
				if r := recover(); r != nil {
					err, ok := r.(error)
					if !ok {
						err = errPanicUnknown
					}

					mu.Lock()
					defer mu.Unlock()
					e = append(e, errors.Join(errPanicGetLatestNews, err))
				}
			}()

			result, err := j.providers[id].Fetch(c, until)
			if err != nil {
				// Use a mutex to safely append errors
				mu.Lock()
				defer mu.Unlock()
				e = append(e, err)
				return nil // Return nil to continue processing other goroutines
			}

			// Limit the number of news to fetch from each provider if limitNews > 0
			if j.limitNews > 0 && len(result) > j.limitNews {
				result = result[:j.limitNews]
			}

			// Use a mutex to safely append results
			mu.Lock()
			defer mu.Unlock()
			results = append(results, result...)
			return nil
		})
	}

	// Wait for all goroutines to finish
	if err := eg.Wait(); err != nil {
		return nil, newError(errlvl.ERROR, errFetchingNews, err)
	}

	results = results.mapIDs()

	if len(j.flagKeys) > 0 {
		results.flagByKeywords(j.flagKeys)
	}

	return results, errors.Join(e...)
}
