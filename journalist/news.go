package journalist

import (
	"crypto/md5"
	"encoding/hex"
	"time"
)

type News struct {
	ID           string    // ID is the md5 hash of link + date
	Title        string    // Title is the title of the news
	Description  string    // Description is the description of the news
	Link         string    // Link is the link to the news
	Date         time.Time // Date is the date of the news
	ProviderName string    // ProviderName is the name of the provider that fetched the news
	// TODO: Add creator field if possible
}

func NewNews(title, description, link, date, provider string) (*News, error) {
	dateTime, err := parseDate(date)
	if err != nil {
		return nil, err
	}

	hash := md5.Sum([]byte(link + dateTime.String()))

	return &News{
		ID:           hex.EncodeToString(hash[:]),
		Title:        title,
		Description:  description,
		Link:         link,
		Date:         dateTime,
		ProviderName: provider,
	}, nil
}
