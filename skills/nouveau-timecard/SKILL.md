---
name: nouveau-timecard
description: "Fill and manage timesheets in the new Smart Timecard (智慧工時系統) — the rebuilt Razor Pages system. Trigger when user asks about: 工時填報, new timecard/timesheet, filling work hours, sync leave (同步休假), checking timecard status — for the NEW system (NOT the legacy TimeCard)."
user-invokable: true
argument-hint: "[action, e.g. 'timesheet', 'projects', 'save', 'sync-leave']"
---

# Nouveau Timecard Skill

Manage timesheets in the **new** Smart Timecard system (智慧工時系統, a Razor Pages
rebuild) via the `nouveau-timecard` CLI. This is the rebuilt system — distinct
from the legacy `timecard` skill, which targets the old TimeCard.

## CRITICAL RULE

- **NEVER submit for approval.** This CLI intentionally has NO submit command —
  only draft save. Do not attempt to submit, approve, or promote records by any
  means. Draft save only.

## Trigger

Activate when the user asks to:
- View or fill their new 工時填報 / timesheet / work hours
- List projects or activities
- Sync leave (同步休假) into the timesheet
- Save time entries as a draft for the new system

## Prerequisites

Reuses the existing TimeCard environment variables:
- `TIMECARD_BASE_URL` — server URL for the new system
- `TIMECARD_USERNAME` — LDAP login account
- `TIMECARD_PASSWORD` — LDAP password

**Option A — Centralized config (recommended):**
Copy `.claude/settings.json.example` to `.claude/settings.local.json` and fill in the `TIMECARD_*` values.

**Option B — Shell profile:**
Export the variables in `~/.zshrc` or `~/.bashrc`.

Invoke via the platform-appropriate launcher:

| Platform | Command |
|----------|---------|
| Linux / macOS | `${CLAUDE_PLUGIN_ROOT}/scripts/nouveau-timecard-launcher.sh <command> [flags]` |
| Windows (PowerShell) | `& "${env:CLAUDE_PLUGIN_ROOT}/scripts/nouveau-timecard-launcher.ps1" <command> [flags]` |

## Workflow

Always follow this sequence when filling timesheets:

1. **View** the current month → understand what's already filled (`timesheet`)
2. **Find** the project and activity IDs (`projects`, then `activities --project <id>`)
3. **Save** records as a draft in one atomic call (`save`)
4. **Verify** by viewing the timesheet again

## Month selection

All read commands accept either `--date YYYY-MM-DD` or `--year`/`--month`.
If none are given, the current month is used. The `save` command derives the
month from the records' dates (all records must be in the same month).

## Commands

### View timecard status
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/nouveau-timecard-launcher.sh timesheet --year 2026 --month 5
```
Returns `entries[]` (regular) and `overtime[]`, each with `project_id`,
`activity_id`, `total_hours`, and a `days` map keyed by date.

### List projects
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/nouveau-timecard-launcher.sh projects --year 2026 --month 5
```
Returns `projects[]` with `id`, `code`, and `name`.

### List activities for a project
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/nouveau-timecard-launcher.sh activities --project 90001 --year 2026 --month 5
```
Returns `activities[]` with `id` (the `activity_id` needed for save) and `name`.

### Save records (DRAFT ONLY, atomic)
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/nouveau-timecard-launcher.sh save --records '[
  {"project_id":90001,"activity_id":80002,"date":"2026-05-04","hours":8,"description":"work"}
]'
```
Each record: `project_id` (int), `activity_id` (int), `date` (YYYY-MM-DD),
`hours` (float), `description` (optional), `overtime` (optional bool, default false).
Existing month records are preserved; your records are overlaid and the whole
month is posted as a draft.

### Sync leave (同步休假, draft only)
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/nouveau-timecard-launcher.sh sync-leave --year 2026 --month 5
```
Fetches the month's approved leave from BPM and fills the project's "休假"
activity as a draft. Days already submitted/approved are skipped.

### Version
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/nouveau-timecard-launcher.sh version
```

## Important Rules

- **Always use the launcher**: NEVER run the CLI binary directly.
- **NEVER submit** — draft save only (the CLI has no submit command by design).
- Hours must be a multiple of 0.5; 0 clears that day.
- Work description must not exceed 100 characters.
- New records on closed projects or inactive activities are rejected by the server.
- Always show the user their current timesheet BEFORE making changes, and verify AFTER saving.

## Clearing Existing Data

To clear hours for a day, set `hours` to `0` (the existing record is deleted):
```json
{"project_id":90001,"activity_id":80002,"date":"2026-05-04","hours":0}
```

## Error Handling

| Exit Code | Meaning | Action |
|-----------|---------|--------|
| 0 | Success | — |
| 1 | General error | Check error message |
| 2 | Auth error | Verify credentials in env vars |
| 3 | Validation error | Check input format |
| 4 | Network error | Verify server URL is reachable |

Errors are JSON on stderr: `{"success":false,"error":"message"}`.
The cookie session expires after ~25 min idle; the CLI re-logs in automatically.

## Example: Fill a Work Week as Draft

```bash
# 1. Check current state
${CLAUDE_PLUGIN_ROOT}/scripts/nouveau-timecard-launcher.sh timesheet --year 2026 --month 5

# 2. Find the right project and activity
${CLAUDE_PLUGIN_ROOT}/scripts/nouveau-timecard-launcher.sh projects --year 2026 --month 5
${CLAUDE_PLUGIN_ROOT}/scripts/nouveau-timecard-launcher.sh activities --project 90001 --year 2026 --month 5

# 3. Save Mon-Fri as a draft
${CLAUDE_PLUGIN_ROOT}/scripts/nouveau-timecard-launcher.sh save --records '[
  {"project_id":90001,"activity_id":80002,"date":"2026-05-04","hours":8,"description":"work"},
  {"project_id":90001,"activity_id":80002,"date":"2026-05-05","hours":8,"description":"work"},
  {"project_id":90001,"activity_id":80002,"date":"2026-05-06","hours":8,"description":"work"},
  {"project_id":90001,"activity_id":80002,"date":"2026-05-07","hours":8,"description":"work"},
  {"project_id":90001,"activity_id":80002,"date":"2026-05-08","hours":8,"description":"work"}
]'

# 4. Verify
${CLAUDE_PLUGIN_ROOT}/scripts/nouveau-timecard-launcher.sh timesheet --year 2026 --month 5
```
