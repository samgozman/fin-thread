package jobs

import (
	"context"
	"errors"
	"fmt"
	"github.com/getsentry/sentry-go"
	"github.com/samgozman/fin-thread/archivist"
	"github.com/samgozman/fin-thread/archivist/models"
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
	providerName      string                       // name of the job provider
}

func NewCalendarJob(
	calendarScavenger *ecal.EconomicCalendar,
	publisher *publisher.TelegramPublisher,
	archivist *archivist.Archivist,
	providerName string,
) *CalendarJob {
	return &CalendarJob{
		calendarScavenger: calendarScavenger,
		publisher:         publisher,
		archivist:         archivist,
		logger:            slog.Default(),
		providerName:      providerName,
	}
}

// RunWeeklyCalendarJob creates events plan for the upcoming week and publishes them to the channel.
// It should be run once a week on Monday.
func (j *CalendarJob) RunWeeklyCalendarJob() JobFunc {
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		j.logger.Info("[calendar] Running weekly plan")

		tx := sentry.StartTransaction(ctx, "RunWeeklyCalendarJob")
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
		// from should be always a Monday of the current week
		from := time.Now().Truncate(24 * time.Hour).Add(-time.Duration(time.Now().Weekday()-time.Monday) * 24 * time.Hour)
		to := from.Add(6 * 24 * time.Hour).Add(23 * time.Hour).Add(59 * time.Minute).Add(59 * time.Second)
		span := tx.StartChild("EconomicCalendar.Fetch")
		events, err := j.calendarScavenger.Fetch(ctx, from, to)
		span.Finish()
		if err != nil {
			e := errors.New(fmt.Sprintf("[job-calendar] Error fetching events: %v", err))
			j.logger.Error(e.Error())
			hub.CaptureException(e)
			return
		}
		hub.AddBreadcrumb(&sentry.Breadcrumb{
			Category: "successful",
			Message:  fmt.Sprintf("EconomicCalendar.Fetch returned %d events", len(events)),
			Level:    sentry.LevelInfo,
		}, nil)
		if len(events) == 0 {
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

		// TODO: add create many method to archivist with transaction
		for _, e := range events {
			ev := mapEventToDB(e, j.publisher.ChannelID, j.providerName)

			span = tx.StartChild("Archivist.CreateEvent")
			err := j.archivist.Entities.Events.Create(ctx, ev)
			span.Finish()
			if err != nil {
				e := errors.New(fmt.Sprintf("[job-calendar] Error saving event: %v", err))
				j.logger.Error(e.Error())
				hub.CaptureException(e)
				return
			}
		}
	}
}

// formatWeeklyEvents formats events to the text for publishing to the telegram channel
func formatWeeklyEvents(events ecal.EconomicCalendarEvents) string {
	if len(events) == 0 {
		return ""
	}

	var m string
	latestDateStr := ""
	for _, e := range events {
		// Add events group date
		dt := e.DateTime.Format("Monday, January 02")
		if dt != latestDateStr {
			latestDateStr = dt
			m += fmt.Sprintf("*%s*\n", dt)
		}

		// Add event
		country := ecal.EconomicCalendarCountryEmoji[e.Currency]
		// Print holiday events without time
		if e.Impact == ecal.EconomicCalendarImpactHoliday {
			m += fmt.Sprintf("%s %s\n", country, e.Title)
			continue
		}
		m += fmt.Sprintf("%s %s %s", country, e.DateTime.Format("15:04"), e.Title)

		// Print forecast and previous values if they are not empty
		if e.Forecast != "" {
			m += fmt.Sprintf(", forecast: %s", e.Forecast)
		}
		if e.Previous != "" {
			m += fmt.Sprintf(", last: %s", e.Previous)
		}

		m += "\n"
	}

	header := "📅 Economic calendar for the upcoming week\n\n"
	footer := "*All times are in UTC*\n#calendar #economy"
	return header + m + footer
}

// mapEventToDB maps calendar event to the database event instance.
// One crucial thing is that we use actual date if event time is available.
// There is no need to store 2 event dates in the database.
func mapEventToDB(e *ecal.EconomicCalendarEvent, channelID, providerName string) *models.Event {
	// use actual date if event time is available
	var dt time.Time
	if e.EventTime.After(e.DateTime) {
		dt = e.EventTime
	} else {
		dt = e.DateTime
	}
	return &models.Event{
		ChannelID:    channelID,
		ProviderName: providerName,
		DateTime:     dt,
		Currency:     e.Currency,
		Impact:       e.Impact,
		Title:        e.Title,
		Forecast:     e.Forecast,
		Previous:     e.Previous,
	}
}
