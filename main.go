package main

import (
	"github.com/go-co-op/gocron"
	"time"

	. "github.com/samgozman/go-fin-feed/journalist"
)

// TODO: Add TgPublisher to App structure and use it in the jobs
func main() {
	app := App{
		staff: Staff{
			marketJournalist: NewJournalist([]NewsProvider{
				NewRssProvider("bloomberg:markets", "https://feeds.bloomberg.com/markets/news.rss"),
				NewRssProvider("cnbc:finance", "https://search.cnbc.com/rs/search/combinedcms/view.xml?partnerId=wrss01&id=10000664"),
				NewRssProvider("nasdaq:markets", "https://www.nasdaq.com/feed/rssoutbound?category=Markets"),
				NewRssProvider("wsj:markets", "https://feeds.a.dj.com/rss/RSSMarketsMain.xml"),
			}),
			teJournalist: NewJournalist([]NewsProvider{
				NewRssProvider("trading-economics:repo-rate", "https://tradingeconomics.com/rss/news.aspx?i=repo+rate"),
				NewRssProvider("trading-economics:european-union", "https://tradingeconomics.com/european-union/rss"),
				NewRssProvider("trading-economics:food-inflation", "https://tradingeconomics.com/rss/news.aspx?i=food+inflation"),
				NewRssProvider("trading-economics:inflation-rate-mom", "https://tradingeconomics.com/rss/news.aspx?i=inflation+rate+mom"),
				NewRssProvider("trading-economics:core-inflation-rate-mom", "https://tradingeconomics.com/rss/news.aspx?i=core+inflation+rate+mom"),
				NewRssProvider("trading-economics:wholesale-prices-mom", "https://tradingeconomics.com/rss/news.aspx?i=wholesale+prices+mom"),
				NewRssProvider("trading-economics:weapons-sales", "https://tradingeconomics.com/rss/news.aspx?i=weapons+sales"),
				NewRssProvider("trading-economics:rent-inflation", "https://tradingeconomics.com/rss/news.aspx?i=rent+inflation"),
				NewRssProvider("trading-economics:housing-index", "https://tradingeconomics.com/rss/news.aspx?i=housing+index"),
				NewRssProvider("trading-economics:housing-starts", "https://tradingeconomics.com/rss/news.aspx?i=housing+starts"),
				NewRssProvider("trading-economics:households-debt-to-gdp", "https://tradingeconomics.com/rss/news.aspx?i=households+debt+to+gdp"),
				NewRssProvider("trading-economics:government-debt", "https://tradingeconomics.com/rss/news.aspx?i=government+debt"),
			}),
		},
	}

	s := gocron.NewScheduler(time.UTC)
	// .WaitForSchedule()
	_, err := s.Every(60).Second().Do(app.CreateMarketNewsJob(time.Now().Add(-60 * time.Second)))
	if err != nil {
		return
	}

	_, err = s.Every(90).Second().Do(app.CreateTradingEconomicsNewsJob(time.Now().Add(-90 * time.Second)))
	if err != nil {
		return
	}

	defer s.Stop()
	s.StartAsync()

	select {}
}
