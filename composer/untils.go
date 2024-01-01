package composer

import "regexp"

// Find first array group. This will fix most weird OpenAI bugs with broken JSON
func aiJSONStringFixer(str string) (string, error) {
	// Find first array group in the string [{...}]
	re := regexp.MustCompile(`\[{([\S\s]*)}]`)
	matches := re.FindString(str)
	if matches != "" {
		return matches, nil
	}

	// If not, try first array []
	re = regexp.MustCompile(`\[([\S\s]*)]`)
	matches = re.FindString(str)
	if matches == "" {
		return "", newErr(ErrEmptyRegexMatch, "aiJSONStringFixer", "regexp.FindString").WithValue(str)
	}

	return matches, nil
}
