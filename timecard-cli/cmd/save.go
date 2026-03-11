package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/keith-hung/timecard-cli/internal/types"
	"github.com/spf13/cobra"
)

var saveCmd = &cobra.Command{
	Use:   "save",
	Short: "Save timesheet changes (draft only)",
	Run: func(cmd *cobra.Command, args []string) {
		date, _ := cmd.Flags().GetString("date")
		entriesJSON, _ := cmd.Flags().GetString("entries")
		hoursJSON, _ := cmd.Flags().GetString("hours")
		notesJSON, _ := cmd.Flags().GetString("notes")

		if date == "" {
			ExitError("--date is required", 3)
		}

		// At least one change set must be provided
		if entriesJSON == "" && hoursJSON == "" && notesJSON == "" {
			ExitError("at least one of --entries, --hours, --notes is required", 3)
		}

		// Parse JSON inputs
		var entries []types.SaveEntryInput
		var hours []types.SaveHoursInput
		var notes []types.SaveNotesInput

		if entriesJSON != "" {
			if err := json.Unmarshal([]byte(entriesJSON), &entries); err != nil {
				ExitError(fmt.Sprintf("invalid --entries JSON: %v", err), 3)
			}
		}
		if hoursJSON != "" {
			if err := json.Unmarshal([]byte(hoursJSON), &hours); err != nil {
				ExitError(fmt.Sprintf("invalid --hours JSON: %v", err), 3)
			}
		}
		if notesJSON != "" {
			if err := json.Unmarshal([]byte(notesJSON), &notes); err != nil {
				ExitError(fmt.Sprintf("invalid --notes JSON: %v", err), 3)
			}
		}

		sess, err := NewSession(&gf)
		if err != nil {
			ExitError(err.Error(), 1)
		}

		if err := sess.SaveTimesheet(date, entries, hours, notes); err != nil {
			// Determine exit code based on error type
			errMsg := err.Error()
			code := 1
			lower := strings.ToLower(errMsg)
			if containsSave(lower, "authentication", "login", "password", "username") {
				code = 2
			} else if containsSave(lower, "invalid", "forbidden", "must be", "exceeded") {
				code = 3
			} else if containsSave(lower, "network", "connection", "timeout") {
				code = 4
			}
			ExitError(errMsg, code)
		}

		totalChanges := len(entries) + len(hours) + len(notes)
		OutputJSON(types.SaveOutput{
			Success: true,
			Message: fmt.Sprintf("Timesheet saved successfully - applied %d changes", totalChanges),
		}, gf.Pretty)
	},
}

func init() {
	saveCmd.Flags().String("date", "", "Target date YYYY-MM-DD (required)")
	saveCmd.Flags().String("entries", "", "JSON array of entry configs")
	saveCmd.Flags().String("hours", "", "JSON array of hour values")
	saveCmd.Flags().String("notes", "", "JSON array of note values")
	rootCmd.AddCommand(saveCmd)
}

func containsSave(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
