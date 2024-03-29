package jobs

import (
	"context"
	"fmt"
	"github.com/avast/retry-go"
	"github.com/getsentry/sentry-go"
	"github.com/samgozman/fin-thread/archivist"
	"github.com/samgozman/fin-thread/internal/utils"
	"github.com/samgozman/fin-thread/publisher"
	"github.com/samgozman/fin-thread/scavenger/ecal"
	"log/slog"
	"math"
	"strings"
	"time"
)

// CalendarJob is the struct that will fetch calendar events and publish them to the channel.
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

// RunDailyCalendarJob creates events plan for the upcoming day and publishes them to the channel.
// It should be run every business day.
func (j *CalendarJob) RunDailyCalendarJob() JobFunc {
	return func() {
		_ = retry.Do(func() error {
			ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
			defer cancel()
			j.logger.Info("[calendar] Running daily plan")

			tx := sentry.StartTransaction(ctx, "RunDailyCalendarJob")
			tx.Op = "job-calendar"

			// Sentry performance monitoring
			hub := sentry.GetHubFromContext(ctx)
			if hub == nil {
				hub = sentry.CurrentHub().Clone()
				ctx = sentry.SetHubOnContext(ctx, hub)
			}

			defer tx.Finish()
			defer hub.Flush(2 * time.Second)
			defer hub.Recover(nil)

			// Create events plan for the current day
			from := time.Now().Truncate(24 * time.Hour)
			to := from.Add(23 * time.Hour).Add(59 * time.Minute).Add(59 * time.Second)
			span := tx.StartChild("EconomicCalendar.Fetch")
			events, err := j.calendarScavenger.Fetch(ctx, from, to)
			span.Finish()
			if err != nil {
				e := fmt.Errorf("[job-calendar] Error fetching events: %w", err)
				j.logger.Error(e.Error())
				utils.CaptureSentryException("calendarJobFetchError", hub, e)
				return e
			}
			hub.AddBreadcrumb(&sentry.Breadcrumb{
				Category: "successful",
				Message:  fmt.Sprintf("EconomicCalendar.Fetch returned %d events", len(events)),
				Level:    sentry.LevelInfo,
			}, nil)
			if len(events) == 0 {
				return nil
			}

			// Format events to the text
			m := formatDailyEvents(events)

			// Publish events to the channel
			span = tx.StartChild("TelegramPublisher.Publish")
			_, err = j.publisher.Publish(m)
			span.Finish()
			if err != nil {
				e := fmt.Errorf("[job-calendar] Error publishing events: %w", err)
				j.logger.Error(e.Error())
				utils.CaptureSentryException("calendarJobPublishError", hub, e)
				// Note: Unrecoverable error, because Telegram API often hangs up, but somehow publishes the message
				return retry.Unrecoverable(e) //nolint:wrapcheck
			}

			hub.AddBreadcrumb(&sentry.Breadcrumb{
				Category: "successful",
				Message:  "Calendar published successfully",
				Level:    sentry.LevelInfo,
			}, nil)

			mappedEvents := make([]*archivist.Event, 0, len(events))
			for _, e := range events {
				mappedEvents = append(mappedEvents, mapEventToDB(e, j.publisher.ChannelID, j.providerName))
			}

			span = tx.StartChild("Archivist.CreateEvents")
			err = j.archivist.Entities.Events.Create(ctx, mappedEvents)
			span.Finish()
			if err != nil {
				e := fmt.Errorf("[job-calendar] Error saving events: %w", err)
				j.logger.Error(e.Error())
				utils.CaptureSentryException("calendarJobSaveError", hub, e)
				return retry.Unrecoverable(e) //nolint:wrapcheck
			}

			hub.AddBreadcrumb(&sentry.Breadcrumb{
				Category: "successful",
				Message:  fmt.Sprintf("Events.Create saved %d events", len(mappedEvents)),
				Level:    sentry.LevelInfo,
			}, nil)

			return nil
		},
			retry.Attempts(5),
			retry.Delay(10*time.Minute),
		)
	}
}

// RunCalendarUpdatesJob fetches "Actual" values for today's events and publishes updates to the channel.
func (j *CalendarJob) RunCalendarUpdatesJob() JobFunc {
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()

		tx := sentry.StartTransaction(ctx, "RunCalendarUpdatesJob")
		tx.Op = "job-calendar-updates"

		// Sentry performance monitoring
		hub := sentry.GetHubFromContext(ctx)
		if hub == nil {
			hub = sentry.CurrentHub().Clone()
			ctx = sentry.SetHubOnContext(ctx, hub)
		}

		defer tx.Finish()
		defer hub.Flush(2 * time.Second)
		defer hub.Recover(nil)

		// Fetch eventsDB for today from the database
		span := tx.StartChild("Archivist.FindRecentEventsWithoutValue")
		eventsDB, err := j.archivist.Entities.Events.FindRecentEventsWithoutValue(ctx)
		span.Finish()
		if err != nil {
			e := fmt.Errorf("[job-calendar-updates] Error fetching eventsDB: %w", err)
			j.logger.Error(e.Error())
			utils.CaptureSentryException("calendarUpdatesJobFindRecentError", hub, e)
			return
		}
		hub.AddBreadcrumb(&sentry.Breadcrumb{
			Category: "successful",
			Message:  fmt.Sprintf("Archivist.FindRecentEventsWithoutValue returned %d eventsDB", len(eventsDB)),
			Level:    sentry.LevelInfo,
		}, nil)
		if len(eventsDB) == 0 {
			return
		}

		// Fetch eventsDB for today from the calendar
		span = tx.StartChild("EconomicCalendar.Fetch")
		from := time.Now().Truncate(24 * time.Hour)
		to := from.Add(23 * time.Hour).Add(59 * time.Minute).Add(59 * time.Second)
		calendarEvents, err := j.calendarScavenger.Fetch(ctx, from, to)
		span.Finish()
		if err != nil {
			e := fmt.Errorf("[job-calendar-updates] Error fetching events from provider: %w", err)
			j.logger.Error(e.Error())
			utils.CaptureSentryException("calendarUpdatesJobFetchError", hub, e)
			return
		}
		hub.AddBreadcrumb(&sentry.Breadcrumb{
			Category: "successful",
			Message:  fmt.Sprintf("EconomicCalendar.Fetch returned %d eventsDB", len(calendarEvents)),
			Level:    sentry.LevelInfo,
		}, nil)
		if len(calendarEvents) == 0 {
			return
		}
		if !calendarEvents.HasActualEvents() {
			hub.AddBreadcrumb(&sentry.Breadcrumb{
				Category: "debug",
				Message:  "EconomicCalendar.Fetch returned eventsDB without actual values",
				Level:    sentry.LevelDebug,
			}, nil)
			return
		}

		// Update eventsDB with actual values
		var updatedEventsDB []*archivist.Event
		for _, e := range eventsDB {
			for _, ce := range calendarEvents {
				if e.Country != ce.Country || e.Currency != ce.Currency || e.Title != ce.Title || ce.Actual == "" {
					continue
				}
				ev := &archivist.Event{
					ID:           e.ID,
					ChannelID:    e.ChannelID,
					ProviderName: e.ProviderName,
					DateTime:     e.DateTime,
					Country:      e.Country,
					Currency:     e.Currency,
					Impact:       e.Impact,
					Title:        e.Title,
					Forecast:     ce.Forecast,
					Previous:     ce.Previous,
					Actual:       ce.Actual,
					UpdatedAt:    time.Now(),
				}

				updatedEventsDB = append(updatedEventsDB, ev)
			}
		}

		// TODO: add update many method to archivist with transaction
		for _, event := range updatedEventsDB {
			span = tx.StartChild("Archivist.UpdateEvent")
			err = j.archivist.Entities.Events.Update(ctx, event)
			span.Finish()
			if err != nil {
				e := fmt.Errorf("[job-calendar-updates] Error updating event: %w", err)
				j.logger.Error(e.Error())
				utils.CaptureSentryException("calendarUpdatesJobUpdateEventError", hub, e)
				return
			}
		}

		hub.AddBreadcrumb(&sentry.Breadcrumb{
			Category: "successful",
			Message:  fmt.Sprintf("Events.Update updated %d events", len(updatedEventsDB)),
			Level:    sentry.LevelInfo,
		}, nil)

		// Group events by country
		eventsByCountry := make(map[ecal.EconomicCalendarCountry][]*archivist.Event)
		for _, e := range updatedEventsDB {
			eventsByCountry[e.Country] = append(eventsByCountry[e.Country], e)
		}

		// Publish eventsDB to the channel
		for country, events := range eventsByCountry {
			m := formatEventsUpdate(country, events)
			if m == "" {
				continue
			}

			span = tx.StartChild("TelegramPublisher.Publish")
			_, err := j.publisher.Publish(m)
			span.Finish()
			if err != nil {
				e := fmt.Errorf("[job-calendar-updates] Error publishing event: %w", err)
				j.logger.Error(e.Error())
				utils.CaptureSentryException("calendarUpdatesJobPublishError", hub, e)
				return
			}
		}

		hub.AddBreadcrumb(&sentry.Breadcrumb{
			Category: "successful",
			Message:  fmt.Sprintf("TelegramPublisher.Publish published %d events", len(eventsByCountry)),
			Level:    sentry.LevelInfo,
		}, nil)
	}
}

// formatDailyEvents formats events to the text for publishing to the telegram channel.
func formatDailyEvents(events ecal.EconomicCalendarEvents) string {
	// Handle empty events case
	if len(events) == 0 {
		return ""
	}

	var m strings.Builder

	// Build header
	m.WriteString("📅 Economic calendar for today\n\n")

	// Iterate through events
	for _, e := range events {
		// Add event
		country := ecal.GetCountryEmoji(e.Country)

		// Print holiday events without time
		if e.Impact == ecal.EconomicCalendarImpactHoliday {
			m.WriteString(fmt.Sprintf("%s %s\n", country, e.Title))
		} else {
			m.WriteString(fmt.Sprintf("%s %s %s", country, e.DateTime.Format("15:04"), e.Title))

			// Print forecast and previous values if they are not empty
			if e.Forecast != "" {
				m.WriteString(fmt.Sprintf(", forecast: %s", e.Forecast))
			}
			if e.Previous != "" {
				m.WriteString(fmt.Sprintf(", last: %s", e.Previous))
			}

			m.WriteString("\n")
		}
	}

	// Build footer
	m.WriteString("*Time is in UTC*\n#calendar #economy")

	return m.String()
}

func formatEventsUpdate(country ecal.EconomicCalendarCountry, events []*archivist.Event) string {
	// Handle nil event case
	if len(events) == 0 {
		return ""
	}

	// Initialize message string
	var m strings.Builder

	// Add country emoji and hashtag
	countryEmoji := ecal.GetCountryEmoji(country)
	countryHashtag := ecal.GetCountryHashtag(country)
	m.WriteString(fmt.Sprintf("%s #%s\n", countryEmoji, countryHashtag))

	// Iterate through events
	for i, event := range events {
		// Add new line between events
		if i > 0 {
			m.WriteString("\n")
		}

		// Add event
		m.WriteString(formatEvent(event))
	}

	return m.String()
}

func formatEvent(event *archivist.Event) string {
	var ev strings.Builder

	actualNumber := utils.StrValueToFloat(event.Actual)
	previousNumber := utils.StrValueToFloat(event.Previous)
	forecastNumber := utils.StrValueToFloat(event.Forecast)

	// Check for a change in actual value compared to previous value or forecast value
	if (event.Previous != "" && actualNumber != previousNumber) ||
		(event.Forecast != "" && actualNumber != forecastNumber) {
		if event.Impact == ecal.EconomicCalendarImpactHigh {
			ev.WriteString("🔥 ")
		} else {
			ev.WriteString("⚠️ ")
		}
	}

	// Add event title and actual value in bold
	ev.WriteString(fmt.Sprintf("%s: *%s*", event.Title, event.Actual))

	// For non-percentage events, add percentage change from previous value
	if event.Previous != "" && !strings.Contains(event.Previous, "%") {
		p := ((actualNumber / previousNumber) - 1) * 100

		if p != math.Inf(1) && p != math.Inf(-1) {
			if p > 0 {
				ev.WriteString(fmt.Sprintf(" (+%.2f%%)", p))
			} else {
				ev.WriteString(fmt.Sprintf(" (%.2f%%)", p))
			}
		}
	}

	// Print forecast and previous values if they are not empty
	if event.Forecast != "" {
		ev.WriteString(fmt.Sprintf(", forecast: %s", event.Forecast))
	}
	if event.Previous != "" {
		ev.WriteString(fmt.Sprintf(", last: %s", event.Previous))
	}

	return ev.String()
}

// mapEventToDB maps calendar event to the database event instance.
// One crucial thing is that we use actual date if event time is available.
// There is no need to store 2 event dates in the database.
func mapEventToDB(e *ecal.EconomicCalendarEvent, channelID, providerName string) *archivist.Event {
	// use actual date if event time is available
	var dt time.Time
	if e.EventTime.After(e.DateTime) {
		dt = e.EventTime
	} else {
		dt = e.DateTime
	}
	return &archivist.Event{
		ChannelID:    channelID,
		ProviderName: providerName,
		DateTime:     dt,
		Country:      e.Country,
		Currency:     e.Currency,
		Impact:       e.Impact,
		Title:        e.Title,
		Forecast:     e.Forecast,
		Previous:     e.Previous,
	}
}
