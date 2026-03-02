package parser

import (
	"regexp"
	"strings"

	"github.com/keith-hung/timecard-cli/internal/types"
)

// 5-param: act.append('projectId','name','isBottom','uid','progress')
var regex5Param = regexp.MustCompile(
	`act\.append\(\s*'([^']*)'\s*,\s*'((?:[^'\\]|\\.)*)'\s*,\s*'([^']*)'\s*,\s*'([^']*)'\s*,\s*'([^']*)'\s*\)`,
)

// 6-param: activityList.append('projectId','name','isBottom','uid','progress','activityId')
var regex6Param = regexp.MustCompile(
	`activityList\.append\(\s*'([^']*)'\s*,\s*'((?:[^'\\]|\\.)*)'\s*,\s*'([^']*)'\s*,\s*'([^']*)'\s*,\s*'([^']*)'\s*,\s*'([^']*)'\s*\)`,
)

// ParseActivityList parses the activity list from HTML.
// Supports both 5-param act.append() (live server) and
// 6-param activityList.append() (JSP source) formats.
func ParseActivityList(html string) []types.ActivityEntry {
	matches := regex5Param.FindAllStringSubmatch(html, -1)
	if len(matches) > 0 {
		results := make([]types.ActivityEntry, 0, len(matches))
		for _, m := range matches {
			results = append(results, types.ActivityEntry{
				ProjectID:  m[1],
				Name:       strings.ReplaceAll(m[2], `\'`, `'`),
				IsBottom:   m[3],
				UID:        m[4],
				Progress:   m[5],
				ActivityID: "",
			})
		}
		return results
	}

	// Fallback to 6-param format
	matches = regex6Param.FindAllStringSubmatch(html, -1)
	results := make([]types.ActivityEntry, 0, len(matches))
	for _, m := range matches {
		results = append(results, types.ActivityEntry{
			ProjectID:  m[1],
			Name:       strings.ReplaceAll(m[2], `\'`, `'`),
			IsBottom:   m[3],
			UID:        m[4],
			Progress:   m[5],
			ActivityID: m[6],
		})
	}
	return results
}
