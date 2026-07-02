// Package types defines shared data structures for the chpw CLI.
package types

// NextStep tells the caller (human or agent) what to run after `login`.
type NextStep struct {
	Command string `json:"command"`
	Hint    string `json:"hint"`
}

// LoginOutput is the JSON output for the `login` command.
type LoginOutput struct {
	Success    bool     `json:"success"`
	Message    string   `json:"message"`
	OtpTTL     int      `json:"otp_ttl"`
	SessionTTL int      `json:"session_ttl"`
	Next       NextStep `json:"next"`
}

// SubmitOutput is the JSON output for the `submit` command.
type SubmitOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// VersionOutput is the JSON output for the `version` command.
type VersionOutput struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
}

// ErrorOutput is the JSON error output written to stderr.
type ErrorOutput struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}
