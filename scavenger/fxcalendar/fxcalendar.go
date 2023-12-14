package fxcalendar

import "time"

// ForexCalendar is the struct that fetches the calendar events from the ForexFactory website.
type ForexCalendar struct {
	Url        string                  // URL of the calendar page (e.g. https://www.forexfactory.com/calendar)
	Currencies []ForexCalendarCurrency // Currencies to filter the events by
	Impacts    []ForexCalendarImpact   // Impacts to filter the events by
}

// NewForexCalendar creates a new ForexCalendar instance
func NewForexCalendar(
	url string,
	currencies []ForexCalendarCurrency,
	impacts []ForexCalendarImpact,
) *ForexCalendar {
	return &ForexCalendar{
		Url:        url,
		Currencies: currencies,
		Impacts:    impacts,
	}
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
)

// ForexCalendarImpact impact of the event on the market
type ForexCalendarImpact = string

const (
	ForexCalendarImpactLow    ForexCalendarImpact = "Low"    // Low impact event
	ForexCalendarImpactMedium ForexCalendarImpact = "Medium" // Medium impact event
	ForexCalendarImpactHigh   ForexCalendarImpact = "High"   // High impact event
	ForexCalendarImpactNone   ForexCalendarImpact = "None"   // No economic event
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
