---
name: timecard
description: "Fill and manage timesheets in TimeCard. Trigger when user asks about: timesheet, timecard, work hours, filling hours, checking attendance."
user-invokable: true
argument-hint: "[action, e.g. 'view', 'fill', 'summary']"
---

# TimeCard Skill

Manage timesheets via the `timecard` CLI.

## Trigger

Activate when user asks to:
- View, fill, update, or check their timesheet / timecard / work hours
- Query projects or activities
- Save time entries for a specific week

## Prerequisites

Environment variables must be set before use:
- `TIMECARD_BASE_URL` — TimeCard server URL (e.g., `http://timecard.example.com/TimeCard/`)
- `TIMECARD_USERNAME` — Login username
- `TIMECARD_PASSWORD` — Login password

The CLI binary is in `${CLAUDE_PLUGIN_ROOT}/scripts/`. Invoke via the platform-appropriate launcher:

| Platform | Command |
|----------|---------|
| Linux / macOS | `${CLAUDE_PLUGIN_ROOT}/scripts/timecard-launcher.sh <command> [flags]` |
| Windows (PowerShell) | `& "${env:CLAUDE_PLUGIN_ROOT}/scripts/timecard-launcher.ps1" <command> [flags]` |

## Workflow

Always follow this sequence when filling timesheets:

1. **View** current timesheet → understand what's already filled
2. **Find** project and activity → get the `activity_value` needed for save
3. **Save** entries + hours + notes in one atomic call
4. **Verify** by viewing the timesheet again

## Commands

### View current timesheet
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/timecard-launcher.sh timesheet --date YYYY-MM-DD
```
Returns JSON with `entries[]`, each containing `daily_hours`, `daily_status`, `daily_notes`.

### List projects
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/timecard-launcher.sh projects
```
Returns `projects[]` with `id` and `name`.

### List activities for a project
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/timecard-launcher.sh activities --project <project_id>
```
The `value` field in each activity is the `activity_value` needed for the save command.

### Save timesheet (atomic)
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/timecard-launcher.sh save --date YYYY-MM-DD \
  --entries '[{"entry_index":0,"project_id":"PID","activity_value":"VAL"}]' \
  --hours '[{"entry_index":0,"date":"YYYY-MM-DD","hours":8}]' \
  --notes '[{"entry_index":0,"date":"YYYY-MM-DD","note":"Description"}]'
```
At least one of `--entries`, `--hours`, `--notes` is required.

### Weekly summary
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/timecard-launcher.sh summary --date YYYY-MM-DD
```
Returns `total_hours`, `daily_totals`, `project_breakdown`, `statistics`.

### Version
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/timecard-launcher.sh version
```

## Important Rules

- `entry_index` is 0-9 (max 10 entries per week)
- All dates in `--hours`/`--notes` must be in the same week as `--date`
- Daily hours per day MUST NOT exceed 8
- Notes CANNOT contain: `#$%^&*=+{}[]|?'"`
- Only draft save is supported — NEVER submit for approval
- Always show the user their current timesheet BEFORE making changes
- Always verify changes AFTER saving

## Clearing Existing Data

To clear hours for a specific day, set `hours` to `0`:
```json
{"entry_index": 0, "date": "2026-03-02", "hours": 0}
```

To clear a note, set `note` to an empty string:
```json
{"entry_index": 0, "date": "2026-03-02", "note": ""}
```

## Cross-Month Weeks

TimeCard weeks run Monday to Saturday. When a week spans two months (e.g., Mon Mar 30 – Sat Apr 4), just use the exact `YYYY-MM-DD` dates as usual. The CLI converts each date to a day index (0=Mon, 5=Sat) internally — month boundaries are irrelevant.

If some days in the week were already submitted (e.g., old month submitted, new month still draft), just fill the new days normally. The server ignores already-submitted records — no risk of overwriting or downgrading them.

Example: filling hours for a week that crosses March into April:
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/timecard-launcher.sh save --date 2026-03-31 \
  --hours '[
    {"entry_index":0,"date":"2026-04-01","hours":8},
    {"entry_index":0,"date":"2026-04-02","hours":8},
    {"entry_index":0,"date":"2026-04-03","hours":8}
  ]'
```
In this example, Mon Mar 30 and Tue Mar 31 were already submitted — we simply omit them and only fill the new month's days.

## Error Handling

| Exit Code | Meaning | Action |
|-----------|---------|--------|
| 0 | Success | — |
| 1 | General error | Check error message |
| 2 | Auth error | Verify credentials in env vars |
| 3 | Validation error | Check input format |
| 4 | Network error | Verify server URL is reachable |

Errors are JSON on stderr: `{"success":false,"error":"message"}`.
Session expires after 25 min idle; the CLI handles re-login automatically.

## Example: Fill a Full Week

```bash
# 1. Check current state
${CLAUDE_PLUGIN_ROOT}/scripts/timecard-launcher.sh timesheet --date 2026-03-02

# 2. Find the right activity
${CLAUDE_PLUGIN_ROOT}/scripts/timecard-launcher.sh projects
${CLAUDE_PLUGIN_ROOT}/scripts/timecard-launcher.sh activities --project 17647

# 3. Save entries + hours + notes
${CLAUDE_PLUGIN_ROOT}/scripts/timecard-launcher.sh save --date 2026-03-02 \
  --entries '[{"entry_index":0,"project_id":"17647","activity_value":"true$9$17647$100"}]' \
  --hours '[
    {"entry_index":0,"date":"2026-03-02","hours":8},
    {"entry_index":0,"date":"2026-03-03","hours":8},
    {"entry_index":0,"date":"2026-03-04","hours":8},
    {"entry_index":0,"date":"2026-03-05","hours":8},
    {"entry_index":0,"date":"2026-03-06","hours":8}
  ]' \
  --notes '[
    {"entry_index":0,"date":"2026-03-02","note":"Development work"}
  ]'

# 4. Verify
${CLAUDE_PLUGIN_ROOT}/scripts/timecard-launcher.sh timesheet --date 2026-03-02
```
