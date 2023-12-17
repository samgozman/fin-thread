package jobs

import (
	"github.com/samgozman/fin-thread/scavenger/ecal"
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
						Currency:  ecal.EconomicCalendarUSD,
						Impact:    ecal.EconomicCalendarImpactHigh,
						Title:     "CPI Announcement",
						Forecast:  "2.9%",
						Previous:  "2.8%",
					},
					{
						DateTime:  time.Date(2023, time.April, 10, 12, 0, 0, 0, time.UTC),
						EventTime: time.Date(2023, time.April, 10, 13, 0, 0, 0, time.UTC),
						Currency:  ecal.EconomicCalendarUSD,
						Impact:    ecal.EconomicCalendarImpactHigh,
						Title:     "Some other event",
						Forecast:  "2.9%",
						Previous:  "2.8%",
					},
					{
						DateTime:  time.Date(2023, time.April, 10, 12, 0, 0, 0, time.UTC),
						EventTime: time.Date(2023, time.April, 10, 14, 0, 0, 0, time.UTC),
						Currency:  ecal.EconomicCalendarEUR,
						Impact:    ecal.EconomicCalendarImpactHigh,
						Title:     "Some woke event",
						Forecast:  "1.3%",
						Previous:  "1.8%",
					},
					{
						DateTime:  time.Date(2023, time.April, 11, 12, 0, 0, 0, time.UTC),
						EventTime: time.Date(2023, time.April, 11, 12, 0, 0, 0, time.UTC),
						Currency:  ecal.EconomicCalendarUSD,
						Impact:    ecal.EconomicCalendarImpactHigh,
						Title:     "Inflation Announcement",
						Forecast:  "6.9%",
						Previous:  "6.8%",
					},
					{
						DateTime:  time.Date(2023, time.April, 11, 12, 0, 0, 0, time.UTC),
						EventTime: time.Date(2023, time.April, 11, 13, 0, 0, 0, time.UTC),
						Currency:  ecal.EconomicCalendarUSD,
						Impact:    ecal.EconomicCalendarImpactHigh,
						Title:     "Some other event",
						Forecast:  "1.0%",
					},
					{
						DateTime:  time.Date(2023, time.April, 12, 00, 0, 0, 0, time.UTC),
						EventTime: time.Date(2023, time.April, 12, 23, 59, 0, 0, time.UTC),
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
