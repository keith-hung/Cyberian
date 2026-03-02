package cmd

import (
	"flag"
	"fmt"

	"github.com/keith-hung/wedaka-cli/internal/types"
)

// RunTimelog handles the timelog subcommand.
func RunTimelog(gf *GlobalFlags, args []string) {
	fs := flag.NewFlagSet("timelog", flag.ContinueOnError)
	monthFlag := fs.Int("month", 0, "Month (1-12)")
	yearFlag := fs.Int("year", 0, "Year (2020-2030)")
	if err := fs.Parse(args); err != nil {
		ExitError(fmt.Sprintf("invalid flags: %v", err), 3)
	}

	if *monthFlag < 1 || *monthFlag > 12 {
		ExitError("--month is required and must be 1-12", 3)
	}
	if *yearFlag < 2020 || *yearFlag > 2030 {
		ExitError("--year is required and must be 2020-2030", 3)
	}
	if gf.Username == "" {
		ExitError("--username or WEDAKA_USERNAME is required", 2)
	}

	c := NewClient(gf)
	resp, err := c.SearchTimelog(gf.Username, *monthFlag, *yearFlag)
	if err != nil {
		ExitErrorInfer(fmt.Sprintf("search timelog: %v", err))
	}

	if !resp.Status {
		ExitErrorInfer(fmt.Sprintf("API error: %s", resp.ErrorMessage))
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
