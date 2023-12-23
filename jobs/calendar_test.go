package jobs

import (
	"github.com/samgozman/fin-thread/archivist/models"
	"github.com/samgozman/fin-thread/scavenger/ecal"
	"reflect"
	"testing"
	"time"
)

func Test_formatWeeklyEvents(t *testing.T) {
	type args struct {
		// Note: events should be sorted by date in ascending order
		events ecal.EconomicCalendarEvents
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "case many weekly events",
			args: args{
				events: ecal.EconomicCalendarEvents{
					{
						DateTime:  time.Date(2023, time.April, 10, 12, 0, 0, 0, time.UTC),
						EventTime: time.Date(2023, time.April, 10, 12, 0, 0, 0, time.UTC),
						Country:   ecal.EconomicCalendarUnitedStates,
						Currency:  ecal.EconomicCalendarUSD,
						Impact:    ecal.EconomicCalendarImpactHigh,
						Title:     "CPI Announcement",
						Forecast:  "2.9%",
						Previous:  "2.8%",
					},
					{
						DateTime:  time.Date(2023, time.April, 10, 12, 0, 0, 0, time.UTC),
						EventTime: time.Date(2023, time.April, 10, 13, 0, 0, 0, time.UTC),
						Country:   ecal.EconomicCalendarUnitedStates,
						Currency:  ecal.EconomicCalendarUSD,
						Impact:    ecal.EconomicCalendarImpactHigh,
						Title:     "Some other event",
						Forecast:  "2.9%",
						Previous:  "2.8%",
					},
					{
						DateTime:  time.Date(2023, time.April, 10, 12, 0, 0, 0, time.UTC),
						EventTime: time.Date(2023, time.April, 10, 14, 0, 0, 0, time.UTC),
						Country:   ecal.EconomicCalendarEuropeanUnion,
						Currency:  ecal.EconomicCalendarEUR,
						Impact:    ecal.EconomicCalendarImpactHigh,
						Title:     "Some woke event",
						Forecast:  "1.3%",
						Previous:  "1.8%",
					},
					{
						DateTime:  time.Date(2023, time.April, 11, 12, 0, 0, 0, time.UTC),
						EventTime: time.Date(2023, time.April, 11, 12, 0, 0, 0, time.UTC),
						Country:   ecal.EconomicCalendarUnitedStates,
						Currency:  ecal.EconomicCalendarUSD,
						Impact:    ecal.EconomicCalendarImpactHigh,
						Title:     "Inflation Announcement",
						Forecast:  "6.9%",
						Previous:  "6.8%",
					},
					{
						DateTime:  time.Date(2023, time.April, 11, 12, 0, 0, 0, time.UTC),
						EventTime: time.Date(2023, time.April, 11, 13, 0, 0, 0, time.UTC),
						Country:   ecal.EconomicCalendarUnitedStates,
						Currency:  ecal.EconomicCalendarUSD,
						Impact:    ecal.EconomicCalendarImpactHigh,
						Title:     "Some other event",
						Forecast:  "1.0%",
					},
					{
						DateTime:  time.Date(2023, time.April, 12, 00, 0, 0, 0, time.UTC),
						EventTime: time.Date(2023, time.April, 12, 23, 59, 0, 0, time.UTC),
						Country:   ecal.EconomicCalendarUnitedStates,
						Currency:  ecal.EconomicCalendarUSD,
						Impact:    ecal.EconomicCalendarImpactHoliday,
						Title:     "Some holiday",
						Forecast:  "",
						Previous:  "",
					},
				},
			},
			want: "📅 Economic calendar for the upcoming week\n\n" +
				"*Monday, April 10*\n" +
				"🇺🇸 12:00 CPI Announcement, forecast: 2.9%, last: 2.8%\n" +
				"🇺🇸 12:00 Some other event, forecast: 2.9%, last: 2.8%\n" +
				"🇪🇺 12:00 Some woke event, forecast: 1.3%, last: 1.8%\n" +
				"*Tuesday, April 11*\n" +
				"🇺🇸 12:00 Inflation Announcement, forecast: 6.9%, last: 6.8%\n" +
				"🇺🇸 12:00 Some other event, forecast: 1.0%\n" +
				"*Wednesday, April 12*\n" +
				"🇺🇸 Some holiday\n" +
				"*All times are in UTC*\n" +
				"#calendar #economy",
		},
		{
			name: "case none events",
			args: args{
				events: ecal.EconomicCalendarEvents{},
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatWeeklyEvents(tt.args.events)
			if got != tt.want {
				t.Errorf("formatWeeklyEvents() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_mapEventToDB(t *testing.T) {
	type args struct {
		e            *ecal.EconomicCalendarEvent
		channelID    string
		providerName string
	}
	tests := []struct {
		name string
		args args
		want *models.Event
	}{
		{
			name: "case 1 - event time is after event date",
			args: args{
				e: &ecal.EconomicCalendarEvent{
					DateTime:  time.Date(2023, time.April, 10, 12, 0, 0, 0, time.UTC),
					EventTime: time.Date(2023, time.April, 10, 13, 0, 0, 0, time.UTC),
					Currency:  ecal.EconomicCalendarUSD,
					Impact:    ecal.EconomicCalendarImpactHigh,
					Title:     "CPI Announcement",
					Forecast:  "2.9%",
					Previous:  "2.8%",
				},
				channelID:    "channel-id",
				providerName: "provider-name",
			},
			want: &models.Event{
				ChannelID:    "channel-id",
				ProviderName: "provider-name",
				DateTime:     time.Date(2023, time.April, 10, 13, 0, 0, 0, time.UTC),
				Currency:     ecal.EconomicCalendarUSD,
				Impact:       ecal.EconomicCalendarImpactHigh,
				Title:        "CPI Announcement",
				Forecast:     "2.9%",
				Previous:     "2.8%",
			},
		},
		{
			name: "case 2 - event time is before event date",
			args: args{
				e: &ecal.EconomicCalendarEvent{
					DateTime:  time.Date(2023, time.April, 10, 12, 0, 0, 0, time.UTC),
					EventTime: time.Date(2023, time.April, 10, 11, 0, 0, 0, time.UTC),
					Currency:  ecal.EconomicCalendarUSD,
					Impact:    ecal.EconomicCalendarImpactHigh,
					Title:     "CPI Announcement",
					Forecast:  "2.9%",
					Previous:  "2.8%",
				},
				channelID:    "channel-id",
				providerName: "provider-name",
			},
			want: &models.Event{
				ChannelID:    "channel-id",
				ProviderName: "provider-name",
				DateTime:     time.Date(2023, time.April, 10, 12, 0, 0, 0, time.UTC),
				Currency:     ecal.EconomicCalendarUSD,
				Impact:       ecal.EconomicCalendarImpactHigh,
				Title:        "CPI Announcement",
				Forecast:     "2.9%",
				Previous:     "2.8%",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mapEventToDB(tt.args.e, tt.args.channelID, tt.args.providerName); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("mapEventToDB() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_formatEventUpdate(t *testing.T) {
	type args struct {
		event *models.Event
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "case 1 - event with previous value",
			args: args{
				event: &models.Event{
					DateTime: time.Date(2023, time.April, 10, 12, 0, 0, 0, time.UTC),
					Country:  ecal.EconomicCalendarUnitedStates,
					Currency: ecal.EconomicCalendarUSD,
					Impact:   ecal.EconomicCalendarImpactHigh,
					Title:    "CPI Announcement",
					Actual:   "2.9%",
					Forecast: "2.9%",
					Previous: "2.8%",
				},
			},
			want: "🔥🇺🇸 #usa\nCPI Announcement: *2.9%*, forecast: 2.9%, last: 2.8%",
		},
		{
			name: "case 2 - event without previous value or forecast",
			args: args{
				event: &models.Event{
					DateTime: time.Date(2023, time.April, 10, 12, 0, 0, 0, time.UTC),
					Country:  ecal.EconomicCalendarEuropeanUnion,
					Currency: ecal.EconomicCalendarEUR,
					Impact:   ecal.EconomicCalendarImpactHigh,
					Title:    "EU is strongly concerned score",
					Actual:   "1.3%",
				},
			},
			want: "🇪🇺 #europe\nEU is strongly concerned score: *1.3%*",
		},
		{
			name: "case 3 - with multiplier",
			args: args{
				event: &models.Event{
					DateTime: time.Date(2023, time.April, 10, 12, 0, 0, 0, time.UTC),
					Country:  ecal.EconomicCalendarUnitedStates,
					Currency: ecal.EconomicCalendarUSD,
					Impact:   ecal.EconomicCalendarImpactHigh,
					Title:    "Home sales",
					Actual:   "4.5 M",
					Forecast: "4.25 M",
					Previous: "4.0 M",
				},
			},
			want: "🔥🇺🇸 #usa\nHome sales: *4.5 M* (+12.50%), forecast: 4.25 M, last: 4.0 M",
		},
		{
			name: "case 4 - with multiplier and negative value",
			args: args{
				event: &models.Event{
					DateTime: time.Date(2023, time.April, 10, 12, 0, 0, 0, time.UTC),
					Country:  ecal.EconomicCalendarUnitedStates,
					Currency: ecal.EconomicCalendarUSD,
					Impact:   ecal.EconomicCalendarImpactHigh,
					Title:    "Home sales",
					Actual:   "4.0 M",
					Forecast: "4.25 M",
					Previous: "4.5 M",
				},
			},
			want: "🔥🇺🇸 #usa\nHome sales: *4.0 M* (-11.11%), forecast: 4.25 M, last: 4.5 M",
		},
		{
			name: "case 5 - with zero values",
			args: args{
				event: &models.Event{
					DateTime: time.Date(2023, time.April, 10, 12, 0, 0, 0, time.UTC),
					Country:  ecal.EconomicCalendarUnitedStates,
					Currency: ecal.EconomicCalendarUSD,
					Impact:   ecal.EconomicCalendarImpactHigh,
					Title:    "Home sales",
					Actual:   "2 M",
					Forecast: "1 M",
					Previous: "0 M",
				},
			},
			want: "🔥🇺🇸 #usa\nHome sales: *2 M*, forecast: 1 M, last: 0 M",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatEventUpdate(tt.args.event); got != tt.want {
				t.Errorf("formatEventUpdate() = %v, want %v", got, tt.want)
			}
		})
	}
}
