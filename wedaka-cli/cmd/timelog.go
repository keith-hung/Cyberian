package cmd

import (
	"github.com/keith-hung/wedaka-cli/internal/types"
	"github.com/spf13/cobra"
)

var timelogCmd = &cobra.Command{
	Use:   "timelog",
	Short: "View attendance time log",
	Run: func(cmd *cobra.Command, args []string) {
		month, _ := cmd.Flags().GetInt("month")
		year, _ := cmd.Flags().GetInt("year")

		if month < 1 || month > 12 {
			ExitError("--month is required and must be 1-12", 3)
		}
		if year < 2020 || year > 2030 {
			ExitError("--year is required and must be 2020-2030", 3)
		}
		if gf.Username == "" {
			ExitError("--username or WEDAKA_USERNAME is required", 2)
		}

		c := NewClient(&gf)
		resp, err := c.SearchTimelog(gf.Username, month, year)
		if err != nil {
			ExitErrorInfer("search timelog: " + err.Error())
		}

		if !resp.Status {
			ExitErrorInfer("API error: " + resp.ErrorMessage)
		}

		records := make([]types.TimelogRecord, 0, len(resp.TimeLog))
		var clockIns, clockOuts, leaves int

		for _, r := range resp.TimeLog {
			desc := workItemDescription(r.WorkItem)
			isLeave := r.WorkItem != nil && *r.WorkItem == "2"

			if r.WorkItem != nil {
				switch *r.WorkItem {
				case "1":
					clockIns++
				case "2":
					leaves++
				case "4":
					clockOuts++
				}
			}

			records = append(records, types.TimelogRecord{
				DateType:    r.DateType,
				LeaveHours:  r.LeaveHours,
				Memo:        r.Memo,
				WorkItem:    r.WorkItem,
				WorkTime:    r.WorkTime,
				WorkType:    r.WorkType,
				WorkDate:    r.WorkDate,
				Description: desc,
				IsLeave:     isLeave,
			})
		}

		OutputJSON(types.TimelogOutput{
			Success:      true,
			TotalRecords: len(records),
			Summary: types.TimelogSummary{
				ClockIns:  clockIns,
				ClockOuts: clockOuts,
				Leaves:    leaves,
			},
			Records: records,
		}, gf.Pretty)
	},
}

func init() {
	timelogCmd.Flags().Int("month", 0, "Month (1-12)")
	timelogCmd.Flags().Int("year", 0, "Year (2020-2030)")
	rootCmd.AddCommand(timelogCmd)
}

func workItemDescription(wi *string) string {
	if wi == nil {
		return "Unknown"
	}
	switch *wi {
	case "1":
		return "Clock In"
	case "2":
		return "Leave"
	case "4":
		return "Clock Out"
	default:
		return "Unknown (" + *wi + ")"
	}
}
