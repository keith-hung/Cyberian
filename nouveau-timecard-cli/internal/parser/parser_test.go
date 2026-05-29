package parser

import (
	"encoding/json"
	"testing"
)

// pageHead is a synthetic fixture using fake placeholder IDs and names (no real
// data). It exercises the real markup shapes: antiforgery token, the Alpine
// x-data reference that must NOT be mistaken for the assignment, project
// <select> options (with an HTML entity to verify decoding), and serverTimeRows.
const pageHead = `
<input name="__RequestVerificationToken" type="hidden" value="TOKEN123abc">
<div x-data="batchTimeEntry(serverTimeRows, serverOvertimeRows, 31, activityHtmlCache)" x-cloak></div>
<select class="form-select" data-record-type="time">
    <option value="">-- 請選擇專案 --</option>
    <option value="90001" data-status="InProgress">
        [SAMPLE] Sample &amp; Co
    </option>
    <option value="90002" data-status="InProgress">
        [SAMPLE2] Sample Project Two
    </option>
</select>
<script>
    const serverTimeRows = [{"activityId":80002,"projectId":90001,"projectName":"Sample Project","projectStatus":"InProgress","activityActive":true,"activityName":"Task Two","dailyHoursArray":[null,null,null,8,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null],"dailyDescriptionsArray":["","","","","","","","","","","","","","","","","","","","","","","","","","","","","","",""],"recordIdsArray":[null,null,null,70001,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null],"recordStatusesArray":["","","","Save","","","","","","","","","","","","","","","","","","","","","","","","","","",""],"recordDatesArray":[null,null,null,"2026-01-05T00:00:00.0000000",null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null,null]}];
    const serverOvertimeRows = [];
`

// livePage assembles the fixture, building activityHtmlCache via json.Marshal so
// the embedded option HTML is escaped exactly as the server emits it.
func livePage(t *testing.T) string {
	t.Helper()
	optHTML := "<option value=\"\">-- 請選擇活動 --</option>\r\n" +
		"<option value=\"80001\">Sample Project→Group→Task One</option>\r\n" +
		"<option value=\"80002\">Sample Project→Group→Task Two</option>\r\n"
	cache, err := json.Marshal(map[string]string{"90001": optHTML})
	if err != nil {
		t.Fatalf("marshal cache: %v", err)
	}
	return pageHead + "    const activityHtmlCache = " + string(cache) + ";\n    const holidays = [];\n</script>\n"
}

func TestParseAntiForgeryToken(t *testing.T) {
	if got := ParseAntiForgeryToken(livePage(t)); got != "TOKEN123abc" {
		t.Fatalf("token = %q, want TOKEN123abc", got)
	}
}

func TestIsLoginPage(t *testing.T) {
	if IsLoginPage(livePage(t)) {
		t.Fatal("timesheet page wrongly detected as login page")
	}
	login := `<form method="post"><input id="username"><input id="password" type="password"></form>`
	if !IsLoginPage(login) {
		t.Fatal("login page not detected")
	}
}

func TestParseProjectOptions(t *testing.T) {
	projects := ParseProjectOptions(livePage(t))
	if len(projects) != 2 {
		t.Fatalf("got %d projects, want 2", len(projects))
	}
	// project[0] name carries an HTML entity to verify decoding.
	if p := projects[0]; p.ID != "90001" || p.Code != "SAMPLE" || p.Name != "Sample & Co" {
		t.Fatalf("project[0] = %+v, want id=90001 code=SAMPLE name=\"Sample & Co\"", p)
	}
	if p := projects[1]; p.Code != "SAMPLE2" || p.Name != "Sample Project Two" {
		t.Fatalf("project[1] = %+v", p)
	}
}

func TestParseActivityCache(t *testing.T) {
	cache, err := ParseActivityCache(livePage(t))
	if err != nil {
		t.Fatalf("ParseActivityCache error: %v", err)
	}
	acts := cache["90001"]
	if len(acts) != 2 {
		t.Fatalf("got %d activities, want 2 (placeholder must be skipped)", len(acts))
	}
	if acts[1].ID != "80002" || acts[1].Name != "Sample Project→Group→Task Two" {
		t.Fatalf("activity[1] = %+v", acts[1])
	}
}

func TestParseServerRows(t *testing.T) {
	page := livePage(t)
	rows, err := ParseServerRows(page, "serverTimeRows")
	if err != nil {
		t.Fatalf("ParseServerRows error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	r := rows[0]
	if r.ActivityID != 80002 || r.ProjectID != 90001 {
		t.Fatalf("row ids = %d/%d", r.ProjectID, r.ActivityID)
	}
	if r.DailyHoursArray[3] == nil || *r.DailyHoursArray[3] != 8 {
		t.Fatalf("day-4 hours not 8: %+v", r.DailyHoursArray[3])
	}
	if r.RecordIdsArray[3] == nil || *r.RecordIdsArray[3] != 70001 {
		t.Fatalf("day-4 recordId mismatch")
	}
	if r.RecordStatusesArray[3] != "Save" {
		t.Fatalf("day-4 status = %q, want Save", r.RecordStatusesArray[3])
	}
	overtime, err := ParseServerRows(page, "serverOvertimeRows")
	if err != nil || len(overtime) != 0 {
		t.Fatalf("overtime rows = %d (err %v), want 0", len(overtime), err)
	}
}
