package ecal

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func Test_parseEvent(t *testing.T) {
	tests := []struct {
		name    string
		event   mql5Calendar
		want    *EconomicCalendarEvent
		wantErr bool
	}{
		{
			name: "case 1 - regular event",
			event: mql5Calendar{
				ActualValue:   "0.2%",
				CurrencyCode:  "USD",
				Country:       840,
				ForecastValue: "0.2\u00A0%",
				Importance:    "high",
				PreviousValue: "0.3\u00A0%",
				EventName:     "Core CPI m/m",
				FullDate:      "2023-11-13T12:58:48",
				ReleaseDate:   1702450800000,
			},
			want: &EconomicCalendarEvent{
				Actual:    "0.2%",
				Currency:  EconomicCalendarUSD,
				Country:   EconomicCalendarUnitedStates,
				DateTime:  time.Date(2023, 11, 13, 12, 58, 48, 0, time.UTC),
				EventTime: time.Date(2023, 12, 13, 07, 00, 00, 0, time.UTC),
				Forecast:  "0.2%",
				Impact:    EconomicCalendarImpactHigh,
				Previous:  "0.3%",
				Title:     "Core CPI m/m",
			},
			wantErr: false,
		},
		{
			name: "case 2 - holiday",
			event: mql5Calendar{
				ActualValue:   "",
				CurrencyCode:  "EUR",
				Country:       999,
				ForecastValue: "",
				Importance:    "none",
				EventType:     2,
				PreviousValue: "",
				EventName:     "The Day of Flying Spaghetti Monster",
				FullDate:      "2023-11-13T12:58:48",
				ReleaseDate:   0,
			},
			want: &EconomicCalendarEvent{
				Actual:    "",
				Currency:  EconomicCalendarEUR,
				Country:   EconomicCalendarEuropeanUnion,
				DateTime:  time.Date(2023, 11, 13, 12, 58, 48, 0, time.UTC),
				EventTime: time.Time{},
				Forecast:  "",
				Impact:    EconomicCalendarImpactHoliday,
				Previous:  "",
				Title:     "The Day of Flying Spaghetti Monster",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseEvent(tt.event)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseEvent() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEconomicCalendar_Fetch(t *testing.T) {
	type args struct {
		from time.Time
		to   time.Time
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// Just a stupid test to check if the fetch works and returns something without errors
		{
			name: "fetch for today",
			args: args{
				from: time.Now().Truncate(24 * time.Hour),
				to:   time.Now().Truncate(24 * time.Hour).Add(24 * time.Hour),
			},
			wantErr: false,
		},
		{
			name: "error if from is after to",
			args: args{
				from: time.Now().Add(24 * time.Hour),
				to:   time.Now(),
			},
			wantErr: true,
		},
		{
			name: "error if to is more than 7 days after from",
			args: args{
				from: time.Now().Add(-24 * time.Hour),
				to:   time.Now().Add(8 * 24 * time.Hour),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &EconomicCalendar{}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			got, err := c.Fetch(ctx, tt.args.from, tt.args.to)
			if (err != nil) != tt.wantErr {
				t.Errorf("Fetch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// NOTE: len can be 0 if there are no events for the day with the current filter
			// TODO: Should split the actual fetch in future to test thing properly
			if !tt.wantErr && len(got) > 1 {
				// check that first event if before the last one
				i := len(got) - 1
				if got[0].DateTime != got[i].DateTime && got[0].DateTime.After(got[i].DateTime) {
					t.Errorf("Fetch() got invalid events order. First %s, Last %s", got[0].DateTime, got[i].DateTime)
				}
			}
		})
	}
}
