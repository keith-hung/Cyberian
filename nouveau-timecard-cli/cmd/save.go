package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/keith-hung/nouveau-timecard-cli/internal/types"
	"github.com/spf13/cobra"
)

var saveCmd = &cobra.Command{
	Use:   "save",
	Short: "Save timecard records as a DRAFT (never submits for approval)",
	Long: `Save one or more timecard records as a draft.

The save is atomic: existing records for the month are preserved and your
records are overlaid on top, then the whole month is posted as a draft.
Set hours to 0 to clear a day. This command NEVER submits for approval.

--records is a JSON array of:
  {"project_id":90001,"activity_id":80002,"date":"2026-05-04","hours":8,"description":"work","overtime":false}`,
	Run: func(cmd *cobra.Command, args []string) {
		recordsJSON, _ := cmd.Flags().GetString("records")
		if recordsJSON == "" {
			ExitError("--records is required", 3)
		}

		var records []types.SaveRecordInput
		if err := json.Unmarshal([]byte(recordsJSON), &records); err != nil {
			ExitError(fmt.Sprintf("invalid --records JSON: %v", err), 3)
		}
		if len(records) == 0 {
			ExitError("--records must contain at least one record", 3)
		}

		// Derive year/month from the records (all must share one month).
		year, month, err := monthFromRecords(records)
		if err != nil {
			ExitError(err.Error(), 3)
		}

		sess, err := NewSession(&gf)
		if err != nil {
			ExitError(err.Error(), 1)
		}

		applied, err := sess.SaveDraft(year, month, records)
		if err != nil {
			ExitError(err.Error(), classifyError(err))
		}

		OutputJSON(types.SaveOutput{
			Success:      true,
			Message:      fmt.Sprintf("Draft saved for %04d-%02d", year, month),
			AppliedCount: applied,
		}, gf.Pretty)
	},
}

func init() {
	saveCmd.Flags().String("records", "", "JSON array of records (required)")
	rootCmd.AddCommand(saveCmd)
}

// monthFromRecords validates that all records fall in the same year/month and
// returns it.
func monthFromRecords(records []types.SaveRecordInput) (int, int, error) {
	year, month := 0, 0
	for i, r := range records {
		y, m, err := resolveYearMonth(r.Date, 0, 0)
		if err != nil {
			return 0, 0, fmt.Errorf("record %d: %v", i, err)
		}
		if year == 0 {
			year, month = y, m
			continue
		}
		if y != year || m != month {
			return 0, 0, fmt.Errorf("all records must be in the same month (got %04d-%02d and %04d-%02d)", year, month, y, m)
		}
	}
	return year, month, nil
}
