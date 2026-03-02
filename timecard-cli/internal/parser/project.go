package parser

import (
	"regexp"
	"strings"

	"github.com/keith-hung/timecard-cli/internal/types"
)

var selectRegex = regexp.MustCompile(`name='project0'[^>]*>([\s\S]*?)</select>`)
var optionRegex = regexp.MustCompile(`<option\s+value='(\d+)'[^>]*>([^<]+)</option>`)

// ParseProjectOptions extracts <option> tags from the project0 select element.
func ParseProjectOptions(html string) []types.ProjectOption {
	selectMatch := selectRegex.FindStringSubmatch(html)
	if selectMatch == nil {
		return nil
	}

	matches := optionRegex.FindAllStringSubmatch(selectMatch[1], -1)
	results := make([]types.ProjectOption, 0, len(matches))
	for _, m := range matches {
		results = append(results, types.ProjectOption{
			ID:   m[1],
			Name: strings.TrimSpace(m[2]),
		})
	}
	return results
}
