// Package cmd implements CLI subcommand routing and global flag handling.
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/keith-hung/wedaka-cli/internal/client"
	"github.com/keith-hung/wedaka-cli/internal/types"
)

// GlobalFlags holds the parsed global flags.
type GlobalFlags struct {
	URL      string
	Username string
	EmpNo    string
	DeviceID string
	Pretty   bool
}

// ParseGlobalFlags extracts global flags from args. Returns parsed flags and remaining args.
func ParseGlobalFlags(args []string) (*GlobalFlags, []string) {
	gf := &GlobalFlags{
		URL:      envOrDefault("WEDAKA_API_URL", ""),
		Username: envOrDefault("WEDAKA_USERNAME", ""),
		EmpNo:    envOrDefault("WEDAKA_EMP_NO", ""),
		DeviceID: envOrDefault("WEDAKA_DEVICE_ID", ""),
	}

	var remaining []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--url" && i+1 < len(args):
			i++
			gf.URL = args[i]
		case arg == "--username" && i+1 < len(args):
			i++
			gf.Username = args[i]
		case arg == "--emp-no" && i+1 < len(args):
			i++
			gf.EmpNo = args[i]
		case arg == "--device-id" && i+1 < len(args):
			i++
			gf.DeviceID = args[i]
		case arg == "--pretty":
			gf.Pretty = true
		default:
			remaining = append(remaining, arg)
		}
	}

	return gf, remaining
}

// NewClient creates a client.Client from global flags.
func NewClient(gf *GlobalFlags) *client.Client {
	if gf.URL == "" {
		ExitError("--url or WEDAKA_API_URL is required", 2)
	}
	if gf.DeviceID == "" {
		ExitError("--device-id or WEDAKA_DEVICE_ID is required", 2)
	}
	return client.New(gf.URL, gf.DeviceID)
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

// ExitErrorInfer writes an error JSON to stderr and infers the exit code from the message.
func ExitErrorInfer(msg string) {
	lower := strings.ToLower(msg)
	code := 1
	switch {
	case containsAny(lower, "network", "timeout", "connection", "refused"):
		code = 4
	case containsAny(lower, "invalid", "format", "required", "validation"):
		code = 3
	case containsAny(lower, "config", "environment", "url", "device"):
		code = 2
	}
	ExitError(msg, code)
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
