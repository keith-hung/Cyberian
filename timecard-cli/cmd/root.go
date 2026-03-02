// Package cmd implements CLI subcommand routing and global flag handling.
package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/keith-hung/timecard-cli/internal/session"
	"github.com/keith-hung/timecard-cli/internal/types"
)

// GlobalFlags holds the parsed global flags.
type GlobalFlags struct {
	URL         string
	User        string
	Pass        string
	SessionFile string
	Pretty      bool
	PassStdin   bool // read password from stdin
}

// ParseGlobalFlags extracts global flags from args (which may be interleaved
// with subcommand flags). Returns parsed global flags and remaining args.
func ParseGlobalFlags(args []string) (*GlobalFlags, []string) {
	gf := &GlobalFlags{
		URL:         envOrDefault("TIMECARD_BASE_URL", ""),
		User:        envOrDefault("TIMECARD_USERNAME", ""),
		Pass:        envOrDefault("TIMECARD_PASSWORD", ""),
		SessionFile: ".timecard-session.json",
	}

	var remaining []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--url" && i+1 < len(args):
			i++
			gf.URL = args[i]
		case arg == "--user" && i+1 < len(args):
			i++
			gf.User = args[i]
		case arg == "--pass-stdin":
			gf.PassStdin = true
		case arg == "--session-file" && i+1 < len(args):
			i++
			gf.SessionFile = args[i]
		case arg == "--pretty":
			gf.Pretty = true
		default:
			remaining = append(remaining, arg)
		}
	}

	// Read password from stdin if requested
	if gf.PassStdin {
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			gf.Pass = strings.TrimSpace(scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			ExitError(fmt.Sprintf("reading password from stdin: %v", err), 1)
		}
	}

	return gf, remaining
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// NewSession creates a session.Session from global flags.
func NewSession(gf *GlobalFlags) (*session.Session, error) {
	if gf.URL == "" {
		return nil, fmt.Errorf("--url or TIMECARD_BASE_URL is required")
	}
	if gf.User == "" {
		return nil, fmt.Errorf("--user or TIMECARD_USERNAME is required")
	}
	if gf.Pass == "" {
		return nil, fmt.Errorf("TIMECARD_PASSWORD env var or --pass-stdin is required")
	}

	return session.New(session.Config{
		BaseURL:     gf.URL,
		Username:    gf.User,
		Password:    gf.Pass,
		SessionFile: gf.SessionFile,
	})
}

// OutputJSON marshals v to JSON and writes to stdout.
func OutputJSON(v interface{}, pretty bool) {
	var data []byte
	var err error
	if pretty {
		data, err = json.MarshalIndent(v, "", "  ")
	} else {
		data, err = json.Marshal(v)
	}
	if err != nil {
		ExitError(fmt.Sprintf("JSON marshal error: %v", err), 1)
	}
	fmt.Println(string(data))
}

// ExitError writes an error JSON to stderr and exits with the given code.
func ExitError(msg string, code int) {
	data, _ := json.Marshal(types.ErrorOutput{
		Success: false,
		Error:   msg,
	})
	fmt.Fprintln(os.Stderr, string(data))
	os.Exit(code)
}
