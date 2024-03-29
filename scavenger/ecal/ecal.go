package ecal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/samgozman/fin-thread/internal/utils"
	"github.com/samgozman/fin-thread/pkg/errlvl"
	"io"
	"mime/multipart"
	"net/http"
	"sort"
	"strings"
	"time"
)

const (
	economicCalendarURL = "https://www.mql5.com/en/economic-calendar/content"
)

// EconomicCalendar is the struct for economics calendar fetcher.
type EconomicCalendar struct{}

// Fetch fetches economics events for the specified period.
func (c *EconomicCalendar) Fetch(ctx context.Context, from, to time.Time) (EconomicCalendarEvents, error) {
	if from.IsZero() || to.IsZero() {
		return nil, fmt.Errorf("invalid date range: from %v, to %v", from, to)
	}

	if from.After(to) {
		return nil, errlvl.Wrap(fmt.Errorf("invalid date range: from %v, to %v", from, to), errlvl.ERROR)
	}

	if to.Sub(from) > 7*24*time.Hour {
		return nil, errlvl.Wrap(fmt.Errorf("invalid date range (more than 7 days): from %v, to %v", from, to), errlvl.ERROR)
	}

	// Create request body with the specified date range
	f := from.Format("2006-01-02T15:04:05")
	t := to.Format("2006-01-02T15:04:05")

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)
	_ = writer.WriteField("date_mode", "1")
	_ = writer.WriteField("from", f)
	_ = writer.WriteField("to", t)
	_ = writer.WriteField("importance", "13")    // importance=13 - high impact, holidays and medium
	_ = writer.WriteField("currencies", "65743") // currencies=65743 - CHF, EUR, GBP, JPY, USD, CNY, INR
	err := writer.Close()
	if err != nil {
		return nil, errlvl.Wrap(fmt.Errorf("error closing multipart writer: %w", err), errlvl.ERROR)
	}

	req, err := http.NewRequest(http.MethodPost, economicCalendarURL, payload)
	if err != nil {
		return nil, errlvl.Wrap(fmt.Errorf("error creating calendar request: %w", err), errlvl.ERROR)
	}
	req = req.WithContext(ctx)
	req.Header.Add("x-requested-with", "XMLHttpRequest")
	req.Header.Set("content-type", writer.FormDataContentType())
	req.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	client := http.DefaultClient
	res, err := client.Do(req)
	if err != nil {
		return nil, errlvl.Wrap(fmt.Errorf("error sending calendar request: %w", err), errlvl.ERROR)
	}

	if res.StatusCode != http.StatusOK {
		return nil, errlvl.Wrap(fmt.Errorf("invalid status code error: %d, value %s", res.StatusCode, res.Status), errlvl.ERROR)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errlvl.Wrap(fmt.Errorf("error reading response body: %w", err), errlvl.ERROR)
	}
	err = res.Body.Close()
	if err != nil {
		return nil, errlvl.Wrap(fmt.Errorf("error closing response body: %w", err), errlvl.ERROR)
	}

	// Unmarshal the response
	var mql5Events []mql5Calendar
	if err := json.Unmarshal(body, &mql5Events); err != nil {
		return nil, errlvl.Wrap(fmt.Errorf("error unmarshalling response body: %w", err), errlvl.ERROR)
	}

	var events EconomicCalendarEvents
	for _, event := range mql5Events {
		e, err := parseEvent(event)
		if err != nil {
			return nil, errlvl.Wrap(err, errlvl.ERROR)
		}
		events = append(events, e)
	}

	if events == nil {
		return nil, nil
	}

	// Need to remove events that are not in the specified date range.
	// MQL5 API returns events for one extra day for some reason.
	events = events.Distinct().FilterByDateRange(from, to)
	events.SortByDate()

	return events, nil
}

// parseEvent parses a single event from the calendar.
func parseEvent(event mql5Calendar) (*EconomicCalendarEvent, error) {
	currency, err := parseCurrency(event)
	if err != nil {
		return nil, errlvl.Wrap(err, errlvl.ERROR)
	}

	country := parseCountry(event)

	impact, err := parseImpact(event)
	if err != nil {
		return nil, errlvl.Wrap(err, errlvl.ERROR)
	}

	// Parse dates
	dt, err := utils.ParseDate(event.FullDate)
	if err != nil {
		return nil, errlvl.Wrap(fmt.Errorf("error parsing date: %w, value %v", err, event.FullDate), errlvl.ERROR)
	}
	et, err := utils.ParseDate(event.ReleaseDate)
	if err != nil {
		return nil, errlvl.Wrap(fmt.Errorf("error parsing date: %w, value %v", err, event.ReleaseDate), errlvl.ERROR)
	}

	e := &EconomicCalendarEvent{
		DateTime:  dt,
		EventTime: et,
		Country:   country,
		Currency:  currency,
		Impact:    impact,
		Title:     event.EventName,
		Actual:    strings.ReplaceAll(strings.ToLower(event.ActualValue), "\u00a0", ""), // Remove nbsp symbol, convert to lowercase
		Forecast:  strings.ReplaceAll(strings.ToLower(event.ForecastValue), "\u00a0", ""),
		Previous:  strings.ReplaceAll(strings.ToLower(event.PreviousValue), "\u00a0", ""),
	}

	return e, nil
}

func parseImpact(event mql5Calendar) (EconomicCalendarImpact, error) {
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
		return "", errlvl.Wrap(fmt.Errorf("unknown impact: %s", event.Importance), errlvl.ERROR)
	}
	return impact, nil
}

func parseCountry(event mql5Calendar) EconomicCalendarCountry { //nolint:gocyclo
	// Parse country
	var country EconomicCalendarCountry
	switch event.Country {
	case 36:
		country = EconomicCalendarAustralia
	case 76:
		country = EconomicCalendarBrazil
	case 124:
		country = EconomicCalendarCanada
	case 156:
		country = EconomicCalendarChina
	case 999, 918:
		country = EconomicCalendarEuropeanUnion
	case 250:
		country = EconomicCalendarFrance
	case 276:
		country = EconomicCalendarGermany
	case 344:
		country = EconomicCalendarHongKong
	case 356:
		country = EconomicCalendarIndia
	case 380:
		country = EconomicCalendarItaly
	case 392:
		country = EconomicCalendarJapan
	case 484:
		country = EconomicCalendarMexico
	case 554:
		country = EconomicCalendarNewZealand
	case 578:
		country = EconomicCalendarNorway
	case 702:
		country = EconomicCalendarSingapore
	case 710:
		country = EconomicCalendarSouthAfrica
	case 410:
		country = EconomicCalendarSouthKorea
	case 724:
		country = EconomicCalendarSpain
	case 752:
		country = EconomicCalendarSweden
	case 756:
		country = EconomicCalendarSwitzerland
	case 826:
		country = EconomicCalendarUnitedKingdom
	case 840:
		country = EconomicCalendarUnitedStates
	default:
		country = ""
	}
	return country
}

func parseCurrency(event mql5Calendar) (EconomicCalendarCurrency, error) {
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
	case "ALL":
		currency = EconomicCalendarALL
	default:
		return "", errlvl.Wrap(fmt.Errorf("unknown currency: %s", event.CurrencyCode), errlvl.ERROR)
	}
	return currency, nil
}

// EconomicCalendarCurrency impacted currencies(economic markets) by the event.
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
	EconomicCalendarALL EconomicCalendarCurrency = "ALL" // All currencies
)

type EconomicCalendarCountry = string

const (
	EconomicCalendarAustralia     EconomicCalendarCountry = "Australia"
	EconomicCalendarBrazil        EconomicCalendarCountry = "Brazil"
	EconomicCalendarCanada        EconomicCalendarCountry = "Canada"
	EconomicCalendarChina         EconomicCalendarCountry = "China"
	EconomicCalendarEuropeanUnion EconomicCalendarCountry = "European Union"
	EconomicCalendarFrance        EconomicCalendarCountry = "France"
	EconomicCalendarGermany       EconomicCalendarCountry = "Germany"
	EconomicCalendarHongKong      EconomicCalendarCountry = "Hong Kong"
	EconomicCalendarIndia         EconomicCalendarCountry = "India"
	EconomicCalendarItaly         EconomicCalendarCountry = "Italy"
	EconomicCalendarJapan         EconomicCalendarCountry = "Japan"
	EconomicCalendarMexico        EconomicCalendarCountry = "Mexico"
	EconomicCalendarNewZealand    EconomicCalendarCountry = "New Zealand"
	EconomicCalendarNorway        EconomicCalendarCountry = "Norway"
	EconomicCalendarSingapore     EconomicCalendarCountry = "Singapore"
	EconomicCalendarSouthAfrica   EconomicCalendarCountry = "South Africa"
	EconomicCalendarSouthKorea    EconomicCalendarCountry = "South Korea"
	EconomicCalendarSpain         EconomicCalendarCountry = "Spain"
	EconomicCalendarSweden        EconomicCalendarCountry = "Sweden"
	EconomicCalendarSwitzerland   EconomicCalendarCountry = "Switzerland"
	EconomicCalendarUnitedKingdom EconomicCalendarCountry = "United Kingdom"
	EconomicCalendarUnitedStates  EconomicCalendarCountry = "United States"
)

// GetCountryHashtag returns the country hashtag for the specified country.
func GetCountryHashtag(country EconomicCalendarCountry) string {
	m := map[EconomicCalendarCountry]string{
		EconomicCalendarAustralia:     "australia",
		EconomicCalendarBrazil:        "brazil",
		EconomicCalendarCanada:        "canada",
		EconomicCalendarChina:         "china",
		EconomicCalendarEuropeanUnion: "europe",
		EconomicCalendarFrance:        "france",
		EconomicCalendarGermany:       "germany",
		EconomicCalendarHongKong:      "hongkong",
		EconomicCalendarIndia:         "india",
		EconomicCalendarItaly:         "italy",
		EconomicCalendarJapan:         "japan",
		EconomicCalendarMexico:        "mexico",
		EconomicCalendarNewZealand:    "newzealand",
		EconomicCalendarNorway:        "norway",
		EconomicCalendarSingapore:     "singapore",
		EconomicCalendarSouthAfrica:   "southafrica",
		EconomicCalendarSouthKorea:    "southkorea",
		EconomicCalendarSpain:         "spain",
		EconomicCalendarSweden:        "sweden",
		EconomicCalendarSwitzerland:   "switzerland",
		EconomicCalendarUnitedKingdom: "uk",
		EconomicCalendarUnitedStates:  "usa",
	}
	return m[country]
}

// GetCountryEmoji returns the country emoji for the specified country.
func GetCountryEmoji(country EconomicCalendarCountry) string {
	m := map[EconomicCalendarCountry]string{
		EconomicCalendarAustralia:     "🇦🇺",
		EconomicCalendarBrazil:        "🇧🇷",
		EconomicCalendarCanada:        "🇨🇦",
		EconomicCalendarChina:         "🇨🇳",
		EconomicCalendarEuropeanUnion: "🇪🇺",
		EconomicCalendarFrance:        "🇫🇷",
		EconomicCalendarGermany:       "🇩🇪",
		EconomicCalendarHongKong:      "🇭🇰",
		EconomicCalendarIndia:         "🇮🇳",
		EconomicCalendarItaly:         "🇮🇹",
		EconomicCalendarJapan:         "🇯🇵",
		EconomicCalendarMexico:        "🇲🇽",
		EconomicCalendarNewZealand:    "🇳🇿",
		EconomicCalendarNorway:        "🇳🇴",
		EconomicCalendarSingapore:     "🇸🇬",
		EconomicCalendarSouthAfrica:   "🇿🇦",
		EconomicCalendarSouthKorea:    "🇰🇷",
		EconomicCalendarSpain:         "🇪🇸",
		EconomicCalendarSweden:        "🇸🇪",
		EconomicCalendarSwitzerland:   "🇨🇭",
		EconomicCalendarUnitedKingdom: "🇬🇧",
		EconomicCalendarUnitedStates:  "🇺🇸",
	}
	return m[country]
}

// EconomicCalendarImpact impact of the event on the market (low, medium, high, holiday, none).
type EconomicCalendarImpact = string

const (
	EconomicCalendarImpactLow     EconomicCalendarImpact = "Low"      // Low impact event
	EconomicCalendarImpactMedium  EconomicCalendarImpact = "Medium"   // Medium impact event
	EconomicCalendarImpactHigh    EconomicCalendarImpact = "High"     // High impact event
	EconomicCalendarImpactHoliday EconomicCalendarImpact = "Holidays" // Holiday event
	EconomicCalendarImpactNone    EconomicCalendarImpact = "None"     // No impact event
)

// EconomicCalendarEvent is the struct for economics calendar event object.
type EconomicCalendarEvent struct {
	DateTime  time.Time                // Date of the event
	EventTime time.Time                // Time of the event (if available)
	Country   EconomicCalendarCountry  // Country of the event
	Currency  EconomicCalendarCurrency // Currency impacted by the event
	Impact    EconomicCalendarImpact   // Impact of the event on the market
	Title     string                   // Event title
	Actual    string                   // Actual value of the event (if available)
	Forecast  string                   // Forecasted value of the event (if available)
	Previous  string                   // Previous value of the event (if available)
}

// MQL5 calendar event object.
type mql5Calendar struct {
	ID               int         `json:"ID"`
	EventType        int         `json:"EventType"`
	TimeMode         int         `json:"TimeMode"`
	Processed        int         `json:"Processed"`
	URL              string      `json:"URL"`
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

// EconomicCalendarEvents is the slice of economics calendar events.
type EconomicCalendarEvents []*EconomicCalendarEvent

// FilterByDateRange filters events by date range, returns new slice.
func (e EconomicCalendarEvents) FilterByDateRange(from, to time.Time) EconomicCalendarEvents {
	var filtered EconomicCalendarEvents
	for _, v := range e {
		if v.DateTime.After(from) && v.DateTime.Before(to) {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

// SortByDate sorts events by date (ascending).
func (e EconomicCalendarEvents) SortByDate() {
	sort.Slice(e, func(i, j int) bool {
		return e[i].DateTime.Before(e[j].DateTime)
	})
}

// Distinct removes duplicates from the slice.
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

// HasActualEvents checks if there are any events with EconomicCalendarEvent.Actual values.
func (e EconomicCalendarEvents) HasActualEvents() bool {
	for _, v := range e {
		if v.Actual != "" {
			return true
		}
	}
	return false
}
