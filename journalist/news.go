package journalist

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/microcosm-cc/bluemonday"
	"strings"
	"time"
)

type News struct {
	ID           string    // ID is the md5 hash of URL + title + description + date
	Title        string    // Title is the title of the news
	Description  string    // Description is the description of the news
	Link         string    // Link is the link to the news
	Date         time.Time // Date is the date of the news
	ProviderName string    // ProviderName is the name of the provider that fetched the news
	IsSuspicious bool      // IsSuspicious is true if the news contains keywords that should be checked by human before publishing
	// TODO: Add creator field if possible
}

func NewNews(title, description, link, date, provider string) (*News, error) {
	dateTime, err := parseDate(date)
	if err != nil {
		return nil, err
	}

	// Sanitize title and description, because they may contain HTML tags and styles
	p := bluemonday.StrictPolicy()
	title = p.Sanitize(title)
	description = p.Sanitize(description)

	hash := md5.Sum([]byte(link + title + description + dateTime.String()))

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

// FindById finds news by its hash id (URL + title + description + date)
func (n NewsList) FindById(id string) *News {
	for _, news := range n {
		if news.ID == id {
			return news
		}
	}
	return nil
}

// FilterByKeywords returns only a list of news that contains at least one of the keywords
func (n NewsList) FilterByKeywords(keywords []string) NewsList {
	var filteredNews NewsList
	for _, n := range n {
		c := false
		// Check if any keyword is present in the title & description
		for _, k := range keywords {
			if strings.Contains(fmt.Sprintf("%s %s", n.Title, n.Description), k) {
				c = true
				break
			}
		}
		if c {
			filteredNews = append(filteredNews, n)
		}
	}

	return filteredNews
}

// FlagByKeywords sets IsSuspicious to true if the news contains at least one of the keywords
func (n NewsList) FlagByKeywords(keywords []string) {
	for _, news := range n {
		for _, k := range keywords {
			s := strings.ToLower(fmt.Sprintf("%s %s", news.Title, news.Description))
			if strings.Contains(s, k) {
				news.IsSuspicious = true
				break
			}
		}
	}
}

// MapIDs removes duplicates news by creating a map of ID hashes.
// Since same news can be fetched from multiple feeds, we need to filter them out.
func (n NewsList) MapIDs() NewsList {
	var filteredNews NewsList

	// Create a map of news ID to news
	newsMap := make(map[string]*News)
	for _, news := range n {
		newsMap[news.ID] = news
	}

	// Create a list of news from the map
	for _, news := range newsMap {
		filteredNews = append(filteredNews, news)
	}

	return filteredNews
}
