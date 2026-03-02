package session

import (
	"fmt"
	"log"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/keith-hung/timecard-cli/internal/parser"
	"github.com/keith-hung/timecard-cli/internal/types"
)

var forbiddenNoteChars = regexp.MustCompile(`[#$%^&*=+{}\[\]|?'"]`)
var recordFieldRegex = regexp.MustCompile(`^record(\d+)_(\d+)$`)

// SaveTimesheet performs an atomic save: fetch → reconstruct → apply → POST.
func (s *Session) SaveTimesheet(
	date string,
	entries []types.SaveEntryInput,
	hours []types.SaveHoursInput,
	notes []types.SaveNotesInput,
) error {
	if err := s.EnsureAuth(); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	totalChanges := len(entries) + len(hours) + len(notes)
	if totalChanges == 0 {
		return nil
	}

	log.Printf("[Save] Starting save for %s with %d changes", date, totalChanges)

	// 1. Fetch page (sets server session attributes for this week)
	html, err := s.FetchTimesheetPage(date)
	if err != nil {
		return fmt.Errorf("fetching timesheet: %w", err)
	}

	// 2. Reconstruct full form state from HTML
	activities := parser.ParseActivityList(html)
	timeEntries := parser.ParseTimearray(html)
	overtimeEntries := parser.ParseOvertimeArray(html)
	formData := ReconstructFormState(html, activities, timeEntries, overtimeEntries)

	// 3. Compute Monday of the target week (for date → dayIndex conversion)
	targetDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return fmt.Errorf("parsing date %q: %w", date, err)
	}
	dow := int(targetDate.Weekday())
	daysToMonday := dow - 1
	if dow == 0 {
		daysToMonday = 6 // Sunday
	}
	monday := targetDate.AddDate(0, 0, -daysToMonday)

	dateToDayIndex := func(d string) (int, error) {
		dt, err := time.Parse("2006-01-02", d)
		if err != nil {
			return 0, fmt.Errorf("parsing date %q: %w", d, err)
		}
		diff := int(math.Round(dt.Sub(monday).Hours() / 24))
		if diff < 0 || diff > 6 {
			return 0, fmt.Errorf("date %s is not in the same week as %s (Mon %s)",
				d, date, monday.Format("2006-01-02"))
		}
		return diff, nil
	}

	// 4. Apply entries
	for _, entry := range entries {
		if entry.EntryIndex < 0 || entry.EntryIndex > 9 {
			return fmt.Errorf("invalid entry_index %d: must be 0-9", entry.EntryIndex)
		}
		if !strings.Contains(entry.ActivityValue, "$") {
			return fmt.Errorf("invalid activity_value for entry %d: must be 'bottom$uid$pid$progress' format", entry.EntryIndex)
		}
		formData[fmt.Sprintf("project%d", entry.EntryIndex)] = entry.ProjectID
		formData[fmt.Sprintf("activity%d", entry.EntryIndex)] = entry.ActivityValue
	}

	// 5. Apply hours
	for _, h := range hours {
		if h.EntryIndex < 0 || h.EntryIndex > 9 {
			return fmt.Errorf("invalid entry_index %d: must be 0-9", h.EntryIndex)
		}
		dayIndex, err := dateToDayIndex(h.Date)
		if err != nil {
			return err
		}
		value := ""
		if h.Hours != 0 {
			value = strconv.FormatFloat(h.Hours, 'f', -1, 64)
		}
		formData[fmt.Sprintf("record%d_%d", h.EntryIndex, dayIndex)] = value
	}

	// 6. Apply notes
	for _, n := range notes {
		if n.EntryIndex < 0 || n.EntryIndex > 9 {
			return fmt.Errorf("invalid entry_index %d: must be 0-9", n.EntryIndex)
		}
		if forbiddenNoteChars.MatchString(n.Note) {
			found := forbiddenNoteChars.FindAllString(n.Note, -1)
			return fmt.Errorf("note for entry %d contains forbidden characters: %s",
				n.EntryIndex, strings.Join(found, ", "))
		}
		dayIndex, err := dateToDayIndex(n.Date)
		if err != nil {
			return err
		}
		formData[fmt.Sprintf("note%d_%d", n.EntryIndex, dayIndex)] = n.Note
	}

	// 7. Recalculate norTotal after applying changes
	dailyHours := [7]float64{}
	for fieldName, value := range formData {
		m := recordFieldRegex.FindStringSubmatch(fieldName)
		if m == nil || strings.TrimSpace(value) == "" {
			continue
		}
		dayIndex, _ := strconv.Atoi(m[2])
		if dayIndex >= 0 && dayIndex <= 6 {
			if h, err := strconv.ParseFloat(value, 64); err == nil {
				dailyHours[dayIndex] += h
			}
		}
	}
	for k := 0; k < 7; k++ {
		formData[fmt.Sprintf("norTotal%d", k)] = strconv.FormatFloat(dailyHours[k], 'f', -1, 64)
	}

	// 8. Validate daily hours ≤ 8
	dayNames := [7]string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
	var violations []string
	for day := 0; day <= 6; day++ {
		if dailyHours[day] > 8 {
			violations = append(violations, fmt.Sprintf("%s: %.1fhr (max 8hr)", dayNames[day], dailyHours[day]))
		}
	}
	if len(violations) > 0 {
		return fmt.Errorf("daily hours exceeded: %s", strings.Join(violations, ", "))
	}

	// 9. Remove submit buttons (submit is STRICTLY PROHIBITED)
	delete(formData, "submit")
	delete(formData, "submit2")

	// 10. Add save action
	formData["save"] = " save "

	// 11. POST to weekinfo_deal.jsp
	log.Println("[Save] POSTing to weekinfo_deal.jsp")
	resp, err := s.client.Post("Timecard/timecard_week/weekinfo_deal.jsp", formData)
	if err != nil {
		return fmt.Errorf("POST save: %w", err)
	}

	// 12. Check response
	log.Printf("[Save] Response status: %d", resp.Status)
	if resp.Status == 302 || resp.Status == 301 {
		location := resp.Headers.Get("Location")
		if location != "" {
			s.client.Get(location) //nolint:errcheck
		}
		return nil
	}
	if resp.Status == 200 {
		if strings.Contains(resp.Body, "Error Page") {
			return fmt.Errorf("server returned error page")
		}
		return nil
	}

	return fmt.Errorf("unexpected response status: %d", resp.Status)
}
