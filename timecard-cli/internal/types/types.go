// Package types defines shared data structures for the timecard CLI.
package types

// ActivityEntry represents an activity from the TimeCard activity list.
// Parsed from act.append() or activityList.append() calls in HTML.
type ActivityEntry struct {
	ProjectID  string `json:"project_id"`
	Name       string `json:"name"`
	IsBottom   string `json:"is_bottom"`   // "true" or "false"
	UID        string `json:"uid"`         // Unique identifier
	Progress   string `json:"progress"`    // Progress percentage
	ActivityID string `json:"activity_id"` // Only present in 6-param format; empty otherwise
}

// TimearrayEntry represents a single cell in the timearray grid.
// Parsed from timearray[i][j][k] = "duration$status$note$progress".
type TimearrayEntry struct {
	ProjectIndex  int    `json:"project_index"`
	ActivityIndex int    `json:"activity_index"`
	DayIndex      int    `json:"day_index"`
	Duration      string `json:"duration"`
	Status        string `json:"status"`
	Note          string `json:"note"`
	Progress      string `json:"progress"`
}

// ProjectOption represents a <option> in the project select element.
type ProjectOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ErrorInfo holds parsed error page details.
type ErrorInfo struct {
	MainMessage      string `json:"main_message"`
	ExceptionType    string `json:"exception_type"`
	ExceptionMessage string `json:"exception_message"`
}

// IndexMapping maps timearray indices to project/activity metadata.
type IndexMapping struct {
	// ProjectIndexToID maps timearray project index → project ID.
	ProjectIndexToID map[int]string
	// ActivityByIndex maps "projectIndex_activityIndex" → ActivityEntry.
	ActivityByIndex map[string]ActivityEntry
}

// --- CLI output types ---

// ProjectsOutput is the JSON output for the `projects` command.
type ProjectsOutput struct {
	Projects []ProjectOption `json:"projects"`
	Count    int             `json:"count"`
}

// ActivitiesOutput is the JSON output for the `activities` command.
type ActivitiesOutput struct {
	ProjectID  string             `json:"project_id"`
	Activities []ActivityOutEntry `json:"activities"`
	Count      int                `json:"count"`
}

// ActivityOutEntry is the agent-facing activity representation.
type ActivityOutEntry struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"` // "isBottom$uid$pid$progress"
}

// TimesheetOutput is the JSON output for the `timesheet` command.
type TimesheetOutput struct {
	WeekStart string           `json:"week_start"`
	WeekEnd   string           `json:"week_end"`
	Status    string           `json:"status"`
	Entries   []TimesheetEntry `json:"entries"`
}

// TimesheetEntry represents one row in the timesheet.
type TimesheetEntry struct {
	Index       int               `json:"index"`
	Project     ProjectOption     `json:"project"`
	Activity    ActivityOutEntry  `json:"activity"`
	DailyHours  map[string]float64 `json:"daily_hours"`
	DailyStatus map[string]string  `json:"daily_status"`
	DailyNotes  map[string]string  `json:"daily_notes"`
}

// SummaryOutput is the JSON output for the `summary` command.
type SummaryOutput struct {
	WeekStart        string             `json:"week_start"`
	WeekEnd          string             `json:"week_end"`
	TotalHours       float64            `json:"total_hours"`
	ActiveEntries    int                `json:"active_entries"`
	DailyTotals      map[string]float64 `json:"daily_totals"`
	ProjectBreakdown map[string]float64 `json:"project_breakdown"`
	Statistics       SummaryStats       `json:"statistics"`
}

// SummaryStats holds computed statistics for the summary command.
type SummaryStats struct {
	MaxDailyHours  float64 `json:"max_daily_hours"`
	MinDailyHours  float64 `json:"min_daily_hours"`
	WorkingDays    int     `json:"working_days"`
	UniqueProjects int     `json:"unique_projects"`
}

// SaveOutput is the JSON output for the `save` command.
type SaveOutput struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// VersionOutput is the JSON output for the `version` command.
type VersionOutput struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
}

// ErrorOutput is the JSON error output written to stderr.
type ErrorOutput struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

// --- Save input types ---

// SaveEntryInput is a single entry in the save --entries JSON.
type SaveEntryInput struct {
	EntryIndex    int    `json:"entry_index"`
	ProjectID     string `json:"project_id"`
	ActivityValue string `json:"activity_value"`
}

// SaveHoursInput is a single entry in the save --hours JSON.
type SaveHoursInput struct {
	EntryIndex int     `json:"entry_index"`
	Date       string  `json:"date"`
	Hours      float64 `json:"hours"`
}

// SaveNotesInput is a single entry in the save --notes JSON.
type SaveNotesInput struct {
	EntryIndex int    `json:"entry_index"`
	Date       string `json:"date"`
	Note       string `json:"note"`
}
