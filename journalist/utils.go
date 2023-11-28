package journalist

import "time"

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
