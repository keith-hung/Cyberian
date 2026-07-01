// Package cmd implements the chpw CLI commands using cobra.
package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/keith-hung/chpw-cli/internal/types"
	"github.com/spf13/cobra"
)

// GlobalFlags holds the parsed global flags.
type GlobalFlags struct {
	URL         string
	User        string
	Pass        string
	SessionFile string
	Insecure    bool
	Pretty      bool
	PassStdin   bool
}

var gf GlobalFlags

var rootCmd = &cobra.Command{
	Use:           "chpw",
	Short:         "Change on-prem AD password via the self-service portal (two-step: login then submit)",
	SilenceUsage:  true,
	SilenceErrors: true,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		ExitError("usage: chpw <login|submit|version> [flags]. Use --help for details.", 1)
		return nil
	},
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if gf.PassStdin {
			readPasswordFromStdin()
		}
	},
}

// readPasswordFromStdin reads exactly one line from stdin. It refuses to read
// from a TTY (no interactive fallback) so the tool is safe to drive by agents.
func readPasswordFromStdin() {
	info, err := os.Stdin.Stat()
	if err == nil && info.Mode()&os.ModeCharDevice != 0 {
		ExitError("--pass-stdin requires a piped password; stdin is a TTY", 3)
	}
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		gf.Pass = strings.TrimSpace(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		ExitError(fmt.Sprintf("reading password from stdin: %v", err), 1)
	}
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		ExitError(err.Error(), classifyError(err))
	}
}

func init() {
	rootCmd.SetOut(os.Stderr)
	rootCmd.PersistentFlags().StringVar(&gf.URL, "url", envOrDefault("CHPW_BASE_URL", ""), "Portal base URL")
	rootCmd.PersistentFlags().StringVar(&gf.User, "user", envOrDefault("CHPW_USERNAME", ""), "Username")
	rootCmd.PersistentFlags().BoolVar(&gf.PassStdin, "pass-stdin", false, "Read password from stdin (required)")
	rootCmd.PersistentFlags().StringVar(&gf.SessionFile, "session-file", ".chpw-session.json", "Session file path")
	rootCmd.PersistentFlags().BoolVar(&gf.Insecure, "insecure", envBool("CHPW_INSECURE"), "Skip TLS verification (dev only)")
	rootCmd.PersistentFlags().BoolVar(&gf.Pretty, "pretty", false, "Pretty-print JSON output")
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envBool(key string) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	return v == "1" || v == "true" || v == "yes"
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
	data, _ := json.Marshal(types.ErrorOutput{Success: false, Error: msg})
	fmt.Fprintln(os.Stderr, string(data))
	os.Exit(code)
}

// classifyError maps an error message to an exit code.
func classifyError(err error) int {
	lower := strings.ToLower(err.Error())
	switch {
	case containsAny(lower, "authentication", "invalid username", "bad password", "user not found", "antiforgery"):
		return 2
	case containsAny(lower, "validation", "otp", "expired", "policy", "must be", "required"):
		return 3
	case containsAny(lower, "network", "connection", "timeout", "no such host", "dial", "post ", "get "):
		return 4
	default:
		return 1
	}
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
