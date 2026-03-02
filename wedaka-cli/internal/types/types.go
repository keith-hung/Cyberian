// Package types defines request/response structures for the WeDaka API.
package types

// DateTypeResponse is the API response for GetDateType.
type DateTypeResponse struct {
	DateType     string `json:"DateType"`
	Status       bool   `json:"Status"`
	ErrorMessage string `json:"ErrorMessage,omitempty"`
}

// TimeLogResponse is the API response for InsertTimeLog.
type TimeLogResponse struct {
	Status       bool   `json:"Status"`
	ErrorMessage string `json:"ErrorMessage,omitempty"`
	LogId        string `json:"LogId,omitempty"`
	LogTime      string `json:"LogTime,omitempty"`
}

// TimeLogRecord is a single time log entry.
type TimeLogRecord struct {
	DateType   *string  `json:"DateType"`
	LeaveHours *float64 `json:"LeaveHours"`
	Memo       *string  `json:"Memo"`
	WorkItem   *string  `json:"WorkItem"`
	WorkTime   *string  `json:"WorkTime"`
	WorkType   *string  `json:"WorkType"`
	WorkDate   *string  `json:"WorkDate"`
}

// SearchTimelogResponse is the API response for SearchTimelog.
type SearchTimelogResponse struct {
	Status       bool            `json:"Status"`
	ErrorMessage string          `json:"ErrorMessage,omitempty"`
	TimeLog      []TimeLogRecord `json:"TimeLog,omitempty"`
}

// InsertTimeLogPayload is the request body for InsertTimeLog.
type InsertTimeLogPayload struct {
	UserName        string              `json:"UserName"`
	WorkTimeLogData []WorkTimeLogEntry  `json:"WorkTimeLogData"`
}

// WorkTimeLogEntry is a single entry in the InsertTimeLog payload.
type WorkTimeLogEntry struct {
	DateType   string `json:"DateType"`
	LeaveHours int    `json:"LeaveHours"`
	Memo       string `json:"Memo"`
	WorkItem   string `json:"WorkItem"`
	WorkTime   string `json:"WorkTime"`
	WorkType   string `json:"WorkType"`
}

// ErrorOutput is the standard error JSON written to stderr.
type ErrorOutput struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

// ClockResult is the output for clock-in / clock-out commands.
type ClockResult struct {
	Success bool   `json:"success"`
	LogId   string `json:"log_id,omitempty"`
	LogTime string `json:"log_time,omitempty"`
	Message string `json:"message,omitempty"`
}

// TimelogOutput is the output for the timelog command.
type TimelogOutput struct {
	Success      bool            `json:"success"`
	TotalRecords int             `json:"total_records"`
	Summary      TimelogSummary  `json:"summary"`
	Records      []TimelogRecord `json:"records"`
	Message      string          `json:"message,omitempty"`
}

// TimelogSummary counts record types.
type TimelogSummary struct {
	ClockIns  int `json:"clock_ins"`
	ClockOuts int `json:"clock_outs"`
	Leaves    int `json:"leaves"`
}

// TimelogRecord is an enhanced time log record for output.
type TimelogRecord struct {
	DateType    *string  `json:"date_type"`
	LeaveHours  *float64 `json:"leave_hours"`
	Memo        *string  `json:"memo"`
	WorkItem    *string  `json:"work_item"`
	WorkTime    *string  `json:"work_time"`
	WorkType    *string  `json:"work_type"`
	WorkDate    *string  `json:"work_date"`
	Description string   `json:"description"`
	IsLeave     bool     `json:"is_leave"`
}

// CheckWorkdayOutput is the output for the check-workday command.
type CheckWorkdayOutput struct {
	Success     bool   `json:"success"`
	Date        string `json:"date"`
	DateType    string `json:"date_type"`
	IsWorkDay   bool   `json:"is_work_day"`
	Description string `json:"description"`
	Message     string `json:"message,omitempty"`
}
