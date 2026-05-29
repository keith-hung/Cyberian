package session

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"time"

	"github.com/keith-hung/nouveau-timecard-cli/internal/types"
)

// HourIncrement is the minimum granularity enforced by the server.
const HourIncrement = 0.5

// MaxDescriptionLength mirrors the server-side limit.
const MaxDescriptionLength = 100

// dayCell is one (project, activity, day) record in the working model.
type dayCell struct {
	recordID   int
	hours      float64
	desc       string
	recordDate string
	status     string
}

// entryModel groups day cells for one (project, activity) row.
type entryModel struct {
	projectID  int
	activityID int
	days       map[int]*dayCell // keyed by day-of-month (1-based)
}

// entryKey uniquely identifies a row.
type entryKey struct {
	projectID  int
	activityID int
}

// SaveDraft reconstructs the full month form state from existing records,
// overlays the supplied changes, and POSTs a draft save (handler=SaveBoth).
// It NEVER submits for approval.
func (s *Session) SaveDraft(year, month int, records []types.SaveRecordInput) (int, error) {
	if err := validateRecords(year, month, records); err != nil {
		return 0, err
	}

	page, err := s.FetchPageData(year, month)
	if err != nil {
		return 0, err
	}

	regular := seedModel(page.TimeRows)
	overtime := seedModel(page.OvertimeRows)

	for _, rec := range records {
		day := dayOfMonth(rec.Date)
		target := regular
		if rec.Overtime {
			target = overtime
		}
		key := entryKey{rec.ProjectID, rec.ActivityID}
		em, ok := target[key]
		if !ok {
			em = &entryModel{projectID: rec.ProjectID, activityID: rec.ActivityID, days: map[int]*dayCell{}}
			target[key] = em
		}
		cell := em.days[day]
		if cell == nil {
			cell = &dayCell{}
			em.days[day] = cell
		}
		cell.hours = rec.Hours
		cell.desc = rec.Description
	}

	form := url.Values{}
	form.Set("Year", strconv.Itoa(year))
	form.Set("Month", strconv.Itoa(month))
	form.Set("__RequestVerificationToken", page.Token)
	appendEntries(form, "Activities", regular)
	appendEntries(form, "OvertimeActivities", overtime)

	resp, err := s.client.PostForm(saveBothPath(year, month), form)
	if err != nil {
		return 0, fmt.Errorf("save request: %w", err)
	}

	msg := parseMessage(resp.Body)
	if resp.Status != 200 {
		if msg != "" {
			return 0, fmt.Errorf("%s", msg)
		}
		return 0, fmt.Errorf("save failed (status %d)", resp.Status)
	}
	return len(records), nil
}

func saveBothPath(year, month int) string {
	return fmt.Sprintf("Timesheet/Index?handler=SaveBoth&year=%d&month=%d", year, month)
}

// seedModel builds the working model from existing server rows.
func seedModel(rows []types.ServerRow) map[entryKey]*entryModel {
	model := make(map[entryKey]*entryModel)
	for _, row := range rows {
		em := &entryModel{projectID: row.ProjectID, activityID: row.ActivityID, days: map[int]*dayCell{}}
		for i := range row.DailyHoursArray {
			day := i + 1
			var hours float64
			if row.DailyHoursArray[i] != nil {
				hours = *row.DailyHoursArray[i]
			}
			desc := ""
			if i < len(row.DailyDescriptionsArray) {
				desc = row.DailyDescriptionsArray[i]
			}
			recordID := 0
			if i < len(row.RecordIdsArray) && row.RecordIdsArray[i] != nil {
				recordID = *row.RecordIdsArray[i]
			}
			recordDate := ""
			if i < len(row.RecordDatesArray) && row.RecordDatesArray[i] != nil {
				recordDate = *row.RecordDatesArray[i]
			}
			status := ""
			if i < len(row.RecordStatusesArray) {
				status = row.RecordStatusesArray[i]
			}
			if hours > 0 || desc != "" || recordID > 0 {
				em.days[day] = &dayCell{recordID: recordID, hours: hours, desc: desc, recordDate: recordDate, status: status}
			}
		}
		model[entryKey{row.ProjectID, row.ActivityID}] = em
	}
	return model
}

// appendEntries writes the model into form fields under the given prefix
// (Activities or OvertimeActivities), mirroring the page's field layout.
func appendEntries(form url.Values, prefix string, model map[entryKey]*entryModel) {
	keys := make([]entryKey, 0, len(model))
	for k := range model {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(a, b int) bool {
		if keys[a].projectID != keys[b].projectID {
			return keys[a].projectID < keys[b].projectID
		}
		return keys[a].activityID < keys[b].activityID
	})

	for i, k := range keys {
		em := model[k]
		base := fmt.Sprintf("%s[%d]", prefix, i)
		form.Set(base+".ProjectId", strconv.Itoa(em.projectID))
		form.Set(base+".Id", strconv.Itoa(em.activityID))

		days := make([]int, 0, len(em.days))
		for d := range em.days {
			days = append(days, d)
		}
		sort.Ints(days)

		j := 0
		for _, d := range days {
			cell := em.days[d]
			// Include a record if it has hours, a description, or is an existing
			// record (so hours=0 deletes it server-side) — matching the frontend.
			if cell.hours <= 0 && cell.desc == "" && cell.recordID <= 0 {
				continue
			}
			rb := fmt.Sprintf("%s.Records[%d]", base, j)
			form.Set(rb+".RecordId", strconv.Itoa(cell.recordID))
			form.Set(rb+".Day", strconv.Itoa(d))
			form.Set(rb+".Hours", formatHours(cell.hours))
			form.Set(rb+".Description", cell.desc)
			form.Set(rb+".RecordDate", cell.recordDate)
			j++
		}
	}
}

// --- Leave sync ---

// GetLeaveDays calls the LeaveDays handler and returns its parsed response.
func (s *Session) GetLeaveDays(year, month int) (*types.LeaveDaysResponse, error) {
	if err := s.EnsureAuth(); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("Timesheet/Index?handler=LeaveDays&year=%d&month=%d", year, month)
	resp, err := s.client.Get(path)
	if err != nil {
		return nil, err
	}
	var out types.LeaveDaysResponse
	if err := json.Unmarshal([]byte(resp.Body), &out); err != nil {
		return nil, fmt.Errorf("parsing leave days response: %w", err)
	}
	return &out, nil
}

// SyncLeave fetches leave days from BPM and fills the "休假" activity rows as a
// draft save. Approved/submitted days for that activity are skipped.
func (s *Session) SyncLeave(year, month int) (*types.SyncLeaveOutput, error) {
	leave, err := s.GetLeaveDays(year, month)
	if err != nil {
		return nil, err
	}
	if leave.LeaveActivity == nil {
		return nil, fmt.Errorf("no 休假 activity found for this user/month")
	}

	// Determine which days already have a locked (Submit/Approve) leave record.
	page, err := s.FetchPageData(year, month)
	if err != nil {
		return nil, err
	}
	locked := lockedDaysForActivity(page.TimeRows, leave.LeaveActivity.ActivityID)

	var records []types.SaveRecordInput
	var applied []types.LeaveDayOutput
	for _, ld := range leave.LeaveDays {
		if ld.Hours <= 0 || locked[ld.Day] {
			continue
		}
		date := fmt.Sprintf("%04d-%02d-%02d", year, month, ld.Day)
		records = append(records, types.SaveRecordInput{
			ProjectID:   leave.LeaveActivity.ProjectID,
			ActivityID:  leave.LeaveActivity.ActivityID,
			Date:        date,
			Hours:       ld.Hours,
			Description: ld.LeaveType,
		})
		applied = append(applied, types.LeaveDayOutput{Date: date, Day: ld.Day, LeaveType: ld.LeaveType, Hours: ld.Hours})
	}

	out := &types.SyncLeaveOutput{
		ProjectID:   leave.LeaveActivity.ProjectID,
		ActivityID:  leave.LeaveActivity.ActivityID,
		ProjectName: leave.LeaveActivity.ProjectName,
		LeaveDays:   applied,
	}

	if len(records) == 0 {
		out.Success = true
		out.Message = "No leave days to apply for this month"
		return out, nil
	}
	if _, err := s.SaveDraft(year, month, records); err != nil {
		return nil, err
	}
	out.Success = true
	out.Message = fmt.Sprintf("Synced %d leave day(s) as draft", len(records))
	return out, nil
}

func lockedDaysForActivity(rows []types.ServerRow, activityID int) map[int]bool {
	locked := map[int]bool{}
	for _, row := range rows {
		if row.ActivityID != activityID {
			continue
		}
		for i, st := range row.RecordStatusesArray {
			if st == "Submit" || st == "Approve" {
				locked[i+1] = true
			}
		}
	}
	return locked
}

// --- validation & helpers ---

func validateRecords(year, month int, records []types.SaveRecordInput) error {
	if len(records) == 0 {
		return fmt.Errorf("no records provided")
	}
	for i, r := range records {
		if r.ProjectID <= 0 || r.ActivityID <= 0 {
			return fmt.Errorf("record %d: project_id and activity_id must be positive", i)
		}
		t, err := time.Parse("2006-01-02", r.Date)
		if err != nil {
			return fmt.Errorf("record %d: invalid date %q (want YYYY-MM-DD)", i, r.Date)
		}
		if t.Year() != year || int(t.Month()) != month {
			return fmt.Errorf("record %d: date %s is not in %04d-%02d", i, r.Date, year, month)
		}
		if r.Hours < 0 {
			return fmt.Errorf("record %d: hours cannot be negative", i)
		}
		if r.Hours > 0 && mod(r.Hours, HourIncrement) != 0 {
			return fmt.Errorf("record %d: hours must be a multiple of %.1f", i, HourIncrement)
		}
		if len([]rune(r.Description)) > MaxDescriptionLength {
			return fmt.Errorf("record %d: description exceeds %d characters", i, MaxDescriptionLength)
		}
	}
	return nil
}

func dayOfMonth(date string) int {
	t, _ := time.Parse("2006-01-02", date)
	return t.Day()
}

func mod(a, b float64) float64 {
	n := a / b
	return a - float64(int(n+0.0000001))*b
}

func formatHours(h float64) string {
	return strconv.FormatFloat(h, 'f', -1, 64)
}

// parseMessage extracts the "message" field from a {"message": ...} JSON body.
func parseMessage(body string) string {
	var m struct {
		Message *string `json:"message"`
	}
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		return ""
	}
	if m.Message == nil {
		return ""
	}
	return *m.Message
}
