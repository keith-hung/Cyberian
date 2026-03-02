package parser

import (
	"regexp"
)

var formRegex = regexp.MustCompile(`(?s)name="weekly_info".*?</form>`)
var hiddenInputRegex = regexp.MustCompile(`(?i)<input\s+[^>]*type="hidden"[^>]*>`)
var nameAttrRegex = regexp.MustCompile(`name="([^"]*)"`)
var valueAttrRegex = regexp.MustCompile(`value="([^"]*)"`)
var cdateRegex = regexp.MustCompile(`name="cdate"[^>]*value="([^"]*)"`)

// ParseHiddenInputs extracts all <input type="hidden"> fields from the weekly_info form.
func ParseHiddenInputs(html string) map[string]string {
	results := make(map[string]string)

	searchHTML := html
	if formMatch := formRegex.FindString(html); formMatch != "" {
		searchHTML = formMatch
	}

	matches := hiddenInputRegex.FindAllString(searchHTML, -1)
	for _, tag := range matches {
		nameMatch := nameAttrRegex.FindStringSubmatch(tag)
		if nameMatch == nil {
			continue
		}
		valueMatch := valueAttrRegex.FindStringSubmatch(tag)
		value := ""
		if valueMatch != nil {
			value = valueMatch[1]
		}
		results[nameMatch[1]] = value
	}

	return results
}

// ParseWeekDate extracts the cdate hidden input value.
func ParseWeekDate(html string) string {
	match := cdateRegex.FindStringSubmatch(html)
	if match == nil {
		return ""
	}
	return match[1]
}
