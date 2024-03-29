package archivist

import (
	"github.com/samgozman/fin-thread/pkg/errlvl"
	"gorm.io/gorm"
)

// entities is a struct that contains all the entities that Archivist is responsible for.
type entities struct {
	News   *NewsDB
	Events *EventsDB
}

// Archivist is responsible for storing and retrieving data from the database.
type Archivist struct {
	db       *gorm.DB
	Entities *entities
}

// NewArchivist creates a new Archivist with provided DSN to connect to database.
//
// DSN is a string in the format of: "user=gorm password=gorm dbname=gorm port=9920 sslmode=disable".
func NewArchivist(dsn string) (*Archivist, error) {
	conn, err := connectToPG(dsn)
	if err != nil {
		return nil, err
	}

	// Migrate the schema automatically for now.
	// TODO: Add migration tool later.
	err = conn.AutoMigrate(&News{}, &Event{})
	if err != nil {
		return nil, newError(errlvl.FATAL, errFailedMigration, err)
	}

	return &Archivist{
		db: conn,
		Entities: &entities{
			News:   NewNewsDB(conn),
			Events: NewEventsDB(conn),
		},
	}, nil
}
