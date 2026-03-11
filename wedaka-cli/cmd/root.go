// Package cmd implements CLI commands using cobra.
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/keith-hung/wedaka-cli/internal/client"
	"github.com/keith-hung/wedaka-cli/internal/types"
	"github.com/spf13/cobra"
)

// GlobalFlags holds the parsed global flags.
type GlobalFlags struct {
	URL      string
	Username string
	EmpNo    string
	DeviceID string
	Pretty   bool
}

var gf GlobalFlags

var rootCmd = &cobra.Command{
	Use:           "wedaka",
	Short:         "WeDaka attendance management CLI",
	SilenceUsage:  true,
	SilenceErrors: true,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		ExitError(err.Error(), 1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&gf.URL, "url", envOrDefault("WEDAKA_API_URL", ""), "WeDaka API URL")
	rootCmd.PersistentFlags().StringVar(&gf.Username, "username", envOrDefault("WEDAKA_USERNAME", ""), "Username")
	rootCmd.PersistentFlags().StringVar(&gf.EmpNo, "emp-no", envOrDefault("WEDAKA_EMP_NO", ""), "Employee number")
	rootCmd.PersistentFlags().StringVar(&gf.DeviceID, "device-id", envOrDefault("WEDAKA_DEVICE_ID", ""), "Device ID")
	rootCmd.PersistentFlags().BoolVar(&gf.Pretty, "pretty", false, "Pretty-print JSON output")
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
