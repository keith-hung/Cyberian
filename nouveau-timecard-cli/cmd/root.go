// Package cmd implements the Nouveau Timecard CLI commands using cobra.
package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/keith-hung/nouveau-timecard-cli/internal/session"
	"github.com/keith-hung/nouveau-timecard-cli/internal/types"
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
	Use:           "nouveau-timecard",
	Short:         "Smart Timecard (智慧工時系統) timesheet CLI — draft only, never submits",
	SilenceUsage:  true,
	SilenceErrors: true,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		ExitError("usage: nouveau-timecard <command> [flags]. Use --help for details.", 1)
		return nil
	},
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		gf.Pass = envOrDefault("TIMECARD_PASSWORD", "")
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
	rootCmd.PersistentFlags().StringVar(&gf.URL, "url", envOrDefault("TIMECARD_BASE_URL", ""), "Server base URL")
	rootCmd.PersistentFlags().StringVar(&gf.User, "user", envOrDefault("TIMECARD_USERNAME", ""), "Username (LDAP account)")
	rootCmd.PersistentFlags().BoolVar(&gf.PassStdin, "pass-stdin", false, "Read password from stdin")
	rootCmd.PersistentFlags().StringVar(&gf.SessionFile, "session-file", ".nouveau-timecard-session.json", "Session file path")
	rootCmd.PersistentFlags().BoolVar(&gf.Insecure, "insecure", envBool("TIMECARD_INSECURE"), "Skip TLS verification (dev only)")
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
		Insecure:    gf.Insecure,
	})
}

// resolveYearMonth derives (year, month) from a --date YYYY-MM-DD flag or from
// --year/--month flags. If none are set, it defaults to the current month.
func resolveYearMonth(date string, year, month int) (int, int, error) {
	if date != "" {
		t, err := time.Parse("2006-01-02", date)
		if err != nil {
			return 0, 0, fmt.Errorf("invalid --date %q (want YYYY-MM-DD)", date)
		}
		return t.Year(), int(t.Month()), nil
	}
	if year != 0 || month != 0 {
		// One without the other is almost always a mistake — fail loudly rather
		// than silently using the current year/month.
		if year == 0 || month == 0 {
			return 0, 0, fmt.Errorf("provide both --year and --month together (or use --date)")
		}
		if month < 1 || month > 12 {
			return 0, 0, fmt.Errorf("--month must be 1-12")
		}
		return year, month, nil
	}
	now := time.Now()
	return now.Year(), int(now.Month()), nil
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

// classifyError maps an error message to an exit code (2 auth, 3 validation, 4 network, 1 general).
func classifyError(err error) int {
	lower := strings.ToLower(err.Error())
	switch {
	case containsAny(lower, "authentication", "login", "password", "username", "user not found", "antiforgery"):
		return 2
	case containsAny(lower, "invalid", "must be", "exceeds", "not in", "negative", "no records", "want yyyy"):
		return 3
	case containsAny(lower, "network", "connection", "timeout", "no such host", "dial"):
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

// addMonthFlags registers the shared --date/--year/--month flags.
func addMonthFlags(cmd *cobra.Command) {
	cmd.Flags().String("date", "", "Target date YYYY-MM-DD (alternative to --year/--month)")
	cmd.Flags().Int("year", 0, "Target year (e.g. 2026)")
	cmd.Flags().Int("month", 0, "Target month (1-12)")
}

func monthFromFlags(cmd *cobra.Command) (int, int, error) {
	date, _ := cmd.Flags().GetString("date")
	year, _ := cmd.Flags().GetInt("year")
	month, _ := cmd.Flags().GetInt("month")
	return resolveYearMonth(date, year, month)
}
