package journalist

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/microcosm-cc/bluemonday"
	"github.com/samgozman/fin-thread/internal/utils"
	"github.com/samgozman/fin-thread/pkg/errlvl"
	"html"
	"regexp"
	"strings"
	"time"
)

type News struct {
	ID           string    // ID is the md5 hash of title + description
	Title        string    // Title is the title of the news
	Description  string    // Description is the description of the news
	Link         string    // Link is the link to the news
	Date         time.Time // Date is the date of the news
	ProviderName string    // ProviderName is the Name of the provider that fetched the news
	IsSuspicious bool      // IsSuspicious is true if the news contains keywords that should be checked by human before publishing
	IsFiltered   bool      // IsFiltered is true if the news was filtered out by others service (e.g. Composer.Filter)
	// TODO: Add creator field if possible
}

// newNews creates a new News instance from the given parameters.
// It sanitizes the title and description from HTML tags and styles.
// It also generates the ID of the news by hashing the link, title, description and date.
func newNews(title, description, link, date, provider string) (*News, error) {
	dateTime, err := utils.ParseDate(date)
	if err != nil {
		return nil, newError(errlvl.ERROR, fmt.Errorf("failed to parse date '%s'", date), err)
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

	hash := md5.Sum([]byte(title + description))

	return &News{
		ID:           hex.EncodeToString(hash[:]),
		Title:        title,
		Description:  description,
		Link:         link,
		Date:         dateTime,
		ProviderName: provider,
		IsFiltered:   false,
	}, nil
}

func (n *News) contains(keywords []string) bool {
	symbolsMatcherRe := regexp.MustCompile("^[^a-zA-Z0-9]*$")

	for _, k := range keywords {
		ke := strings.ToLower(regexp.QuoteMeta(k))

		var pattern string
		// Check that the keyword contains only symbols (for lagging by symbols feature)
		if symbolsMatcherRe.MatchString(k) {
			pattern = ke // Don't add word boundaries if the keyword contains only symbols
		} else {
			pattern = fmt.Sprintf("\\b%s\\b", ke)
		}

		s := strings.ToLower(fmt.Sprintf("%s %s", n.Title, n.Description))
		match, _ := regexp.MatchString(pattern, s)
		if match {
			return true
		}
	}
	return false
}

type NewsList []*News

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
		return "", newError(errlvl.ERROR, errMarshalSimpleNews, err)
	}
	return string(jsonData), nil
}

// RemoveFlagged returns a new NewsList without the flagged (IsFiltered, IsSuspicious) news.
func (n NewsList) RemoveFlagged() NewsList {
	var news NewsList
	for _, n := range n {
		if !n.IsFiltered && !n.IsSuspicious {
			news = append(news, n)
		}
	}

	return news
}

// filterByKeywords returns only a list of news that contains at least one of the keywords.
func (n NewsList) filterByKeywords(keywords []string) NewsList {
	var filteredNews NewsList
	for _, n := range n {
		if n.contains(keywords) {
			filteredNews = append(filteredNews, n)
		}
	}

	return filteredNews
}

// flagByKeywords sets IsSuspicious to true if the news contains at least one of the keywords.
func (n NewsList) flagByKeywords(keywords []string) {
	for _, news := range n {
		if news.contains(keywords) {
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
