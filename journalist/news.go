package journalist

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/microcosm-cc/bluemonday"
	"github.com/samgozman/fin-thread/utils"
	"html"
	"regexp"
	"strings"
	"time"
)

type News struct {
	ID           string    // ID is the md5 hash of URL + title + description + date
	Title        string    // Title is the title of the news
	Description  string    // Description is the description of the news
	Link         string    // Link is the link to the news
	Date         time.Time // Date is the date of the news
	ProviderName string    // ProviderName is the Name of the provider that fetched the news
	IsSuspicious bool      // IsSuspicious is true if the news contains keywords that should be checked by human before publishing
	// TODO: Add creator field if possible
}

// newNews creates a new News instance from the given parameters.
// It sanitizes the title and description from HTML tags and styles.
// It also generates the ID of the news by hashing the link, title, description and date.
func newNews(title, description, link, date, provider string) (*News, error) {
	dateTime, err := utils.ParseDate(date)
	if err != nil {
		return nil, err
	}

	// Sanitize title and description, because they may contain HTML tags and styles
	p := bluemonday.StrictPolicy()
	title = p.Sanitize(title)
	description = p.Sanitize(description)

	// Replace code symbols like &#39; with their actual symbols.
	// This is placed after sanitization, because sanitization may replace some symbols along the way.
	title = html.UnescapeString(title)
	description = html.UnescapeString(description)

	// Replace Unicode escape sequences (e.g., \u0026)
	title = utils.ReplaceUnicodeSymbols(title)
	description = utils.ReplaceUnicodeSymbols(description)

	if len(description) > 1024 {
		description = description[:1024]
	}

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

func (n *News) Contains(keywords []string) bool {
	for _, k := range keywords {
		s := strings.ToLower(fmt.Sprintf("%s %s", n.Title, n.Description))
		match, _ := regexp.MatchString(fmt.Sprintf("\\b%s\\b", strings.ToLower(k)), s)
		if match {
			return true
		}
	}
	return false
}

type NewsList []*News

// ToJSON returns the JSON of the news list.
func (n NewsList) ToJSON() (string, error) {
	jsonData, err := json.Marshal(n)
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}

// ToContentJSON returns the JSON of the news content only: id, title, description.
func (n NewsList) ToContentJSON() (string, error) {
	type simpleNews struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		Description string `json:"description"`
	}

	contentNews := make([]*simpleNews, 0, len(n))
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

// filterByKeywords returns only a list of news that contains at least one of the keywords.
func (n NewsList) filterByKeywords(keywords []string) NewsList {
	var filteredNews NewsList
	for _, n := range n {
		if n.Contains(keywords) {
			filteredNews = append(filteredNews, n)
		}
	}

	return filteredNews
}

// flagByKeywords sets IsSuspicious to true if the news contains at least one of the keywords.
func (n NewsList) flagByKeywords(keywords []string) {
	for _, news := range n {
		if news.Contains(keywords) {
			news.IsSuspicious = true
		}
	}
}

// mapIDs removes duplicates news by creating a map of ID hashes.
// Since same news can be fetched from multiple feeds, we need to filter them out.
func (n NewsList) mapIDs() NewsList {
	filteredNews := make(NewsList, 0, len(n))

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
