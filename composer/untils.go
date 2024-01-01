package composer

import "regexp"

// Find first array group. This will fix most weird OpenAI bugs with broken JSON
func aiJSONStringFixer(str string) (string, error) {
	re := regexp.MustCompile(`\[([\S\s]*)\]`)
	matches := re.FindString(str)
	if matches == "" {
		return "", newErr(ErrEmptyRegexMatch, "aiJSONStringFixer", "regexp.FindString").WithValue(str)
	}
	return matches, nil
}
