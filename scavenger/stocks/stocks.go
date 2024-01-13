package stocks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Screener is a struct to fetch all available Stocks from external API.
type Screener struct{}

// FetchAll fetches all available Stocks from external API
// and returns them as a map of `ticker` -> Stock.
func (f *Screener) FetchAll(ctx context.Context) (*StockMap, error) {
	url := "https://api.nasdaq.com/api/screener/stocks?tableonly=true&limit=25&offset=0&download=true"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request to fetch stocks from nasdaq: %w", err)
	}
	req = req.WithContext(ctx)
	req.Header.Set("accept", "application/json")
	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	client := &http.Client{}
	resp, err := client.Do(req) //nolint:bodyclose
	if err != nil {
		return nil, fmt.Errorf("error fetching stocks from nasdaq: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println("error closing response body:", err)
		}
	}(resp.Body)

	var respParsed nasdaqScreenerResponse
	if err := json.NewDecoder(resp.Body).Decode(&respParsed); err != nil {
		return nil, fmt.Errorf("error parsing response from nasdaq: %w", err)
	}

	stockMap := make(StockMap)
	for _, stock := range respParsed.Data.Rows {
		// replace / with . in ticker to match the format of other sources (BRK/A -> BRK.A)
		s := strings.ReplaceAll(stock.Symbol, "/", ".")
		if strings.Contains(s, "^") { // Exclude tickers with ^ separator
			continue
		}
		stockMap[s] = Stock{
			Name:      stock.Name,
			MarketCap: stock.MarketCap,
			Country:   stock.Country,
			Industry:  stock.Industry,
			Sector:    stock.Sector,
		}
	}

	return &stockMap, nil
}

type Stock struct {
	Name      string `json:"name"`
	MarketCap string `json:"marketCap"`
	Country   string `json:"country"`
	Industry  string `json:"industry"`
	Sector    string `json:"sector"`
}

// StockMap is a map of `ticker` -> Stock.
type StockMap map[string]Stock

type nasdaqScreenerResponse struct {
	Data struct {
		AsOf    string `json:"asOf"`    // unnecessary, but keeping it for JSON unmarshalling
		Headers any    `json:"headers"` // unnecessary, but keeping it for JSON unmarshalling
		Rows    []struct {
			Symbol    string `json:"symbol"`
			Name      string `json:"name"`
			LastSale  string `json:"lastsale"`
			NetChange string `json:"netchange"`
			PctChange string `json:"pctchange"`
			Volume    string `json:"volume"`
			MarketCap string `json:"marketCap"`
			Country   string `json:"country"`
			IPOYear   string `json:"ipoyear"`
			Industry  string `json:"industry"`
			Sector    string `json:"sector"`
			URL       string `json:"url"`
		} `json:"rows"` // Stocks array
	} `json:"data"`
	Message string `json:"message"`
	Status  any    `json:"status"` // Status object, probably not needed for this project
}
