package jobs

import (
	"context"
	"errors"
	"fmt"
	"github.com/getsentry/sentry-go"
	"github.com/samgozman/fin-thread/archivist"
	"github.com/samgozman/fin-thread/publisher"
	"github.com/samgozman/fin-thread/scavenger/ecal"
	"log/slog"
	"time"
)

// CalendarJob is the struct that will fetch calendar events and publish them to the channel
type CalendarJob struct {
	calendarScavenger *ecal.EconomicCalendar       // calendar scavenger that will fetch calendar events
	publisher         *publisher.TelegramPublisher // publisher that will publish news to the channel
	archivist         *archivist.Archivist         // archivist that will save news to the database
	logger            *slog.Logger                 // special logger for the job
}

func NewCalendarJob(
	calendarScavenger *ecal.EconomicCalendar,
	publisher *publisher.TelegramPublisher,
	archivist *archivist.Archivist,
) *CalendarJob {
	return &CalendarJob{
		calendarScavenger: calendarScavenger,
		publisher:         publisher,
		archivist:         archivist,
		logger:            slog.Default(),
	}
}

// RunWeeklyCalendar creates events plan for the upcoming week and publishes them to the channel.
// It should be run once a week.
func (j *CalendarJob) RunWeeklyCalendar() JobFunc {
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		j.logger.Info("[calendar] Running weekly plan")

		tx := sentry.StartTransaction(ctx, "RunWeeklyCalendar")
		tx.Op = "job-calendar"

		// Sentry performance monitoring
		hub := sentry.GetHubFromContext(ctx)
		if hub == nil {
			hub = sentry.CurrentHub().Clone()
			ctx = sentry.SetHubOnContext(ctx, hub)
		}

		defer func() {
			tx.Finish()
			hub.Flush(2 * time.Second)
		}()

		// Create events plan for the upcoming week
		from := time.Now().Truncate(24 * time.Hour)
		to := from.Add(7 * 24 * time.Hour)
		span := tx.StartChild("EconomicCalendar.Fetch")
		events, err := j.calendarScavenger.Fetch(ctx, from, to)
		span.Finish()
		if err != nil {
			e := errors.New(fmt.Sprintf("[job-calendar] Error fetching events: %v", err))
			j.logger.Error(e.Error())
			hub.CaptureException(e)
			return
		}

		// Format events to the text
		m := formatWeeklyEvents(events)

		// Publish events to the channel
		span = tx.StartChild("TelegramPublisher.Publish")
		_, err = j.publisher.Publish(m)
		span.Finish()
		if err != nil {
			e := errors.New(fmt.Sprintf("[job-calendar] Error publishing events: %v", err))
			j.logger.Error(e.Error())
			hub.CaptureException(e)
			return
		}

		// TODO: save events to the database
	}
}

// formatWeeklyEvents formats events to the text for publishing to the telegram channel
func formatWeeklyEvents(events []*ecal.EconomicCalendarEvent) string {
	// Group by date
	ge := make(map[string][]*ecal.EconomicCalendarEvent)
	for _, e := range events {
		ge[e.DateTime.Format("2006-01-02")] = append(ge[e.DateTime.Format("2006-01-02")], e)
	}

	var m string
	for k, v := range ge {
		s := fmt.Sprintf("**%s**\n", k)
		for _, e := range v {
			country := ecal.EconomicCalendarCountryEmoji[e.Currency]
			// Print holiday events without time
			if e.Impact == ecal.EconomicCalendarImpactHoliday {
				s += fmt.Sprintf("%s %s", country, e.Title)
				continue
			}
			s += fmt.Sprintf("%s %s %s", country, e.DateTime.Format("15:04"), e.Title)

			// Print forecast and previous values if they are not empty
			if e.Forecast != "" {
				s += fmt.Sprintf(", forecast: %s", e.Forecast)
			}
			if e.Previous != "" {
				s += fmt.Sprintf(", last: %s", e.Previous)
			}

			s += "\n"
		}
		m += s
	}
	if m == "" {
		return ""
	}

	header := "📅 Economic calendar for the upcoming week\n"
	footer := "\n*All times are in UTC*\n#calendar #economy"
	return header + m + footer
}
