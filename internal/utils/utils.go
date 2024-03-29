package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ParseDate parses a date string into a time.Time object in UTC.
func ParseDate(dateString Datable) (time.Time, error) {
	var timestamp int64
	// switch type
	switch dateString := dateString.(type) {
	case nil:
		return time.Time{}, nil
	case string:
		if dateString == "" {
			return time.Time{}, nil
		}
		// List of potential layouts to try
		layouts := []string{
			time.RFC1123,
			time.RFC1123Z,
			time.RFC3339,
			"2006-01-02T15:04:05",
		}

		var parsedTime time.Time
		var err error

		for _, layout := range layouts {
			parsedTime, err = time.Parse(layout, dateString)
			if err == nil {
				return parsedTime.UTC(), nil
			}
		}

		// If none of the layouts could parse the date string, return the last error
		if err != nil {
			return time.Time{}, fmt.Errorf("error parsing date: %s, error: %w", dateString, err)
		}
	case int:
		timestamp = int64(dateString)
	case int32:
		timestamp = int64(dateString)
	case int64:
		timestamp = dateString

	default:
		return time.Time{}, fmt.Errorf("unknown type: %T of value %v", dateString, dateString)
	}

	if timestamp == 0 {
		return time.Time{}, nil
	}

	// If Unix milliseconds - convert to seconds
	var maxPossibleSeconds int64 = 9999999999
	var millisecondsInSecond int64 = 1000
	if timestamp > maxPossibleSeconds {
		return time.Unix(timestamp/millisecondsInSecond, 0).UTC(), nil
	}
	return time.Unix(timestamp, 0).UTC(), nil
}

// Datable is a type that can be parsed into a date (hopefully).
type Datable interface{}

func StrValueToFloat(value string) float64 {
	var result float64
	// Remove all non-digit and non-dot/comma characters
	re := regexp.MustCompile("[^0-9.,]+")
	value = re.ReplaceAllString(value, "")
	// Replace comma with dot
	value = strings.Replace(value, ",", ".", 1)

	// Convert the cleaned string to float64
	result, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return result
}

// ReplaceUnicodeSymbols replaces Unicode escape sequences with their corresponding characters.
func ReplaceUnicodeSymbols(s string) string {
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
