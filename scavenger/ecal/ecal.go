package ecal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/samgozman/fin-thread/utils"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

const (
	EconomicCalendarUrl = "https://www.mql5.com/en/economic-calendar/content"
)

// EconomicCalendar is the struct for economics calendar fetcher
type EconomicCalendar struct{}

// Fetch fetches economics events for the specified period
func (c *EconomicCalendar) Fetch(ctx context.Context, from, to time.Time) (EconomicCalendarEvents, error) {
	if from.IsZero() || to.IsZero() {
		return nil, errors.New(fmt.Sprintf("invalid date range: from %v, to %v", from, to))
	}

	if from.After(to) {
		return nil, errors.New(fmt.Sprintf("invalid date range: from %v, to %v", from, to))
	}

	if to.Sub(from) > 7*24*time.Hour {
		return nil, errors.New(fmt.Sprintf("invalid date range (more than 7 days): from %v, to %v", from, to))
	}

	fmt.Println("Fetching economic calendar events from", from, "to", to)

	// Create request body with the specified date range
	f := from.Format("2006-01-02T15:04:05")
	t := to.Format("2006-01-02T15:04:05")
	data := strings.NewReader(
		// importance=9 - high impact
		// currencies=65743 - CHF, EUR, GBP, JPY, USD, CNY, INR
		fmt.Sprintf("date_mode=1&from=%s&to=%s&importance=9&currencies=65743", f, t),
	)
	req, err := http.NewRequest(http.MethodPost, EconomicCalendarUrl, data)
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

	var events EconomicCalendarEvents
	for _, event := range mql5Events {
		e, err := parseEvent(&event)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}

	// Need to remove events that are not in the specified date range.
	// MQL5 API returns events for one extra day for some reason.
	events = events.Distinct().FilterByDateRange(from, to)
	events.SortByDate()

	return events, nil
}

// parseEvent parses a single event from the calendar
func parseEvent(event *mql5Calendar) (*EconomicCalendarEvent, error) {
	// Parse currency
	var currency EconomicCalendarCurrency
	switch event.CurrencyCode {
	case "USD":
		currency = EconomicCalendarUSD
	case "EUR":
		currency = EconomicCalendarEUR
	case "GBP":
		currency = EconomicCalendarGBP
	case "JPY":
		currency = EconomicCalendarJPY
	case "CHF":
		currency = EconomicCalendarCHF
	case "CNY":
		currency = EconomicCalendarCNY
	case "AUD":
		currency = EconomicCalendarAUD
	case "NZD":
		currency = EconomicCalendarNZD
	case "INR":
		currency = EconomicCalendarINR
	default:
		return nil, errors.New(fmt.Sprintf("unknown currency: %s", event.CurrencyCode))
	}

	// Parse impact
	var impact EconomicCalendarImpact
	switch event.Importance {
	case "low":
		impact = EconomicCalendarImpactLow
	case "medium":
		impact = EconomicCalendarImpactMedium
	case "high":
		impact = EconomicCalendarImpactHigh
	case "none":
		if event.EventType == 2 {
			impact = EconomicCalendarImpactHoliday
		} else {
			impact = EconomicCalendarImpactNone
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

	e := &EconomicCalendarEvent{
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

// EconomicCalendarCurrency impacted currencies(economic markets) by the event
type EconomicCalendarCurrency = string

const (
	EconomicCalendarUSD EconomicCalendarCurrency = "USD" // US Dollar
	EconomicCalendarEUR EconomicCalendarCurrency = "EUR" // Euro
	EconomicCalendarGBP EconomicCalendarCurrency = "GBP" // British Pound
	EconomicCalendarJPY EconomicCalendarCurrency = "JPY" // Japanese Yen
	EconomicCalendarCHF EconomicCalendarCurrency = "CHF" // Swiss Franc
	EconomicCalendarCNY EconomicCalendarCurrency = "CNY" // Chinese Yuan
	EconomicCalendarAUD EconomicCalendarCurrency = "AUD" // Australian Dollar
	EconomicCalendarNZD EconomicCalendarCurrency = "NZD" // New Zealand Dollar
	EconomicCalendarINR EconomicCalendarCurrency = "INR" // Indian Rupee
)

// EconomicCalendarCountryEmoji is the map of country code to emoji symbol
var EconomicCalendarCountryEmoji = map[EconomicCalendarCurrency]string{
	EconomicCalendarUSD: "🇺🇸",
	EconomicCalendarEUR: "🇪🇺",
	EconomicCalendarGBP: "🇬🇧",
	EconomicCalendarJPY: "🇯🇵",
	EconomicCalendarCHF: "🇨🇭",
	EconomicCalendarCNY: "🇨🇳",
	EconomicCalendarAUD: "🇦🇺",
	EconomicCalendarNZD: "🇳🇿",
	EconomicCalendarINR: "🇮🇳",
}

// EconomicCalendarImpact impact of the event on the market (low, medium, high, holiday, none)
type EconomicCalendarImpact = string

const (
	EconomicCalendarImpactLow     EconomicCalendarImpact = "Low"      // Low impact event
	EconomicCalendarImpactMedium  EconomicCalendarImpact = "Medium"   // Medium impact event
	EconomicCalendarImpactHigh    EconomicCalendarImpact = "High"     // High impact event
	EconomicCalendarImpactHoliday EconomicCalendarImpact = "Holidays" // Holiday event
	EconomicCalendarImpactNone    EconomicCalendarImpact = "None"     // No impact event
)

// EconomicCalendarEvent is the struct for economics calendar event object
type EconomicCalendarEvent struct {
	DateTime  time.Time                // Date of the event
	EventTime time.Time                // Time of the event (if available)
	Currency  EconomicCalendarCurrency // Currency impacted by the event
	Impact    EconomicCalendarImpact   // Impact of the event on the market
	Title     string                   // Event title
	Actual    string                   // Actual value of the event (if available)
	Forecast  string                   // Forecasted value of the event (if available)
	Previous  string                   // Previous value of the event (if available)
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

// EconomicCalendarEvents is the slice of economics calendar events
type EconomicCalendarEvents []*EconomicCalendarEvent

// FilterByDateRange filters events by date range, returns new slice
func (e EconomicCalendarEvents) FilterByDateRange(from, to time.Time) EconomicCalendarEvents {
	var filtered EconomicCalendarEvents
	for _, v := range e {
		if v.DateTime.After(from) && v.DateTime.Before(to) {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

// SortByDate sorts events by date (ascending)
func (e EconomicCalendarEvents) SortByDate() {
	sort.Slice(e, func(i, j int) bool {
		return e[i].DateTime.Before(e[j].DateTime)
	})
}

// Distinct removes duplicates from the slice
func (e EconomicCalendarEvents) Distinct() EconomicCalendarEvents {
	var distinct EconomicCalendarEvents
	seen := make(map[string]bool)
	for _, v := range e {
		id := fmt.Sprintf("%s%s%s", v.DateTime, v.Title, v.Currency)
		if _, ok := seen[id]; !ok {
			seen[id] = true
			distinct = append(distinct, v)
		}
	}
	return distinct
}
