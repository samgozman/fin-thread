package archivist

import (
	"context"
	"github.com/google/uuid"
	"github.com/samgozman/fin-thread/composer"
	"github.com/samgozman/fin-thread/pkg/errlvl"
	"github.com/samgozman/fin-thread/scavenger/ecal"
	"gorm.io/gorm"
	"time"
)

type EventsDB struct {
	Conn *gorm.DB
}

func NewEventsDB(db *gorm.DB) *EventsDB {
	return &EventsDB{
		Conn: db,
	}
}

type Event struct {
	ID           uuid.UUID                     `gorm:"primaryKey;type:uuid;not null;" json:"id"` // ID of the event (UUID)
	ChannelID    string                        `gorm:"size:64" json:"channel_id"`                // ID of the channel (chat ID in Telegram)
	ProviderName string                        `gorm:"size:64" json:"provider_name"`             // Name of the provider (e.g. "mql5")
	Title        string                        `gorm:"size:256" json:"title"`                    // Event title
	DateTime     time.Time                     `gorm:"not null" json:"date_time"`                // Event date and time
	Country      ecal.EconomicCalendarCountry  `gorm:"size:32" json:"country"`                   // Country of the event
	Currency     ecal.EconomicCalendarCurrency `gorm:"size:10" json:"currency"`                  // Currency impacted by the event
	Impact       ecal.EconomicCalendarImpact   `gorm:"size:10" json:"impact"`                    // Impact of the event on the market
	Actual       string                        `gorm:"size:64" json:"actual"`                    // Actual value of the event (if available)
	Forecast     string                        `gorm:"size:64" json:"forecast"`                  // Forecasted value of the event (if available)
	Previous     string                        `gorm:"size:64" json:"previous"`                  // Previous value of the event (if available)
	CreatedAt    time.Time                     `gorm:"default:CURRENT_TIMESTAMP" json:"created_at,omitempty"`
	UpdatedAt    time.Time                     `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at,omitempty"`
}

func (e *Event) Validate() error {
	if len(e.ChannelID) > 64 {
		return newError(errlvl.INFO, errChannelIDTooLong)
	}

	if len(e.ProviderName) > 64 {
		return newError(errlvl.INFO, errProviderNameTooLong)
	}

	if len(e.Title) > 256 {
		return newError(errlvl.INFO, errTitleTooLong)
	}

	return nil
}

func (e *Event) BeforeCreate(_ *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}

	if err := e.Validate(); err != nil {
		return newError(errlvl.INFO, errEventValidation, err)
	}

	return nil
}

func (e *Event) BeforeUpdate(_ *gorm.DB) error {
	if err := e.Validate(); err != nil {
		return newError(errlvl.INFO, errEventValidation, err)
	}

	return nil
}

func (e *Event) ToHeadline() *composer.Headline {
	return &composer.Headline{
		ID:   e.ID.String(),
		Text: e.Title,
		// TODO: Publication link?
	}
}

func (edb *EventsDB) Create(ctx context.Context, e []*Event) error {
	res := edb.Conn.WithContext(ctx).Create(e)
	if res.Error != nil {
		return newError(errlvl.ERROR, errEventCreation, res.Error)
	}

	return nil
}

func (edb *EventsDB) Update(ctx context.Context, e *Event) error {
	res := edb.Conn.WithContext(ctx).Where("id = ?", e.ID).Updates(e)
	if res.Error != nil {
		return newError(errlvl.ERROR, errEventUpdate, res.Error)
	}

	return nil
}

// FindRecentEventsWithoutValue finds events without Event.Actual value from the start of the day.
// Also, it filters out events with Event.Impact = None and Event.Impact = Holiday (e.g. no impact events).
func (edb *EventsDB) FindRecentEventsWithoutValue(ctx context.Context) ([]*Event, error) {
	var events []*Event
	res := edb.
		Conn.
		WithContext(ctx).
		Where("date_time >= ?", time.Now().UTC().Truncate(24*time.Hour)).
		Where("impact NOT IN ?", []ecal.EconomicCalendarImpact{ecal.EconomicCalendarImpactNone, ecal.EconomicCalendarImpactHoliday}).
		Where("actual = ?", "").
		Find(&events)

	if res.Error != nil {
		return nil, newError(errlvl.ERROR, errFindRecentEvents, res.Error)
	}

	return events, nil
}

// FindAllUntilDate finds all events between time.Now until the provided date.
func (edb *EventsDB) FindAllUntilDate(ctx context.Context, until time.Time) ([]*Event, error) {
	var events []*Event
	// Where date_time is between now and until and actual is not empty
	res := edb.Conn.WithContext(ctx).
		Where("date_time BETWEEN ? AND ?", until, time.Now()).
		Where("actual != ?", "").
		Find(&events)
	if res.Error != nil {
		return nil, newError(errlvl.ERROR, errFindUntilEvents, res.Error)
	}

	return events, nil
}
