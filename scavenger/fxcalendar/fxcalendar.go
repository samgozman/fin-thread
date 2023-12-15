package fxcalendar

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/samgozman/fin-thread/utils"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	ForexCalendarUrl = "https://www.mql5.com/en/economic-calendar/content"
)

// ForexCalendar is the struct that fetches the calendar events from the mql5.com website.
// Calendar page: https://www.mql5.com/en/economic-calendar
type ForexCalendar struct{}

// Fetch fetches the calendar events from the mql5.com website.
func (c *ForexCalendar) Fetch(ctx context.Context) ([]*ForexCalendarEvent, error) {
	// importance=9 - high impact
	// currencies=65743 - CHF, EUR, GBP, JPY, USD, CNY, INR
	var data = strings.NewReader(`date_mode=1&from=2023-12-11T00%3A00%3A00&to=2023-12-17T23%3A59%3A59&importance=9&currencies=65743`)
	req, err := http.NewRequest(http.MethodPost, ForexCalendarUrl, data)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	req.Header.Set("accept", "*/*")
	req.Header.Set("content-type", "application/x-www-form-urlencoded")
	req.Header.Set("x-requested-with", "XMLHttpRequest")
	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	client := http.DefaultClient
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("invalid status code error: %d, value %s", res.StatusCode, res.Status))
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("error reading response body: %s", err))
	}
	err = res.Body.Close()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("error closing response body: %s", err))
	}

	// Unmarshal the response
	var mql5Events []mql5Calendar
	if err := json.Unmarshal(body, &mql5Events); err != nil {
		return nil, errors.New(fmt.Sprintf("error unmarshalling response body: %s", err))
	}

	var events []*ForexCalendarEvent
	for _, event := range mql5Events {
		e, err := parseEvent(&event)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}

	return events, nil
}

// parseEvent parses a single event from the calendar
func parseEvent(event *mql5Calendar) (*ForexCalendarEvent, error) {
	// Parse currency
	var currency ForexCalendarCurrency
	switch event.CurrencyCode {
	case "USD":
		currency = ForexCalendarCurrencyUSD
	case "EUR":
		currency = ForexCalendarCurrencyEUR
	case "GBP":
		currency = ForexCalendarCurrencyGBP
	case "JPY":
		currency = ForexCalendarCurrencyJPY
	case "CHF":
		currency = ForexCalendarCurrencyCHF
	case "CNY":
		currency = ForexCalendarCurrencyCNY
	case "AUD":
		currency = ForexCalendarCurrencyAUD
	case "NZD":
		currency = ForexCalendarCurrencyNZD
	case "INR":
		currency = ForexCalendarCurrencyINR
	default:
		return nil, errors.New(fmt.Sprintf("unknown currency: %s", event.CurrencyCode))
	}

	// Parse impact
	var impact ForexCalendarImpact
	switch event.Importance {
	case "low":
		impact = ForexCalendarImpactLow
	case "medium":
		impact = ForexCalendarImpactMedium
	case "high":
		impact = ForexCalendarImpactHigh
	case "none":
		if event.EventType == 2 {
			impact = ForexCalendarImpactHoliday
		} else {
			impact = ForexCalendarImpactNone
		}
	default:
		return nil, errors.New(fmt.Sprintf("unknown impact: %s", event.Importance))
	}

	// Parse dates
	dt, err := utils.ParseDate(event.FullDate)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("error parsing date: %s, value %s", err, event.FullDate))
	}
	et, err := utils.ParseDate(event.ReleaseDate)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("error parsing date: %s, value %v", err, event.ReleaseDate))
	}

	e := &ForexCalendarEvent{
		DateTime:  dt,
		EventTime: et,
		Currency:  currency,
		Impact:    impact,
		Title:     event.EventName,
		Actual:    event.ActualValue,
		Forecast:  event.ForecastValue,
		Previous:  event.PreviousValue,
	}

	return e, nil
}

// ForexCalendarCurrency impacted currencies(economic markets) by the event
type ForexCalendarCurrency = string

const (
	ForexCalendarCurrencyUSD ForexCalendarCurrency = "USD" // US Dollar
	ForexCalendarCurrencyEUR ForexCalendarCurrency = "EUR" // Euro
	ForexCalendarCurrencyGBP ForexCalendarCurrency = "GBP" // British Pound
	ForexCalendarCurrencyJPY ForexCalendarCurrency = "JPY" // Japanese Yen
	ForexCalendarCurrencyCHF ForexCalendarCurrency = "CHF" // Swiss Franc
	ForexCalendarCurrencyCNY ForexCalendarCurrency = "CNY" // Chinese Yuan
	ForexCalendarCurrencyAUD ForexCalendarCurrency = "AUD" // Australian Dollar
	ForexCalendarCurrencyNZD ForexCalendarCurrency = "NZD" // New Zealand Dollar
	ForexCalendarCurrencyINR ForexCalendarCurrency = "INR" // Indian Rupee
)

// ForexCalendarImpact impact of the event on the market
type ForexCalendarImpact = string

const (
	ForexCalendarImpactLow     ForexCalendarImpact = "Low"      // Low impact event
	ForexCalendarImpactMedium  ForexCalendarImpact = "Medium"   // Medium impact event
	ForexCalendarImpactHigh    ForexCalendarImpact = "High"     // High impact event
	ForexCalendarImpactHoliday ForexCalendarImpact = "Holidays" // Holiday event
	ForexCalendarImpactNone    ForexCalendarImpact = "None"     // No impact event
)

type ForexCalendarEvent struct {
	DateTime  time.Time             // Date of the event
	EventTime time.Time             // Time of the event (if available)
	Currency  ForexCalendarCurrency // Currency impacted by the event
	Impact    ForexCalendarImpact   // Impact of the event on the market
	Title     string                // Event title
	Actual    string                // Actual value of the event (if available)
	Forecast  string                // Forecasted value of the event (if available)
	Previous  string                // Previous value of the event (if available)
}

// MQL5 calendar event object
type mql5Calendar struct {
	Id               int         `json:"Id"`
	EventType        int         `json:"EventType"`
	TimeMode         int         `json:"TimeMode"`
	Processed        int         `json:"Processed"`
	Url              string      `json:"Url"`
	EventName        string      `json:"EventName"`
	Importance       string      `json:"Importance"`
	CurrencyCode     string      `json:"CurrencyCode"`
	ForecastValue    string      `json:"ForecastValue"`
	PreviousValue    string      `json:"PreviousValue"`
	OldPreviousValue string      `json:"OldPreviousValue"`
	ActualValue      string      `json:"ActualValue"`
	ReleaseDate      int64       `json:"ReleaseDate"`
	ImpactDirection  int         `json:"ImpactDirection"`
	ImpactValue      string      `json:"ImpactValue"`
	ImpactValueF     string      `json:"ImpactValueF"`
	Country          int         `json:"Country"`
	CountryName      interface{} `json:"CountryName"`
	FullDate         string      `json:"FullDate"`
}
