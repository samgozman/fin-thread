package archivist

import (
	"github.com/cenkalti/backoff/v4"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"time"
)

func connectToPG(dsn string) (*gorm.DB, error) {
	bf := backoff.NewExponentialBackOff()
	bf.InitialInterval = 10 * time.Second
	bf.MaxInterval = 25 * time.Second
	bf.MaxElapsedTime = 90 * time.Second

	db, err := backoff.RetryWithData[*gorm.DB](func() (*gorm.DB, error) {
		conn, err := gorm.Open(postgres.New(postgres.Config{
			DSN: dsn,
		}))
		if err != nil {
			log.Println("Postgres not yet ready...")
			return nil, err
		}
		log.Println("Connected to Postgres!")
		return conn, nil
	}, bf)
	if err != nil {
		return nil, err
	}

	return db, nil
}
