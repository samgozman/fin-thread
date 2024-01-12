package stocks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Screener is a struct to fetch all available Stocks from external API.
type Screener struct{}

// FetchAll fetches all available Stocks from external API.
func (f *Screener) FetchAll(ctx context.Context) ([]Stock, error) {
	url := "https://api.nasdaq.com/api/screener/stocks?tableonly=true&limit=25&offset=0&download=true"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request to fetch stocks from nasdaq: %w", err)
	}
	req = req.WithContext(ctx)

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

	return respParsed.Data.Rows, nil
}

type Stock struct {
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
}

type nasdaqScreenerResponse struct {
	Data struct {
		AsOf    string  `json:"asOf"`    // unnecessary, but keeping it for JSON unmarshalling
		Headers any     `json:"headers"` // unnecessary, but keeping it for JSON unmarshalling
		Rows    []Stock `json:"rows"`    // Stocks array
	} `json:"data"`
	Message string `json:"message"`
	Status  any    `json:"status"` // Status object, probably not needed for this project
}
