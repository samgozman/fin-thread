package jobs

import (
	"context"
	"fmt"
	"github.com/avast/retry-go"
	"github.com/getsentry/sentry-go"
	"github.com/samgozman/fin-thread/archivist"
	"github.com/samgozman/fin-thread/archivist/models"
	"github.com/samgozman/fin-thread/publisher"
	"github.com/samgozman/fin-thread/scavenger/ecal"
	"github.com/samgozman/fin-thread/utils"
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
	options           *CalendarJobOptions          // options for the job
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
		options:           &CalendarJobOptions{},
	}
}

// Publish will set the job to publish events to the channel.
func (j *CalendarJob) Publish() *CalendarJob {
	j.options.shouldPublish = true
	return j
}

// CalendarJobOptions is the struct that contains options for the calendar job.
type CalendarJobOptions struct {
	shouldPublish bool // if true, will publish news to the channel. Else: will just print them to the console (for development)
}

// RunWeeklyCalendarJob creates events plan for the upcoming week and publishes them to the channel.
// It should be run once a week on Monday.
func (j *CalendarJob) RunWeeklyCalendarJob() JobFunc {
	return func() {
		_ = retry.Do(func() error {
			ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
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
			m := formatWeeklyEvents(events)

			if j.options.shouldPublish {
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
			} else {
				fmt.Println(m)
			}

			mappedEvents := make([]*models.Event, 0, len(events))
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

		defer func() {
			tx.Finish()
			hub.Flush(2 * time.Second)
		}()

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
		var updatedEventsDB []*models.Event
		for _, e := range eventsDB {
			for _, ce := range calendarEvents {
				if e.Country != ce.Country || e.Currency != ce.Currency || e.Title != ce.Title || ce.Actual == "" {
					continue
				}
				ev := &models.Event{
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

		// Group events by country
		eventsByCountry := make(map[ecal.EconomicCalendarCountry][]*models.Event)
		for _, e := range updatedEventsDB {
			eventsByCountry[e.Country] = append(eventsByCountry[e.Country], e)
		}

		// Publish eventsDB to the channel
		for country, events := range eventsByCountry {
			m := formatEventUpdate(country, events)
			if m == "" {
				continue
			}

			if !j.options.shouldPublish {
				fmt.Println(m)
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
	}
}

// formatWeeklyEvents formats events to the text for publishing to the telegram channel.
func formatWeeklyEvents(events ecal.EconomicCalendarEvents) string {
	// Handle empty events case
	if len(events) == 0 {
		return ""
	}

	var m strings.Builder

	// Build header
	m.WriteString("📅 Economic calendar for the upcoming week\n\n")

	latestDateStr := ""
	// Iterate through events
	for _, e := range events {
		// Add events group date
		dt := e.DateTime.Format("Monday, January 02")
		if dt != latestDateStr {
			latestDateStr = dt
			m.WriteString(fmt.Sprintf("*%s*\n", dt))
		}

		// Add event
		country := ecal.EconomicCalendarCountryEmoji[e.Country]

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
	m.WriteString("*All times are in UTC*\n#calendar #economy")

	return m.String()
}

func formatEventUpdate(country ecal.EconomicCalendarCountry, events []*models.Event) string {
	// Handle nil event case
	if len(events) == 0 {
		return ""
	}

	// Initialize message string
	var m strings.Builder

	// Add country emoji and hashtag
	countryEmoji := ecal.EconomicCalendarCountryEmoji[country]
	countryHashtag := ecal.EconomicCalendarCountryHashtag[country]
	m.WriteString(fmt.Sprintf("%s #%s\n", countryEmoji, countryHashtag))

	// Iterate through events
	for i, event := range events {
		// Add new line between events
		if i > 0 {
			m.WriteString("\n")
		}

		// Check if the event has a previous value or a forecast value
		if event.Previous != "" || event.Forecast != "" {
			actualNumber := utils.StrValueToFloat(event.Actual)

			// Check for a change in actual value compared to previous value or forecast value
			previousNumber := utils.StrValueToFloat(event.Previous)
			forecastNumber := utils.StrValueToFloat(event.Forecast)

			if (event.Previous != "" && actualNumber != previousNumber) ||
				(event.Forecast != "" && actualNumber != forecastNumber) {
				if event.Impact == ecal.EconomicCalendarImpactHigh {
					m.WriteString("🔥 ")
				} else {
					m.WriteString("⚠️ ")
				}
			}
		}

		// Add event title and actual value in bold
		m.WriteString(fmt.Sprintf("%s: *%s*", event.Title, event.Actual))

		// For non-percentage events, add percentage change from previous value
		if event.Previous != "" && !strings.Contains(event.Previous, "%") {
			actualNumber := utils.StrValueToFloat(event.Actual)
			previousNumber := utils.StrValueToFloat(event.Previous)
			p := ((actualNumber / previousNumber) - 1) * 100

			if p != math.Inf(1) && p != math.Inf(-1) {
				if p > 0 {
					m.WriteString(fmt.Sprintf(" (+%.2f%%)", p))
				} else {
					m.WriteString(fmt.Sprintf(" (%.2f%%)", p))
				}
			}
		}

		// Print forecast and previous values if they are not empty
		if event.Forecast != "" {
			m.WriteString(fmt.Sprintf(", forecast: %s", event.Forecast))
		}
		if event.Previous != "" {
			m.WriteString(fmt.Sprintf(", last: %s", event.Previous))
		}
	}

	return m.String()
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
		Country:      e.Country,
		Currency:     e.Currency,
		Impact:       e.Impact,
		Title:        e.Title,
		Forecast:     e.Forecast,
		Previous:     e.Previous,
	}
}
