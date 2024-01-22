package composer

import (
	"regexp"
	"strings"
)

// aiJSONStringFixer will fix the most weird OpenAI & Mistral bugs with a broken JSON array.
func aiJSONStringFixer(str string) (string, error) {
	// Often Mistral bug for empty arrays
	if str == "[[]]" || strings.Contains(str, "[\\]") {
		return "[]", nil
	}

	// Find a first array group in the string [{...}]
	re := regexp.MustCompile(`\[{([\S\s]*)}]`)
	matches := re.FindString(str)
	if matches != "" {
		return matches, nil
	}

	// If not, try a first array []
	re = regexp.MustCompile(`\[([\S\s]*)]`)
	matches = re.FindString(str)
	if matches == "" {
		return "", newErr(errEmptyRegexMatch, "aiJSONStringFixer", "regexp.FindString").WithValue(str)
	}

	return matches, nil
}
