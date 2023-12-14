package scavenger

import "github.com/samgozman/fin-thread/scavenger/fxcalendar"

// Scavenger is the struct that fetches some custom data from defined sources.
// The Scavenger will hold all available sources and will fetch the data from them.
//
// It shouldn't be used as journalist.Journalist to get news. The main purpose of this struct is to
// fetch custom unstructured data for different purposes. For example to fetch info updates or parse calendar events.
type Scavenger struct {
	ForexCalendar *fxcalendar.ForexCalendar
}
