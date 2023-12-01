package journalist

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
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

type NewsList []*News

// ToJSON returns the JSON of the news list
func (n NewsList) ToJSON() (string, error) {
	jsonData, err := json.Marshal(n)
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}

// ToContentJSON returns the JSON of the news content only: id, title, description
func (n NewsList) ToContentJSON() (string, error) {
	type simpleNews struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		Description string `json:"description"`
	}

	var contentNews []*simpleNews
	for _, news := range n {
		contentNews = append(contentNews, &simpleNews{
			ID:          news.ID,
			Title:       news.Title,
			Description: news.Description,
		})
	}

	jsonData, err := json.Marshal(contentNews)
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}
