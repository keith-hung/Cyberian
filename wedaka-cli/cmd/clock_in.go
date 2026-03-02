package cmd

import (
	"flag"
	"fmt"
	"regexp"
	"time"

	"github.com/keith-hung/wedaka-cli/internal/types"
)

// RunClockIn handles the clock-in subcommand.
func RunClockIn(gf *GlobalFlags, args []string) {
	runClock(gf, args, "in")
}

// RunClockOut handles the clock-out subcommand.
func RunClockOut(gf *GlobalFlags, args []string) {
	runClock(gf, args, "out")
}

func runClock(gf *GlobalFlags, args []string, direction string) {
	fs := flag.NewFlagSet("clock-"+direction, flag.ContinueOnError)
	dateFlag := fs.String("date", "", "Date in YYYY-MM-DD format (default: today)")
	timeFlag := fs.String("time", "", "Time in HH:MM:SS format (default: now)")
	noteFlag := fs.String("note", "", "Additional notes")
	if err := fs.Parse(args); err != nil {
		ExitError(fmt.Sprintf("invalid flags: %v", err), 3)
	}

	if gf.EmpNo == "" {
		ExitError("--emp-no or WEDAKA_EMP_NO is required", 2)
	}
	if gf.Username == "" {
		ExitError("--username or WEDAKA_USERNAME is required", 2)
	}

	now := time.Now()
	dateRegex := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	timeRegex := regexp.MustCompile(`^\d{2}:\d{2}:\d{2}$`)

	// Resolve date
	targetDate := now.Format("2006-01-02")
	if *dateFlag != "" {
		if !dateRegex.MatchString(*dateFlag) {
			ExitError(fmt.Sprintf("invalid date format: %s (expected YYYY-MM-DD)", *dateFlag), 3)
		}
		targetDate = *dateFlag
	}

	// Reject future dates
	today := now.Format("2006-01-02")
	if targetDate > today {
		ExitError(fmt.Sprintf("cannot clock for future date (%s); only today or past dates are allowed", targetDate), 3)
	}

	// Check work day
	c := NewClient(gf)
	dtResp, err := c.GetDateType(gf.EmpNo, targetDate)
	if err != nil {
		ExitErrorInfer(fmt.Sprintf("check work day: %v", err))
	}
	if !dtResp.Status {
		ExitErrorInfer(fmt.Sprintf("check work day failed: %s", dtResp.ErrorMessage))
	}
	if dtResp.DateType != "1" {
		desc := dateTypeDescription(dtResp.DateType)
		ExitError(fmt.Sprintf("date %s is not a work day (%s); clocking is only allowed on work days", targetDate, desc), 3)
	}

	// Resolve time
	targetTime := now.Format("15:04:05")
	if *timeFlag != "" {
		if !timeRegex.MatchString(*timeFlag) {
			ExitError(fmt.Sprintf("invalid time format: %s (expected HH:MM:SS)", *timeFlag), 3)
		}
		targetTime = *timeFlag
	}

	// Build datetime in YYYY/MM/DD HH:MM:SS format (API requirement)
	workTime := targetDate[0:4] + "/" + targetDate[5:7] + "/" + targetDate[8:10] + " " + targetTime

	var workItem, workType string
	if direction == "in" {
		workItem = "1" // clock in
		workType = "1"
	} else {
		workItem = "4" // clock out
		workType = "2"
	}

	payload := &types.InsertTimeLogPayload{
		UserName: gf.Username,
		WorkTimeLogData: []types.WorkTimeLogEntry{
			{
				DateType:   "1",
				LeaveHours: 0,
				Memo:       *noteFlag,
				WorkItem:   workItem,
				WorkTime:   workTime,
				WorkType:   workType,
			},
		},
	}

	resp, err := c.InsertTimeLog(payload)
	if err != nil {
		ExitErrorInfer(fmt.Sprintf("insert time log: %v", err))
	}

	if !resp.Status {
		ExitErrorInfer(fmt.Sprintf("API error: %s", resp.ErrorMessage))
	}

	OutputJSON(types.ClockResult{
		Success: true,
		LogId:   resp.LogId,
		LogTime: resp.LogTime,
	}, gf.Pretty)
}

func dateTypeDescription(dt string) string {
	switch dt {
	case "1":
		return "work day"
	case "2":
		return "leave day"
	case "3":
		return "holiday"
	default:
		return "unknown type " + dt
	}
}
