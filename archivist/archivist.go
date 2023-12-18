package archivist

import (
	"github.com/samgozman/fin-thread/archivist/models"
	"gorm.io/gorm"
)

// Entities is a struct that contains all the entities that Archivist is responsible for.
type Entities struct {
	News   *models.NewsDB
	Events *models.EventsDB
}

// Archivist is responsible for storing and retrieving data from the database.
type Archivist struct {
	db       *gorm.DB
	Entities *Entities
}

// NewArchivist creates a new Archivist with provided DSN to connect to database.
//
// DSN is a string in the format of: "user=gorm password=gorm dbname=gorm port=9920 sslmode=disable"
func NewArchivist(dsn string) (*Archivist, error) {
	conn, err := connectToPG(dsn)
	if err != nil {
		return nil, err
	}

	// Migrate the schema automatically for now.
	// TODO: Add migration tool later.
	err = conn.AutoMigrate(&models.News{})
	if err != nil {
		return nil, err
	}

	return &Archivist{
		db: conn,
		Entities: &Entities{
			News:   models.NewNewsDB(conn),
			Events: models.NewEventsDB(conn),
		},
	}, nil
}
