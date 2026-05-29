// Package parser extracts data from the Nouveau Timecard timesheet HTML page:
// antiforgery token, project options, activity cache, and the embedded
// serverTimeRows / serverOvertimeRows JSON.
package parser

import (
	"encoding/json"
	stdhtml "html"
	"regexp"
	"strings"

	"github.com/keith-hung/nouveau-timecard-cli/internal/types"
)

var (
	tokenRe         = regexp.MustCompile(`name="__RequestVerificationToken"[^>]*\bvalue="([^"]+)"`)
	tokenReAlt      = regexp.MustCompile(`<input[^>]*\bvalue="([^"]+)"[^>]*name="__RequestVerificationToken"`)
	projectOptRe    = regexp.MustCompile(`(?s)<option value="(\d+)" data-status="[^"]*">\s*(.*?)\s*</option>`)
	projectCodeRe   = regexp.MustCompile(`^\[([^\]]*)\]\s*(.*)$`)
	activityOptRe   = regexp.MustCompile(`(?s)<option value="(\d+)">(.*?)</option>`)
	passwordFieldRe = regexp.MustCompile(`(?i)(name|id)="password"`)
)

// ParseAntiForgeryToken returns the __RequestVerificationToken hidden field value.
func ParseAntiForgeryToken(page string) string {
	if m := tokenRe.FindStringSubmatch(page); m != nil {
		return m[1]
	}
	if m := tokenReAlt.FindStringSubmatch(page); m != nil {
		return m[1]
	}
	return ""
}

// IsLoginPage reports whether the HTML is the login page (has a password field).
func IsLoginPage(page string) bool {
	return passwordFieldRe.MatchString(page)
}

// ParseProjectOptions returns deduplicated project options from the project
// <select> elements (identified by the data-status attribute). Display text is
// HTML-entity decoded (the server emits CJK as numeric entities).
func ParseProjectOptions(page string) []types.ProjectOption {
	seen := make(map[string]bool)
	var out []types.ProjectOption
	for _, m := range projectOptRe.FindAllStringSubmatch(page, -1) {
		id := m[1]
		text := strings.TrimSpace(stdhtml.UnescapeString(m[2]))
		if seen[id] {
			continue
		}
		seen[id] = true
		code, name := "", text
		if cm := projectCodeRe.FindStringSubmatch(text); cm != nil {
			code = strings.TrimSpace(cm[1])
			name = strings.TrimSpace(cm[2])
		}
		out = append(out, types.ProjectOption{ID: id, Code: code, Name: name})
	}
	return out
}

// ParseActivityCache parses the activityHtmlCache object (projectID → option HTML)
// into a map of projectID → selectable activities.
func ParseActivityCache(page string) (map[string][]types.Activity, error) {
	expr, ok := extractJSExpr(page, "activityHtmlCache")
	if !ok {
		return map[string][]types.Activity{}, nil
	}
	var cache map[string]string
	if err := json.Unmarshal([]byte(expr), &cache); err != nil {
		return nil, err
	}
	result := make(map[string][]types.Activity, len(cache))
	for projectID, optionHTML := range cache {
		var acts []types.Activity
		for _, m := range activityOptRe.FindAllStringSubmatch(optionHTML, -1) {
			acts = append(acts, types.Activity{
				ID:        m[1],
				ProjectID: projectID,
				Name:      strings.TrimSpace(stdhtml.UnescapeString(m[2])),
			})
		}
		result[projectID] = acts
	}
	return result, nil
}

// ParseServerRows parses serverTimeRows or serverOvertimeRows into []ServerRow.
// varName is "serverTimeRows" or "serverOvertimeRows".
func ParseServerRows(page, varName string) ([]types.ServerRow, error) {
	expr, ok := extractJSExpr(page, varName)
	if !ok {
		return nil, nil
	}
	var rows []types.ServerRow
	if err := json.Unmarshal([]byte(expr), &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

// extractJSExpr finds the JS assignment `varName = <literal>` (where <literal>
// is an object or array) and returns the literal via string-aware bracket
// matching. It skips non-assignment occurrences such as the Alpine
// `x-data="batchTimeEntry(serverTimeRows, ...)"` call that references the same
// identifiers, and is robust against braces/semicolons inside string values.
func extractJSExpr(page, varName string) (string, bool) {
	search := 0
	for {
		rel := strings.Index(page[search:], varName)
		if rel < 0 {
			return "", false
		}
		idx := search + rel
		search = idx + len(varName)

		// Require the next non-space character to be '=' (an assignment),
		// then a '{' or '[' literal — otherwise this is a reference, skip it.
		j := idx + len(varName)
		for j < len(page) && isSpace(page[j]) {
			j++
		}
		if j >= len(page) || page[j] != '=' {
			continue
		}
		k := j + 1
		for k < len(page) && isSpace(page[k]) {
			k++
		}
		if k >= len(page) || (page[k] != '{' && page[k] != '[') {
			continue
		}
		if lit, ok := scanBalanced(page, k); ok {
			return lit, true
		}
	}
}

// scanBalanced returns the balanced {…} or […] literal starting at page[start],
// treating double-quoted strings (with backslash escapes) as opaque.
func scanBalanced(page string, start int) (string, bool) {
	open := page[start]
	var closeCh byte
	switch open {
	case '{':
		closeCh = '}'
	case '[':
		closeCh = ']'
	default:
		return "", false
	}
	depth := 0
	inStr := false
	escaped := false
	for i := start; i < len(page); i++ {
		ch := page[i]
		if inStr {
			if escaped {
				escaped = false
			} else if ch == '\\' {
				escaped = true
			} else if ch == '"' {
				inStr = false
			}
			continue
		}
		switch ch {
		case '"':
			inStr = true
		case open:
			depth++
		case closeCh:
			depth--
			if depth == 0 {
				return page[start : i+1], true
			}
		}
	}
	return "", false
}

func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\r' || b == '\n'
}
