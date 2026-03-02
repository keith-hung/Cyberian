package cmd

import (
	"encoding/json"
	"flag"
	"fmt"

	"github.com/keith-hung/timecard-cli/internal/types"
)

// RunSave performs an atomic save operation.
func RunSave(gf *GlobalFlags, args []string) {
	fs := flag.NewFlagSet("save", flag.ContinueOnError)
	date := fs.String("date", "", "Target date YYYY-MM-DD (required)")
	entriesJSON := fs.String("entries", "", "JSON array of entry configs")
	hoursJSON := fs.String("hours", "", "JSON array of hour values")
	notesJSON := fs.String("notes", "", "JSON array of note values")
	if err := fs.Parse(args); err != nil {
		ExitError(err.Error(), 3)
	}
	if *date == "" {
		ExitError("--date is required", 3)
	}

	// At least one change set must be provided
	if *entriesJSON == "" && *hoursJSON == "" && *notesJSON == "" {
		ExitError("at least one of --entries, --hours, --notes is required", 3)
	}

	// Parse JSON inputs
	var entries []types.SaveEntryInput
	var hours []types.SaveHoursInput
	var notes []types.SaveNotesInput

	if *entriesJSON != "" {
		if err := json.Unmarshal([]byte(*entriesJSON), &entries); err != nil {
			ExitError(fmt.Sprintf("invalid --entries JSON: %v", err), 3)
		}
	}
	if *hoursJSON != "" {
		if err := json.Unmarshal([]byte(*hoursJSON), &hours); err != nil {
			ExitError(fmt.Sprintf("invalid --hours JSON: %v", err), 3)
		}
	}
	if *notesJSON != "" {
		if err := json.Unmarshal([]byte(*notesJSON), &notes); err != nil {
			ExitError(fmt.Sprintf("invalid --notes JSON: %v", err), 3)
		}
	}

	sess, err := NewSession(gf)
	if err != nil {
		ExitError(err.Error(), 1)
	}

	if err := sess.SaveTimesheet(*date, entries, hours, notes); err != nil {
		// Determine exit code based on error type
		errMsg := err.Error()
		code := 1
		if contains(errMsg, "authentication", "login", "password", "username") {
			code = 2
		} else if contains(errMsg, "invalid", "forbidden", "must be", "exceeded") {
			code = 3
		} else if contains(errMsg, "network", "connection", "timeout") {
			code = 4
		}
		ExitError(errMsg, code)
	}

	totalChanges := len(entries) + len(hours) + len(notes)
	OutputJSON(types.SaveOutput{
		Success: true,
		Message: fmt.Sprintf("Timesheet saved successfully - applied %d changes", totalChanges),
	}, gf.Pretty)
}

func contains(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}
