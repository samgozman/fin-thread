package main

import (
	"context"
	"fmt"
	"github.com/go-co-op/gocron"
	"strings"
	"time"

	"github.com/samgozman/go-fin-feed/journalist"
)

// TODO: Move journalist groups and their jobs to another file with App structure
func main() {
	marketNews := journalist.NewJournalist([]journalist.NewsProvider{
		journalist.NewRssProvider("bloomberg:markets", "https://feeds.bloomberg.com/markets/news.rss"),
		journalist.NewRssProvider("cnbc:finance", "https://search.cnbc.com/rs/search/combinedcms/view.xml?partnerId=wrss01&id=10000664"),
		journalist.NewRssProvider("nasdaq:markets", "https://www.nasdaq.com/feed/rssoutbound?category=Markets"),
		journalist.NewRssProvider("wsj:markets", "https://feeds.a.dj.com/rss/RSSMarketsMain.xml"),
	})

	tradingEconomicsNews := journalist.NewJournalist([]journalist.NewsProvider{
		journalist.NewRssProvider("trading-economics:repo-rate", "https://tradingeconomics.com/rss/news.aspx?i=repo+rate"),
		journalist.NewRssProvider("trading-economics:european-union", "https://tradingeconomics.com/european-union/rss"),
		journalist.NewRssProvider("trading-economics:food-inflation", "https://tradingeconomics.com/rss/news.aspx?i=food+inflation"),
		journalist.NewRssProvider("trading-economics:inflation-rate-mom", "https://tradingeconomics.com/rss/news.aspx?i=inflation+rate+mom"),
		journalist.NewRssProvider("trading-economics:core-inflation-rate-mom", "https://tradingeconomics.com/rss/news.aspx?i=core+inflation+rate+mom"),
		journalist.NewRssProvider("trading-economics:wholesale-prices-mom", "https://tradingeconomics.com/rss/news.aspx?i=wholesale+prices+mom"),
		journalist.NewRssProvider("trading-economics:weapons-sales", "https://tradingeconomics.com/rss/news.aspx?i=weapons+sales"),
		journalist.NewRssProvider("trading-economics:rent-inflation", "https://tradingeconomics.com/rss/news.aspx?i=rent+inflation"),
		journalist.NewRssProvider("trading-economics:housing-index", "https://tradingeconomics.com/rss/news.aspx?i=housing+index"),
		journalist.NewRssProvider("trading-economics:housing-starts", "https://tradingeconomics.com/rss/news.aspx?i=housing+starts"),
		journalist.NewRssProvider("trading-economics:households-debt-to-gdp", "https://tradingeconomics.com/rss/news.aspx?i=households+debt+to+gdp"),
		journalist.NewRssProvider("trading-economics:government-debt", "https://tradingeconomics.com/rss/news.aspx?i=government+debt"),
	})

	s := gocron.NewScheduler(time.UTC)
	// .WaitForSchedule()
	_, err := s.Every(60).Second().Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		news, e := marketNews.GetLatestNews(ctx, time.Now().Add(-1*time.Minute))
		if e != nil {
			fmt.Println(e)
			return
		}

		for _, n := range news {
			fmt.Println(n)
		}
	})
	if err != nil {
		return
	}

	_, err = s.Every(90).Second().Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		news, e := tradingEconomicsNews.GetLatestNews(ctx, time.Now().Add(-90*time.Second))
		if e != nil {
			fmt.Println(e)
			return
		}

		// Filter news by keywords, if Title do not contain any of the keywords - skip it
		// TODO: Move to utils, remove N+1 complexity
		filterKeywords := []string{"European Union", "United States", "United Kingdom", "China", "Germany", "France", "Japan", "Italy", "India"}
		var filteredNews []*journalist.News
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
	})
	if err != nil {
		return
	}

	defer s.Stop()
	s.StartAsync()

	select {}
}
