// Package cmd implements CLI commands using cobra.
package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/keith-hung/timecard-cli/internal/session"
	"github.com/keith-hung/timecard-cli/internal/types"
	"github.com/spf13/cobra"
)

// GlobalFlags holds the parsed global flags.
type GlobalFlags struct {
	URL         string
	User        string
	Pass        string
	SessionFile string
	Pretty      bool
	PassStdin   bool
}

var gf GlobalFlags

var rootCmd = &cobra.Command{
	Use:           "timecard",
	Short:         "TimeCard timesheet management CLI",
	SilenceUsage:  true,
	SilenceErrors: true,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		ExitError("usage: timecard <command> [flags]. Use --help for details.", 1)
		return nil
	},
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Load password from env var first.
		gf.Pass = envOrDefault("TIMECARD_PASSWORD", "")

		// Override with stdin if requested.
		if gf.PassStdin {
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				gf.Pass = strings.TrimSpace(scanner.Text())
			}
			if err := scanner.Err(); err != nil {
				ExitError(fmt.Sprintf("reading password from stdin: %v", err), 1)
			}
		}
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		ExitError(err.Error(), 1)
	}
}

func init() {
	rootCmd.SetOut(os.Stderr)
	rootCmd.PersistentFlags().StringVar(&gf.URL, "url", envOrDefault("TIMECARD_BASE_URL", ""), "TimeCard base URL")
	rootCmd.PersistentFlags().StringVar(&gf.User, "user", envOrDefault("TIMECARD_USERNAME", ""), "Username")
	rootCmd.PersistentFlags().BoolVar(&gf.PassStdin, "pass-stdin", false, "Read password from stdin")
	rootCmd.PersistentFlags().StringVar(&gf.SessionFile, "session-file", ".timecard-session.json", "Session file path")
	rootCmd.PersistentFlags().BoolVar(&gf.Pretty, "pretty", false, "Pretty-print JSON output")
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
