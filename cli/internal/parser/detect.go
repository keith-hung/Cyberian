package parser

import (
	"regexp"
	"strings"

	"github.com/keith-hung/timecard-cli/internal/types"
)

var errorMainRegex = regexp.MustCompile(`<b>([^<]*)</b>`)
var errorTypeRegex = regexp.MustCompile(`Exception Type:\s*</td>\s*<td[^>]*>([^<]*)`)
var errorMsgRegex = regexp.MustCompile(`Exception Message:\s*</td>\s*<td[^>]*>([^<]*)`)

// IsLoginPage detects if the response is a login page (session expired).
func IsLoginPage(html, finalURL string) bool {
	return strings.Contains(finalURL, "login.jsp") ||
		strings.Contains(html, `name="userform"`) ||
		strings.Contains(html, "<title>Login</title>")
}

// IsErrorPage detects if the response is an error page.
func IsErrorPage(html string) bool {
	return strings.Contains(html, "Error Page") ||
		strings.Contains(html, "/errorMsg/error.jsp")
}

// ParseErrorPage extracts error details if the HTML is an error page.
// Returns nil if not an error page.
func ParseErrorPage(html string) *types.ErrorInfo {
	if !IsErrorPage(html) {
		return nil
	}

	info := &types.ErrorInfo{
		MainMessage: "Unknown error",
	}

	if m := errorMainRegex.FindStringSubmatch(html); m != nil {
		info.MainMessage = strings.TrimSpace(m[1])
	}
	if m := errorTypeRegex.FindStringSubmatch(html); m != nil {
		info.ExceptionType = strings.TrimSpace(m[1])
	}
	if m := errorMsgRegex.FindStringSubmatch(html); m != nil {
		info.ExceptionMessage = strings.TrimSpace(m[1])
	}

	return info
}
