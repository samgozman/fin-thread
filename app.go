package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/samgozman/go-fin-feed/journalist"
)

type App struct {
	staff Staff
}

func (a *App) CreateMarketNewsJob(until time.Time) JobFunc {
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		news, e := a.staff.marketJournalist.GetLatestNews(ctx, until)
		if e != nil {
			fmt.Println(e)
			return
		}

		for _, n := range news {
			fmt.Println(n)
		}
	}
}

func (a *App) CreateTradingEconomicsNewsJob(until time.Time) JobFunc {
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		news, e := a.staff.teJournalist.GetLatestNews(ctx, until)
		if e != nil {
			fmt.Println(e)
			return
		}

		// Filter news by keywords, if Title do not contain any of the keywords - skip it
		filterKeywords := []string{"European Union", "United States", "United Kingdom", "China", "Germany", "France", "Japan", "Italy", "India"}
		var filteredNews []*News
		for _, n := range news {
			c := false
			// Check if any keyword is present in the title
			for _, k := range filterKeywords {
				if strings.Contains(n.Title, k) {
					c = true
					break
				}
			}
			if c {
				filteredNews = append(filteredNews, n)
			}
		}

		for _, n := range filteredNews {
			fmt.Println(n)
		}
	}
}

// Staff is the structure that holds all the journalists
type Staff struct {
	// Journalist for common market news
	marketJournalist *Journalist
	// Specialized journalist for Trading Economics news RSS feed only
	teJournalist *Journalist
}

type JobFunc func()
