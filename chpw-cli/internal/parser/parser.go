// Package parser extracts the antiforgery token, distinguishes the portal's
// login / OTP / complete pages, and pulls out server-side error messages.
package parser

import (
	"html"
	"regexp"
	"strings"
)

var (
	tokenRe    = regexp.MustCompile(`name="__RequestVerificationToken"[^>]*\bvalue="([^"]+)"`)
	tokenReAlt = regexp.MustCompile(`<input[^>]*\bvalue="([^"]+)"[^>]*name="__RequestVerificationToken"`)
	// A validation span carrying an actual error: class contains
	// field-validation-error and the inner text is the message.
	errorRe = regexp.MustCompile(`(?s)<span[^>]*class="[^"]*field-validation-error[^"]*"[^>]*>(.*?)</span>`)
	tagRe   = regexp.MustCompile(`<[^>]+>`)
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

// IsLoginPage reports whether the page is the credentials (step 1) form.
func IsLoginPage(page string) bool {
	return strings.Contains(page, `name="Username"`) && strings.Contains(page, `name="Password"`)
}

// IsOtpPage reports whether the page is the new-password + OTP (step 2) form.
func IsOtpPage(page string) bool {
	return strings.Contains(page, `name="Otp"`) || strings.Contains(page, "請輸入簡訊驗證碼")
}

// IsCompletePage reports whether the page is the success confirmation.
func IsCompletePage(page string) bool {
	return strings.Contains(page, "新密碼已修改完成") || strings.Contains(page, "恭喜")
}

// ParseErrorMessage joins the text of any field-validation-error spans. Returns
// "" when the page carries no server-side error.
func ParseErrorMessage(page string) string {
	var msgs []string
	for _, m := range errorRe.FindAllStringSubmatch(page, -1) {
		text := strings.TrimSpace(html.UnescapeString(tagRe.ReplaceAllString(m[1], "")))
		if text != "" {
			msgs = append(msgs, text)
		}
	}
	return strings.Join(msgs, "; ")
}
