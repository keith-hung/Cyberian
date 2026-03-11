package cmd

import (
	"fmt"
	"regexp"

	"github.com/keith-hung/wedaka-cli/internal/types"
	"github.com/spf13/cobra"
)

var checkWorkdayCmd = &cobra.Command{
	Use:   "check-workday",
	Short: "Check if a date is a work day",
	Run: func(cmd *cobra.Command, args []string) {
		date, _ := cmd.Flags().GetString("date")

		if date == "" {
			ExitError("--date is required", 3)
		}

		dateRegex := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
		if !dateRegex.MatchString(date) {
			ExitError(fmt.Sprintf("invalid date format: %s (expected YYYY-MM-DD)", date), 3)
		}

		if gf.EmpNo == "" {
			ExitError("--emp-no or WEDAKA_EMP_NO is required", 2)
		}

		c := NewClient(&gf)
		resp, err := c.GetDateType(gf.EmpNo, date)
		if err != nil {
			ExitErrorInfer("check work day: " + err.Error())
		}

		if !resp.Status {
			OutputJSON(types.CheckWorkdayOutput{
				Success: false,
				Date:    date,
				Message: resp.ErrorMessage,
			}, gf.Pretty)
			return
		}

		isWorkDay := resp.DateType == "1"
		OutputJSON(types.CheckWorkdayOutput{
			Success:     true,
			Date:        date,
			DateType:    resp.DateType,
			IsWorkDay:   isWorkDay,
			Description: dateTypeDescription(resp.DateType),
		}, gf.Pretty)
	},
}

func init() {
	checkWorkdayCmd.Flags().String("date", "", "Date in YYYY-MM-DD format (required)")
	rootCmd.AddCommand(checkWorkdayCmd)
}
