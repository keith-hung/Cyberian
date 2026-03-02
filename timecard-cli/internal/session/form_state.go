package session

import (
	"fmt"
	"strconv"

	"github.com/keith-hung/timecard-cli/internal/parser"
	"github.com/keith-hung/timecard-cli/internal/types"
)

// ReconstructFormState rebuilds the full form state as it would be after
// JavaScript initialization on the timesheet page.
// Combines hidden inputs, timearray data, and activity list data.
func ReconstructFormState(
	html string,
	activities []types.ActivityEntry,
	timeEntries []types.TimearrayEntry,
	overtimeEntries []types.TimearrayEntry,
) map[string]string {
	form := parser.ParseHiddenInputs(html)
	mapping := parser.BuildIndexMapping(activities)

	// --- Normal rows ---

	// Group time entries by (projectIndex, activityIndex) to find unique rows.
	// Use ordered slice to maintain insertion order (Go maps don't guarantee order).
	type rowData struct {
		projectIndex  int
		activityIndex int
		entries       []types.TimearrayEntry
	}
	var normalRowKeys []string
	normalRows := make(map[string]*rowData)

	for _, entry := range timeEntries {
		key := fmt.Sprintf("%d_%d", entry.ProjectIndex, entry.ActivityIndex)
		if _, ok := normalRows[key]; !ok {
			normalRowKeys = append(normalRowKeys, key)
			normalRows[key] = &rowData{
				projectIndex:  entry.ProjectIndex,
				activityIndex: entry.ActivityIndex,
			}
		}
		normalRows[key].entries = append(normalRows[key].entries, entry)
	}

	for rowIndex, key := range normalRowKeys {
		row := normalRows[key]
		projectID := mapping.ProjectIndexToID[row.projectIndex]
		actEntry, hasAct := mapping.ActivityByIndex[key]

		form[fmt.Sprintf("project%d", rowIndex)] = projectID

		if hasAct {
			form[fmt.Sprintf("activity%d", rowIndex)] = fmt.Sprintf(
				"%s$%s$%s$%s", actEntry.IsBottom, actEntry.UID, actEntry.ProjectID, actEntry.Progress,
			)
			form[fmt.Sprintf("actprogress%d", rowIndex)] = actEntry.Progress
		} else {
			form[fmt.Sprintf("activity%d", rowIndex)] = ""
			form[fmt.Sprintf("actprogress%d", rowIndex)] = ""
		}

		for _, entry := range row.entries {
			if entry.Duration != "" {
				form[fmt.Sprintf("record%d_%d", rowIndex, entry.DayIndex)] = entry.Duration
			}
			if entry.Note != "" {
				form[fmt.Sprintf("note%d_%d", rowIndex, entry.DayIndex)] = entry.Note
			}
			if entry.Progress != "" {
				form[fmt.Sprintf("progress%d_%d", rowIndex, entry.DayIndex)] = entry.Progress
			}
		}
	}

	normalRowCount := len(normalRowKeys)

	// Initialize remaining normal rows as empty
	for i := normalRowCount; i < 25; i++ {
		if _, ok := form[fmt.Sprintf("project%d", i)]; !ok {
			form[fmt.Sprintf("project%d", i)] = ""
		}
		if _, ok := form[fmt.Sprintf("activity%d", i)]; !ok {
			form[fmt.Sprintf("activity%d", i)] = ""
		}
		for k := 0; k < 7; k++ {
			if _, ok := form[fmt.Sprintf("record%d_%d", i, k)]; !ok {
				form[fmt.Sprintf("record%d_%d", i, k)] = ""
			}
		}
	}

	// --- Overtime rows ---

	var overRowKeys []string
	overRows := make(map[string]*rowData)

	for _, entry := range overtimeEntries {
		key := fmt.Sprintf("%d_%d", entry.ProjectIndex, entry.ActivityIndex)
		if _, ok := overRows[key]; !ok {
			overRowKeys = append(overRowKeys, key)
			overRows[key] = &rowData{
				projectIndex:  entry.ProjectIndex,
				activityIndex: entry.ActivityIndex,
			}
		}
		overRows[key].entries = append(overRows[key].entries, entry)
	}

	for overRowIndex, key := range overRowKeys {
		row := overRows[key]
		projectID := mapping.ProjectIndexToID[row.projectIndex]
		actEntry, hasAct := mapping.ActivityByIndex[key]

		form[fmt.Sprintf("overproject%d", overRowIndex)] = projectID
		if hasAct {
			form[fmt.Sprintf("overactivity%d", overRowIndex)] = fmt.Sprintf(
				"%s$%s$%s$%s", actEntry.IsBottom, actEntry.UID, actEntry.ProjectID, actEntry.Progress,
			)
		} else {
			form[fmt.Sprintf("overactivity%d", overRowIndex)] = ""
		}

		for _, entry := range row.entries {
			if entry.Duration != "" {
				form[fmt.Sprintf("overrecord%d_%d", overRowIndex, entry.DayIndex)] = entry.Duration
			}
			if entry.Note != "" {
				form[fmt.Sprintf("overnote%d_%d", overRowIndex, entry.DayIndex)] = entry.Note
			}
			if entry.Progress != "" {
				form[fmt.Sprintf("overprogress%d_%d", overRowIndex, entry.DayIndex)] = entry.Progress
			}
		}
	}

	overRowCount := len(overRowKeys)

	// Initialize remaining overtime rows
	for i := overRowCount; i < 25; i++ {
		if _, ok := form[fmt.Sprintf("overproject%d", i)]; !ok {
			form[fmt.Sprintf("overproject%d", i)] = ""
		}
		if _, ok := form[fmt.Sprintf("overactivity%d", i)]; !ok {
			form[fmt.Sprintf("overactivity%d", i)] = ""
		}
		for k := 0; k < 7; k++ {
			if _, ok := form[fmt.Sprintf("overrecord%d_%d", i, k)]; !ok {
				form[fmt.Sprintf("overrecord%d_%d", i, k)] = ""
			}
		}
	}

	// --- Calculate daily subtotals ---
	// weekinfo_deal.jsp reads norTotal{k} via Float.parseFloat — NPE if missing!
	norTotals := [7]float64{}
	oveTotals := [7]float64{}

	for _, entry := range timeEntries {
		if entry.Duration != "" {
			if v, err := strconv.ParseFloat(entry.Duration, 64); err == nil {
				norTotals[entry.DayIndex] += v
			}
		}
	}
	for _, entry := range overtimeEntries {
		if entry.Duration != "" {
			if v, err := strconv.ParseFloat(entry.Duration, 64); err == nil {
				oveTotals[entry.DayIndex] += v
			}
		}
	}
	for k := 0; k < 7; k++ {
		form[fmt.Sprintf("norTotal%d", k)] = strconv.FormatFloat(norTotals[k], 'f', -1, 64)
		form[fmt.Sprintf("oveTotal%d", k)] = strconv.FormatFloat(oveTotals[k], 'f', -1, 64)
		form[fmt.Sprintf("colTotal%d", k)] = strconv.FormatFloat(norTotals[k]+oveTotals[k], 'f', -1, 64)
	}

	return form
}
