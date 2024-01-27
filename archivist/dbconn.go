package archivist

import (
	"fmt"
	"github.com/cenkalti/backoff/v4"
	"github.com/samgozman/fin-thread/pkg/errlvl"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log/slog"
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
			slog.Info("[connectToPG] Postgres not yet ready...")
			return nil, fmt.Errorf("failed to connect to Postgres: %w", err)
		}
		slog.Info("[connectToPG] Connected to Postgres!")
		return conn, nil
	}, bf)
	if err != nil {
		return nil, newError(errlvl.FATAL, err)
	}

	return db, nil
}
