package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	. "github.com/samgozman/go-fin-feed/composer"
	. "github.com/samgozman/go-fin-feed/journalist"
	. "github.com/samgozman/go-fin-feed/publisher"
)

type App struct {
	staff     *Staff
	composer  *Composer
	publisher *TelegramPublisher
}

func (a *App) CreateMarketNewsJob(until time.Time) JobFunc {
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		news, e := a.staff.marketJournalist.GetLatestNews(ctx, until)
		if e != nil {
			fmt.Println(e)
			return
		}

		if len(news) == 0 {
			return
		}

		formattedNews, err := a.PrepareNews(ctx, news)
		if err != nil {
			fmt.Println(err)
			return
		}

		for _, n := range formattedNews {
			err := a.publisher.Publish(n)
			if err != nil {
				fmt.Println(err)
			}
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
		var filteredNews NewsList
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

		if len(news) == 0 {
			return
		}

		formattedNews, err := a.PrepareNews(ctx, news)
		if err != nil {
			fmt.Println(err)
			return
		}

		for _, n := range formattedNews {
			err := a.publisher.Publish(n)
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}

func (a *App) PrepareNews(ctx context.Context, news NewsList) ([]string, error) {
	importantNews, err := a.composer.ChooseMostImportantNews(ctx, news)
	if err != nil {
		return nil, err
	}

	composedNews, err := a.composer.ComposeNews(ctx, importantNews)
	if err != nil {
		return nil, err
	}

	// TODO: Add some custom formatting to the news
	var formattedNews []string
	for _, n := range composedNews {
		str := fmt.Sprintf("ID: %s\nHashtags: %s\nTickers: %s\nMarkets: %s\n%s", n.NewsID, n.MetaData.Hashtags, n.MetaData.Tickers, n.MetaData.Markets, n.Text)
		formattedNews = append(formattedNews, str)
	}

	return formattedNews, nil
}

// Staff is the structure that holds all the journalists
type Staff struct {
	// Journalist for common market news
	marketJournalist *Journalist
	// Specialized journalist for Trading Economics news RSS feed only
	teJournalist *Journalist
}

type JobFunc func()
