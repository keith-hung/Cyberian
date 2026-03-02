package cmd

import (
	"flag"
	"fmt"
	"regexp"

	"github.com/keith-hung/wedaka-cli/internal/types"
)

// RunCheckWorkday handles the check-workday subcommand.
func RunCheckWorkday(gf *GlobalFlags, args []string) {
	fs := flag.NewFlagSet("check-workday", flag.ContinueOnError)
	dateFlag := fs.String("date", "", "Date in YYYY-MM-DD format (required)")
	if err := fs.Parse(args); err != nil {
		ExitError(fmt.Sprintf("invalid flags: %v", err), 3)
	}

	if *dateFlag == "" {
		ExitError("--date is required", 3)
	}

	dateRegex := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	if !dateRegex.MatchString(*dateFlag) {
		ExitError(fmt.Sprintf("invalid date format: %s (expected YYYY-MM-DD)", *dateFlag), 3)
	}

	if gf.EmpNo == "" {
		ExitError("--emp-no or WEDAKA_EMP_NO is required", 2)
	}

	c := NewClient(gf)
	resp, err := c.GetDateType(gf.EmpNo, *dateFlag)
	if err != nil {
		ExitErrorInfer(fmt.Sprintf("check work day: %v", err))
	}

	if !resp.Status {
		OutputJSON(types.CheckWorkdayOutput{
			Success: false,
			Date:    *dateFlag,
			Message: resp.ErrorMessage,
		}, gf.Pretty)
		return
	}

	isWorkDay := resp.DateType == "1"
	OutputJSON(types.CheckWorkdayOutput{
		Success:     true,
		Date:        *dateFlag,
		DateType:    resp.DateType,
		IsWorkDay:   isWorkDay,
		Description: dateTypeDescription(resp.DateType),
	}, gf.Pretty)
}
