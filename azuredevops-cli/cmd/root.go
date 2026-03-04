// Package cmd implements CLI subcommand routing and global flag handling.
package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/keith-hung/azuredevops-cli/internal/client"
	"github.com/keith-hung/azuredevops-cli/internal/types"
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
}

// ParseGlobalFlags extracts global flags from args. Returns parsed flags and remaining args.
func ParseGlobalFlags(args []string) (*GlobalFlags, []string) {
	gf := &GlobalFlags{
		BaseURL:    envOrDefault("AZDO_BASE_URL", ""),
		Collection: envOrDefault("AZDO_COLLECTION", ""),
		Domain:     envOrDefault("AZDO_DOMAIN", ""),
		Username:   envOrDefault("AZDO_USERNAME", ""),
		Password:   envOrDefault("AZDO_PASSWORD", ""),
		Project:    envOrDefault("AZDO_PROJECT", ""),
		Repo:       envOrDefault("AZDO_REPO", ""),
		APIVersion: envOrDefault("AZDO_API_VERSION", "5.0"),
	}

	var remaining []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--base-url" && i+1 < len(args):
			i++
			gf.BaseURL = args[i]
		case arg == "--collection" && i+1 < len(args):
			i++
			gf.Collection = args[i]
		case arg == "--domain" && i+1 < len(args):
			i++
			gf.Domain = args[i]
		case arg == "--username" && i+1 < len(args):
			i++
			gf.Username = args[i]
		case arg == "--project" && i+1 < len(args):
			i++
			gf.Project = args[i]
		case arg == "--repo" && i+1 < len(args):
			i++
			gf.Repo = args[i]
		case arg == "--api-version" && i+1 < len(args):
			i++
			gf.APIVersion = args[i]
		case arg == "--pass-stdin":
			gf.PassStdin = true
		case arg == "--pretty":
			gf.Pretty = true
		default:
			remaining = append(remaining, arg)
		}
	}

	if gf.PassStdin {
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			gf.Password = strings.TrimSpace(scanner.Text())
		}
	}

	return gf, remaining
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
	return client.New(gf.BaseURL, gf.Collection, username, gf.Password, gf.APIVersion)
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

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
