package parser

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/keith-hung/timecard-cli/internal/types"
)

var timearrayRegex = regexp.MustCompile(
	`timearray\[(\d+)\]\[(\d+)\]\[(\d+)\]\s*=\s*"([^"]*)"`,
)

var overtimeRegex = regexp.MustCompile(
	`overtimeArray\[(\d+)\]\[(\d+)\]\[(\d+)\]\s*=\s*"([^"]*)"`,
)

// ParseTimearray parses timearray[i][j][k] = "duration$status$note$progress".
func ParseTimearray(html string) []types.TimearrayEntry {
	return parseArrayEntries(timearrayRegex, html)
}

// ParseOvertimeArray parses overtimeArray[i][j][k] (same format as timearray).
func ParseOvertimeArray(html string) []types.TimearrayEntry {
	return parseArrayEntries(overtimeRegex, html)
}

func parseArrayEntries(re *regexp.Regexp, html string) []types.TimearrayEntry {
	matches := re.FindAllStringSubmatch(html, -1)
	results := make([]types.TimearrayEntry, 0, len(matches))
	for _, m := range matches {
		projIdx, _ := strconv.Atoi(m[1])
		actIdx, _ := strconv.Atoi(m[2])
		dayIdx, _ := strconv.Atoi(m[3])
		parts := strings.SplitN(m[4], "$", 4)

		entry := types.TimearrayEntry{
			ProjectIndex:  projIdx,
			ActivityIndex: actIdx,
			DayIndex:      dayIdx,
		}
		if len(parts) > 0 {
			entry.Duration = parts[0]
		}
		if len(parts) > 1 {
			entry.Status = parts[1]
		}
		if len(parts) > 2 {
			entry.Note = strings.ReplaceAll(parts[2], `\n`, "\n")
		}
		if len(parts) > 3 {
			entry.Progress = parts[3]
		}
		results = append(results, entry)
	}
	return results
}
