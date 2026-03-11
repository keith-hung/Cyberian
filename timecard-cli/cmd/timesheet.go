package cmd

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/keith-hung/timecard-cli/internal/parser"
	"github.com/keith-hung/timecard-cli/internal/types"
	"github.com/spf13/cobra"
)

var dayLabels = [6]string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday"}

var timesheetCmd = &cobra.Command{
	Use:   "timesheet",
	Short: "Display timesheet data for a week",
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

		// Compute week range
		weekStart, weekEnd, weekDates := computeWeekRange(date)

		// Build index mapping
		mapping := parser.BuildIndexMapping(data.Activities)

		// Group entries by row
		type rowInfo struct {
			projectIndex  int
			activityIndex int
			entries       []types.TimearrayEntry
		}
		var rowKeys []string
		rows := make(map[string]*rowInfo)

		for _, entry := range data.TimeEntries {
			key := fmt.Sprintf("%d_%d", entry.ProjectIndex, entry.ActivityIndex)
			if _, ok := rows[key]; !ok {
				rowKeys = append(rowKeys, key)
				rows[key] = &rowInfo{
					projectIndex:  entry.ProjectIndex,
					activityIndex: entry.ActivityIndex,
				}
			}
			rows[key].entries = append(rows[key].entries, entry)
		}

		// Build project name map
		projectNameMap := make(map[string]string)
		for _, p := range data.Projects {
			projectNameMap[p.ID] = p.Name
		}

		// Build output entries
		var outEntries []types.TimesheetEntry
		for idx, key := range rowKeys {
			row := rows[key]
			projectID := mapping.ProjectIndexToID[row.projectIndex]
			actEntry, hasAct := mapping.ActivityByIndex[key]

			te := types.TimesheetEntry{
				Index: idx,
				Project: types.ProjectOption{
					ID:   projectID,
					Name: projectNameMap[projectID],
				},
				DailyHours:  make(map[string]float64),
				DailyStatus: make(map[string]string),
				DailyNotes:  make(map[string]string),
			}

			if hasAct {
				te.Activity = types.ActivityOutEntry{
					ID:   actEntry.UID,
					Name: actEntry.Name,
					Value: fmt.Sprintf("%s$%s$%s$%s",
						actEntry.IsBottom, actEntry.UID, actEntry.ProjectID, actEntry.Progress),
				}
			}

			// Initialize all days
			for _, label := range dayLabels {
				te.DailyHours[label] = 0
				te.DailyStatus[label] = "draft"
				te.DailyNotes[label] = ""
			}

			for _, entry := range row.entries {
				if entry.DayIndex >= 0 && entry.DayIndex < 6 {
					label := dayLabels[entry.DayIndex]
					if entry.Duration != "" {
						if v, err := strconv.ParseFloat(entry.Duration, 64); err == nil {
							te.DailyHours[label] = v
						}
					}
					if entry.Status != "" {
						te.DailyStatus[label] = entry.Status
					}
					if entry.Note != "" {
						te.DailyNotes[label] = entry.Note
					}
				}
			}

			outEntries = append(outEntries, te)
		}

		// Determine overall status
		status := "draft"
		for _, entry := range data.TimeEntries {
			if entry.Status == "submitted" || entry.Status == "approved" {
				status = entry.Status
				break
			}
		}

		OutputJSON(types.TimesheetOutput{
			WeekStart: weekStart,
			WeekEnd:   weekEnd,
			Status:    status,
			Entries:   outEntries,
		}, gf.Pretty)

		_ = weekDates // available for future use
	},
}

func init() {
	timesheetCmd.Flags().String("date", "", "Target date YYYY-MM-DD (required)")
	rootCmd.AddCommand(timesheetCmd)
}

// computeWeekRange returns (weekStart, weekEnd, dates[]) for a given date string.
func computeWeekRange(dateStr string) (string, string, []string) {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return dateStr, dateStr, nil
	}

	dow := int(t.Weekday())
	daysToMonday := dow - 1
	if dow == 0 {
		daysToMonday = 6
	}
	monday := t.AddDate(0, 0, -daysToMonday)

	dates := make([]string, 6)
	for i := 0; i < 6; i++ {
		d := monday.AddDate(0, 0, i)
		dates[i] = d.Format("2006-01-02")
	}

	saturday := monday.AddDate(0, 0, 5)
	return monday.Format("2006-01-02"), saturday.Format("2006-01-02"), dates
}

// computeWeekRangeForSummary also returns 7 days (Mon-Sun) for daily totals.
func computeWeekRangeForSummary(dateStr string) (monday time.Time, dates [7]string) {
	t, _ := time.Parse("2006-01-02", dateStr)
	dow := int(t.Weekday())
	daysToMonday := dow - 1
	if dow == 0 {
		daysToMonday = 6
	}
	monday = t.AddDate(0, 0, -daysToMonday)
	for i := 0; i < 7; i++ {
		dates[i] = monday.AddDate(0, 0, i).Format("2006-01-02")
	}
	return
}

// unused but matches TS interface
func isDateInWeek(date string, mondayDate time.Time) bool {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return false
	}
	diff := int(math.Round(t.Sub(mondayDate).Hours() / 24))
	return diff >= 0 && diff <= 5
}
