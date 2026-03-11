// Package cmd implements CLI commands using cobra.
package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/keith-hung/azuredevops-cli/internal/client"
	"github.com/keith-hung/azuredevops-cli/internal/types"
	"github.com/spf13/cobra"
)

// GlobalFlags holds the parsed global flags.
type GlobalFlags struct {
	BaseURL    string
	Collection string
	Domain     string
	Username   string
	Password   string
	Project    string
	Repo       string
	APIVersion string
	Pretty     bool
	PassStdin  bool
	Insecure   bool
}

var gf GlobalFlags

var rootCmd = &cobra.Command{
	Use:           "azuredevops",
	Short:         "Azure DevOps Server management CLI",
	SilenceUsage:  true,
	SilenceErrors: true,
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Load password from env var first.
		gf.Password = envOrDefault("AZDO_PASSWORD", "")

		// Override with stdin if requested.
		if gf.PassStdin {
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				gf.Password = strings.TrimSpace(scanner.Text())
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
	rootCmd.PersistentFlags().StringVar(&gf.BaseURL, "base-url", envOrDefault("AZDO_BASE_URL", ""), "Azure DevOps Server base URL")
	rootCmd.PersistentFlags().StringVar(&gf.Collection, "collection", envOrDefault("AZDO_COLLECTION", ""), "Collection name")
	rootCmd.PersistentFlags().StringVar(&gf.Domain, "domain", envOrDefault("AZDO_DOMAIN", ""), "AD domain name")
	rootCmd.PersistentFlags().StringVar(&gf.Username, "username", envOrDefault("AZDO_USERNAME", ""), "Username")
	rootCmd.PersistentFlags().StringVar(&gf.Project, "project", envOrDefault("AZDO_PROJECT", ""), "Default project name")
	rootCmd.PersistentFlags().StringVar(&gf.Repo, "repo", envOrDefault("AZDO_REPO", ""), "Default repository name")
	rootCmd.PersistentFlags().StringVar(&gf.APIVersion, "api-version", envOrDefault("AZDO_API_VERSION", "5.0-preview.1"), "API version")
	rootCmd.PersistentFlags().BoolVar(&gf.PassStdin, "pass-stdin", false, "Read password from stdin")
	rootCmd.PersistentFlags().BoolVar(&gf.Insecure, "insecure", envOrBool("AZDO_INSECURE", false), "Skip TLS certificate verification")
	rootCmd.PersistentFlags().BoolVar(&gf.Pretty, "pretty", false, "Pretty-print JSON output")
}

// NewClient creates a client.Client from global flags.
func NewClient(gf *GlobalFlags) *client.Client {
	if gf.BaseURL == "" {
		ExitError("--base-url or AZDO_BASE_URL is required", 2)
	}
	if gf.Collection == "" {
		ExitError("--collection or AZDO_COLLECTION is required", 2)
	}
	if gf.Username == "" {
		ExitError("--username or AZDO_USERNAME is required", 2)
	}
	if gf.Password == "" {
		ExitError("AZDO_PASSWORD or --pass-stdin is required", 2)
	}

	// Prepend domain if set and username doesn't already contain one.
	username := gf.Username
	if gf.Domain != "" && !strings.Contains(username, `\`) {
		username = gf.Domain + `\` + username
	}
	return client.New(gf.BaseURL, gf.Collection, username, gf.Password, gf.APIVersion, gf.Insecure)
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
	case containsAny(lower, "invalid", "format", "required", "validation", "not found"):
		code = 3
	case containsAny(lower, "config", "environment", "auth", "password", "username", "forbidden", "401", "403"):
		code = 2
	}
	ExitError(msg, code)
}

// VoteLabel converts a vote integer to a human-readable label.
func VoteLabel(vote int) string {
	switch vote {
	case 10:
		return "approved"
	case 5:
		return "approved with suggestions"
	case 0:
		return "no vote"
	case -5:
		return "waiting for author"
	case -10:
		return "rejected"
	default:
		return fmt.Sprintf("unknown (%d)", vote)
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envOrBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v == "1" || strings.EqualFold(v, "true")
}

func isGUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
