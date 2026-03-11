package cmd

import (
	"fmt"
	"math"
	"strconv"

	"github.com/keith-hung/timecard-cli/internal/parser"
	"github.com/keith-hung/timecard-cli/internal/types"
	"github.com/spf13/cobra"
)

var summaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Display summary statistics for a week",
	Run: func(cmd *cobra.Command, args []string) {
		date, _ := cmd.Flags().GetString("date")
		if date == "" {
			ExitError("--date is required", 3)
		}

		sess, err := NewSession(&gf)
		if err != nil {
			ExitError(err.Error(), 1)
		}
		if err := sess.EnsureAuth(); err != nil {
			ExitError("Authentication failed: "+err.Error(), 2)
		}

		data, err := sess.FetchTimesheetData(date)
		if err != nil {
			ExitError("Failed to get timesheet: "+err.Error(), 1)
		}

		weekStart, weekEnd, _ := computeWeekRange(date)
		mapping := parser.BuildIndexMapping(data.Activities)

		// Build project name map
		projectNameMap := make(map[string]string)
		for _, p := range data.Projects {
			projectNameMap[p.ID] = p.Name
		}

		// Calculate daily totals and project breakdown
		dailyTotals := make(map[string]float64)
		projectBreakdown := make(map[string]float64)
		activeProjects := make(map[string]bool)

		for _, label := range dayLabels {
			dailyTotals[label] = 0
		}

		// Group by row to count active entries
		type rowKey struct{ pi, ai int }
		activeRows := make(map[rowKey]bool)

		for _, entry := range data.TimeEntries {
			if entry.Duration == "" {
				continue
			}
			v, err := strconv.ParseFloat(entry.Duration, 64)
			if err != nil || v == 0 {
				continue
			}

			if entry.DayIndex >= 0 && entry.DayIndex < 6 {
				dailyTotals[dayLabels[entry.DayIndex]] += v
			}

			projectID := mapping.ProjectIndexToID[entry.ProjectIndex]
			projectName := projectNameMap[projectID]
			if projectName == "" {
				projectName = fmt.Sprintf("Project-%s", projectID)
			}
			projectBreakdown[projectName] += v
			activeProjects[projectID] = true
			activeRows[rowKey{entry.ProjectIndex, entry.ActivityIndex}] = true
		}

		// Statistics
		var totalHours float64
		maxDaily := 0.0
		minDaily := math.MaxFloat64
		workingDays := 0

		for _, label := range dayLabels {
			h := dailyTotals[label]
			totalHours += h
			if h > maxDaily {
				maxDaily = h
			}
			if h < minDaily {
				minDaily = h
			}
			if h > 0 {
				workingDays++
			}
		}
		if minDaily == math.MaxFloat64 {
			minDaily = 0
		}

		OutputJSON(types.SummaryOutput{
			WeekStart:        weekStart,
			WeekEnd:          weekEnd,
			TotalHours:       totalHours,
			ActiveEntries:    len(activeRows),
			DailyTotals:      dailyTotals,
			ProjectBreakdown: projectBreakdown,
			Statistics: types.SummaryStats{
				MaxDailyHours:  maxDaily,
				MinDailyHours:  minDaily,
				WorkingDays:    workingDays,
				UniqueProjects: len(activeProjects),
			},
		}, gf.Pretty)
	},
}

func init() {
	summaryCmd.Flags().String("date", "", "Target date YYYY-MM-DD (required)")
	rootCmd.AddCommand(summaryCmd)
}
