package utils

import (
	"errors"
	"fmt"
	"time"
)

// ParseDate parses a date string into a time.Time object in UTC
func ParseDate(dateString Datable) (time.Time, error) {
	// switch type
	switch dateString.(type) {
	case nil:
		return time.Time{}, nil
	case int:
		// If Unix milliseconds - convert to seconds
		if dateString.(int) > 9999999999 {
			return time.Unix(int64(dateString.(int)/1000), 0).UTC(), nil
		}
		return time.Unix(int64(dateString.(int)), 0).UTC(), nil
	case string:
		// List of potential layouts to try
		layouts := []string{
			time.RFC1123,
			time.RFC1123Z,
			time.RFC3339,
		}

		var parsedTime time.Time
		var err error

		for _, layout := range layouts {
			parsedTime, err = time.Parse(layout, dateString.(string))
			if err == nil {
				break
			}
		}

		return parsedTime.UTC(), err
	default:
		return time.Time{}, errors.New(fmt.Sprintf("unknown type: %T of value %s", dateString, dateString))
	}
}

// Datable is a type that can be parsed into a date (hopefully)
type Datable interface{}
