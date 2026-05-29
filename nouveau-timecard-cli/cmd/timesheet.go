package cmd

import (
	"fmt"

	"github.com/keith-hung/nouveau-timecard-cli/internal/types"
	"github.com/spf13/cobra"
)

var timesheetCmd = &cobra.Command{
	Use:   "timesheet",
	Short: "Show the timecard fill status for a month",
	Run: func(cmd *cobra.Command, args []string) {
		year, month, err := monthFromFlags(cmd)
		if err != nil {
			ExitError(err.Error(), 3)
		}

		sess, err := NewSession(&gf)
		if err != nil {
			ExitError(err.Error(), 1)
		}

		page, err := sess.FetchPageData(year, month)
		if err != nil {
			ExitError("Failed to get timesheet: "+err.Error(), classifyError(err))
		}

		OutputJSON(types.TimesheetOutput{
			Year:     year,
			Month:    month,
			Entries:  buildEntries(year, month, page.TimeRows),
			Overtime: buildEntries(year, month, page.OvertimeRows),
		}, gf.Pretty)
	},
}

func init() {
	addMonthFlags(timesheetCmd)
	rootCmd.AddCommand(timesheetCmd)
}

// buildEntries converts server rows into output entries, emitting only the days
// that actually have hours, a description, or an existing record.
func buildEntries(year, month int, rows []types.ServerRow) []types.TimesheetEntry {
	var out []types.TimesheetEntry
	for _, row := range rows {
		entry := types.TimesheetEntry{
			ProjectID:    row.ProjectID,
			ProjectName:  row.ProjectName,
			ActivityID:   row.ActivityID,
			ActivityName: row.ActivityName,
			Days:         map[string]types.DayValue{},
		}
		for i := range row.DailyHoursArray {
			day := i + 1
			var hours float64
			if row.DailyHoursArray[i] != nil {
				hours = *row.DailyHoursArray[i]
			}
			desc := ""
			if i < len(row.DailyDescriptionsArray) {
				desc = row.DailyDescriptionsArray[i]
			}
			recordID := 0
			if i < len(row.RecordIdsArray) && row.RecordIdsArray[i] != nil {
				recordID = *row.RecordIdsArray[i]
			}
			status := ""
			if i < len(row.RecordStatusesArray) {
				status = row.RecordStatusesArray[i]
			}
			if hours <= 0 && desc == "" && recordID <= 0 {
				continue
			}
			date := fmt.Sprintf("%04d-%02d-%02d", year, month, day)
			entry.Days[date] = types.DayValue{Hours: hours, Description: desc, Status: status, RecordID: recordID}
			entry.TotalHours += hours
		}
		out = append(out, entry)
	}
	return out
}
