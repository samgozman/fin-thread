package journalist

import (
	"regexp"
	"strconv"
	"time"
)

// parseDate parses a date string into a time.Time object in UTC
func parseDate(dateString string) (time.Time, error) {
	// List of potential layouts to try
	layouts := []string{
		time.RFC1123,
		time.RFC1123Z,
		time.RFC3339,
	}

	var parsedTime time.Time
	var err error

	for _, layout := range layouts {
		parsedTime, err = time.Parse(layout, dateString)
		if err == nil {
			break
		}
	}

	return parsedTime.UTC(), err
}

// replaceUnicodeSymbols replaces Unicode escape sequences with their corresponding characters
func replaceUnicodeSymbols(s string) string {
	// Replace Unicode escape sequences (e.g., \u0026 with &)
	re := regexp.MustCompile(`\\u([0-9A-Fa-f]{4})`)
	decoded := re.ReplaceAllStringFunc(s, func(match string) string {
		unicodeCode := match[2:] // Ignore "\u" at the beginning
		num, err := strconv.ParseInt(unicodeCode, 16, 32)
		if err != nil {
			return match // If conversion fails, return the original sequence
		}
		// Convert Unicode code to a string and return the corresponding character
		return string(rune(num))
	})

	return decoded
}
