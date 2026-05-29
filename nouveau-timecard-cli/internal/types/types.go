// Package types defines shared data structures for the Nouveau Timecard CLI.
package types

// --- Parsed page data ---

// ProjectOption represents an <option> in the project <select> element.
type ProjectOption struct {
	ID   string `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

// Activity represents a bottom-level (selectable) activity for a project,
// parsed from the activityHtmlCache embedded in the timesheet page.
type Activity struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	Name      string `json:"name"` // Full hierarchical path, e.g. "Sample Project→Group→Task"
}

// ServerRow mirrors one entry in serverTimeRows / serverOvertimeRows JSON
// embedded in the timesheet page. Arrays are indexed 0..daysInMonth-1.
type ServerRow struct {
	ActivityID            int        `json:"activityId"`
	ProjectID             int        `json:"projectId"`
	ProjectName           string     `json:"projectName"`
	ProjectStatus         string     `json:"projectStatus"`
	ActivityActive        bool       `json:"activityActive"`
	ActivityName          string     `json:"activityName"`
	DailyHoursArray       []*float64 `json:"dailyHoursArray"`
	DailyDescriptionsArray []string  `json:"dailyDescriptionsArray"`
	RecordIdsArray        []*int     `json:"recordIdsArray"`
	RecordStatusesArray   []string   `json:"recordStatusesArray"`
	RecordDatesArray      []*string  `json:"recordDatesArray"`
}

// --- CLI output types ---

// ProjectsOutput is the JSON output for the `projects` command.
type ProjectsOutput struct {
	Projects []ProjectOption `json:"projects"`
	Count    int             `json:"count"`
}

// ActivitiesOutput is the JSON output for the `activities` command.
type ActivitiesOutput struct {
	ProjectID  string     `json:"project_id"`
	Activities []Activity `json:"activities"`
	Count      int        `json:"count"`
}

// TimesheetOutput is the JSON output for the `timesheet` command.
type TimesheetOutput struct {
	Year     int              `json:"year"`
	Month    int              `json:"month"`
	Entries  []TimesheetEntry `json:"entries"`
	Overtime []TimesheetEntry `json:"overtime"`
}

// TimesheetEntry represents one project/activity row with daily values.
// Days maps "YYYY-MM-DD" → DayValue for days that have hours, a note, or a record.
type TimesheetEntry struct {
	ProjectID    int                 `json:"project_id"`
	ProjectName  string              `json:"project_name"`
	ActivityID   int                 `json:"activity_id"`
	ActivityName string              `json:"activity_name"`
	TotalHours   float64             `json:"total_hours"`
	Days         map[string]DayValue `json:"days"`
}

// DayValue holds one day's filled values for an entry.
type DayValue struct {
	Hours       float64 `json:"hours"`
	Description string  `json:"description"`
	Status      string  `json:"status"`
	RecordID    int     `json:"record_id"`
}

// SaveOutput is the JSON output for the `save` command.
type SaveOutput struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	AppliedCount int    `json:"applied_count"`
}

// SyncLeaveOutput is the JSON output for the `sync-leave` command.
type SyncLeaveOutput struct {
	Success     bool             `json:"success"`
	Message     string           `json:"message"`
	ProjectID   int              `json:"project_id"`
	ActivityID  int              `json:"activity_id"`
	ProjectName string           `json:"project_name"`
	LeaveDays   []LeaveDayOutput `json:"leave_days"`
}

// LeaveDayOutput is a single applied leave day.
type LeaveDayOutput struct {
	Date      string  `json:"date"`
	Day       int     `json:"day"`
	LeaveType string  `json:"leave_type"`
	Hours     float64 `json:"hours"`
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

// SaveRecordInput is a single record in the save --records JSON.
// Date is YYYY-MM-DD; Hours of 0 clears the day; Overtime targets the
// overtime table instead of the regular table.
type SaveRecordInput struct {
	ProjectID   int     `json:"project_id"`
	ActivityID  int     `json:"activity_id"`
	Date        string  `json:"date"`
	Hours       float64 `json:"hours"`
	Description string  `json:"description"`
	Overtime    bool    `json:"overtime"`
}

// --- Internal leave-days response (from LeaveDays handler) ---

// LeaveDaysResponse mirrors the JSON returned by the LeaveDays handler.
type LeaveDaysResponse struct {
	LeaveDays []struct {
		Day       int     `json:"day"`
		LeaveType string  `json:"leaveType"`
		Hours     float64 `json:"hours"`
	} `json:"leaveDays"`
	LeaveActivity *struct {
		ProjectID   int    `json:"projectId"`
		ActivityID  int    `json:"activityId"`
		ProjectName string `json:"projectName"`
	} `json:"leaveActivity"`
}
