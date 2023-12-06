package models

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"time"
)

type NewsDB struct {
	Conn *gorm.DB
}

func NewNewsDB(db *gorm.DB) *NewsDB {
	return &NewsDB{Conn: db.Table("news")}
}

type News struct {
	ID            uuid.UUID      `gorm:"primaryKey;type:uuid;not null;" json:"id"`  // ID of the news (UUID)
	Hash          string         `gorm:"size:32;uniqueIndex;not null;" json:"hash"` // MD5 Hash of the news (URL + title + description + date)
	ChannelID     string         `gorm:"size:64" json:"channel_id"`                 // ID of the channel (chat ID in Telegram)
	PublicationID string         `gorm:"size:64" json:"publication_id"`             // ID of the publication (message ID in Telegram)
	URL           string         `gorm:"size:256" json:"url"`                       // URL of the original news
	OriginalTitle string         `gorm:"size:256" json:"original_title"`            // Original News title
	OriginalDesc  string         `gorm:"size:1024" json:"original_desc"`            // Original News description
	ComposedText  string         `gorm:"size:512" json:"composed_text"`             // Composed text
	MetaData      datatypes.JSON `gorm:"" json:"meta_data"`                         // Meta data (tickers, markets, hashtags, etc.)
	PublishedAt   time.Time      `gorm:"default:null" json:"published_at"`          // Composed News publication date
	OriginalDate  time.Time      `gorm:"not null" json:"original_date"`             // Original News date
	CreatedAt     time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"created_at,omitempty"`
	UpdatedAt     time.Time      `gorm:"default:CURRENT_TIMESTAMP" json:"updated_at,omitempty"`
}

func (n *News) Validate() error {
	if len(n.ChannelID) > 64 {
		return errors.New("channel_id is too long")
	}

	if len(n.Hash) > 32 {
		return errors.New("hash is too long")
	}

	if len(n.PublicationID) > 64 {
		return errors.New("publication_id is too long")
	}

	if len(n.URL) > 256 {
		return errors.New("url is too long")
	}

	if len(n.OriginalTitle) > 256 {
		return errors.New("original_title is too long")
	}

	if len(n.OriginalDesc) > 1024 {
		return errors.New("original_desc is too long")
	}

	if len(n.ComposedText) > 512 {
		return errors.New("composed_text is too long")
	}

	if n.OriginalDate.IsZero() {
		return errors.New("original_date is empty")
	}

	return nil
}

// GenerateHash generates the hash of the news (URL + title + description + date)
func (n *News) GenerateHash() {
	h := md5.Sum([]byte(n.URL + n.OriginalTitle + n.OriginalDesc + n.OriginalDate.String()))
	n.Hash = hex.EncodeToString(h[:])
}

func (n *News) BeforeCreate(*gorm.DB) error {
	// Create UUID ID.
	n.ID = uuid.New()

	if len(n.Hash) == 0 {
		n.GenerateHash()
	}

	if len(n.OriginalDesc) > 1024 {
		n.OriginalDesc = n.OriginalDesc[:1024]
	}

	err := n.Validate()
	if err != nil {
		return err
	}

	return nil
}

func (db *NewsDB) Create(ctx context.Context, n *News) error {
	res := db.Conn.WithContext(ctx).Create(&n)
	if res.Error != nil {
		return res.Error
	}

	return nil
}

func (db *NewsDB) Update(ctx context.Context, n *News) error {
	res := db.Conn.WithContext(ctx).Save(&n)
	if res.Error != nil {
		return res.Error
	}

	return nil
}

// FindAllByHashes finds news by its hash (URL + title + description + date)
func (db *NewsDB) FindAllByHashes(ctx context.Context, hashes []string) ([]*News, error) {
	var n []*News
	res := db.Conn.WithContext(ctx).Where("hash IN ?", hashes).Find(&n)
	if res.Error != nil {
		return nil, res.Error
	}

	return n, nil
}

// FindAllUntilDate finds all news until the provided published date
func (db *NewsDB) FindAllUntilDate(ctx context.Context, until time.Time) ([]*News, error) {
	var n []*News
	res := db.Conn.WithContext(ctx).Where("published_at >= ?", until).Find(&n)
	if res.Error != nil {
		return nil, res.Error
	}

	return n, nil
}
