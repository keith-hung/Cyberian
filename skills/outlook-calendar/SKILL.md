---
name: outlook-calendar
description: "Read Outlook calendar events via ICS subscription. Use when user asks about meetings, schedule, calendar, appointments, or availability."
---

# Outlook Calendar Reader

## Overview

This skill reads calendar events from Outlook via ICS subscription URLs. Supports multiple calendars.

## When to Use

- User asks about today's meetings or schedule
- User asks about upcoming events
- User wants to check availability
- User asks "What's on my calendar?"

## Prerequisites

The environment variable `OUTLOOK_ICS_URLS` must be set to one or more ICS subscription URLs (comma-separated).

**Option A — Centralized config (recommended):**
Copy `.claude/settings.json.example` to `.claude/settings.local.json` and fill in the `OUTLOOK_ICS_URLS` value.

**Option B — Shell profile:**
Export the variable in `~/.zshrc` or `~/.bashrc`:
```bash
# Single calendar
export OUTLOOK_ICS_URLS="https://outlook.office365.com/owa/calendar/.../reachcalendar.ics"

# Multiple calendars (comma-separated)
export OUTLOOK_ICS_URLS="https://outlook.office365.com/.../cal1.ics,https://outlook.office365.com/.../cal2.ics"
```

Calendar names are automatically extracted from each ICS feed's `X-WR-CALNAME` property.

## Execution Steps

### Method 1: Use ics_parser.py (Recommended)

The skill includes `ics_parser.py` that handles all parsing and RRULE expansion.

**Prerequisite**: [uv](https://docs.astral.sh/uv/) must be installed.

```bash
# Get today's date
TODAY=$(date +%Y-%m-%d)

# Parse events for today (all calendars)
uv run ${CLAUDE_PLUGIN_ROOT}/skills/outlook-calendar/ics_parser.py \
  --start "$TODAY" \
  --end "$TODAY"

# Parse events for a date range
uv run ${CLAUDE_PLUGIN_ROOT}/skills/outlook-calendar/ics_parser.py \
  --start 2025-12-01 \
  --end 2025-12-31

# Filter by calendar name (substring match)
uv run ${CLAUDE_PLUGIN_ROOT}/skills/outlook-calendar/ics_parser.py \
  --start "$TODAY" \
  --end "$TODAY" \
  --calendar "Work"

# Specify URLs directly (ignores env var)
uv run ${CLAUDE_PLUGIN_ROOT}/skills/outlook-calendar/ics_parser.py \
  --url "https://...cal1.ics" \
  --url "https://...cal2.ics" \
  --start "$TODAY" \
  --end "$TODAY"

# Output as JSON
uv run ${CLAUDE_PLUGIN_ROOT}/skills/outlook-calendar/ics_parser.py \
  --start 2025-12-01 \
  --end 2025-12-31 \
  --format json

# Debug mode (shows event counts per calendar)
uv run ${CLAUDE_PLUGIN_ROOT}/skills/outlook-calendar/ics_parser.py \
  --start 2025-12-01 \
  --end 2025-12-31 \
  --debug
```

### Method 2: Manual Parsing with WebFetch

If the script is not available, use WebFetch and parse manually:

1. **Fetch ICS**: Use WebFetch to retrieve each ICS URL
2. **Parse Events**: Extract VEVENT blocks
3. **Handle RRULE**: Expand recurring events (see RRULE Reference below)
4. **Filter & Format**: Filter by date range and output as table

## Script Features

The `ics_parser.py` script handles:

- Multiple calendars with auto-detected names (X-WR-CALNAME)
- `--calendar` filter for viewing specific calendars
- All 3 DTSTART formats (UTC, TZID, all-day)
- RRULE expansion (WEEKLY, DAILY with BYDAY, INTERVAL, UNTIL, COUNT)
- Automatic timezone conversion (UTC → Taipei UTC+8)
- Dependencies auto-installed by uv
- Table or JSON output format
- Calendar column shown automatically when multiple calendars are present
- Debug mode for troubleshooting

## Output Format

The script outputs a markdown table. When multiple calendars are present, a Calendar column is included:

| Date | Time | Event | Location | Calendar |
|------|------|-------|----------|----------|
| 2025-12-02 (Tue) | 10:30-11:00 | 🔄 Standup Meeting | Microsoft Teams | Work |
| 2025-12-02 (Tue) | 14:00-15:00 | 🔄 Weekly Meeting | Conference Room | Work |
| 2025-12-03 (Wed) | 12:00-13:00 | Lunch | — | Personal |

- 🔄 indicates a recurring event

## RRULE Reference

### Identify Recurring Events

A VEVENT with `RRULE` is a recurring event:

```
BEGIN:VEVENT
DTSTART;TZID=Taipei Standard Time:20250424T103000
RRULE:FREQ=WEEKLY;INTERVAL=1;BYDAY=MO,TU,WE,TH,FR
SUMMARY:Standup Meeting
END:VEVENT
```

### Common RRULE Parameters

| Parameter | Meaning | Example |
|-----------|---------|---------|
| `FREQ` | Frequency | DAILY, WEEKLY, MONTHLY, YEARLY |
| `INTERVAL` | Every N periods | 2 = every 2 weeks |
| `BYDAY` | Day of week | MO, TU, WE, TH, FR, SA, SU |
| `UNTIL` | End date | 20251231T235959Z |
| `COUNT` | Total occurrences | 10 |

### DTSTART Formats

Outlook ICS uses 3 different formats:

| Format | Example | Timezone |
|--------|---------|----------|
| UTC | `DTSTART:20251211T060000Z` | UTC (convert +8) |
| With TZID | `DTSTART;TZID=Taipei Standard Time:20250424T103000` | Local time |
| All-day | `DTSTART;VALUE=DATE:20251225` | Date only |

## Example Queries

- "What meetings do I have today?"
- "Show my calendar for this week"
- "Am I free tomorrow afternoon?"
- "What's on my schedule for 12/5?"
- "Show my Work calendar for this week"
- "What meetings are on my Personal calendar today?"

## Troubleshooting

### Quick Debug

```bash
# Run with debug flag
uv run ${CLAUDE_PLUGIN_ROOT}/skills/outlook-calendar/ics_parser.py \
  --start 2025-12-01 --end 2025-12-31 --debug
```

Output shows (per calendar):
- Calendar name (from X-WR-CALNAME)
- Total VEVENTs in ICS
- Number of recurring events
- Whether dateutil is available
- Events found in range

### Common Issues

| Issue | Solution |
|-------|----------|
| `uv: command not found` | Install uv: `curl -LsSf https://astral.sh/uv/install.sh \| sh` |
| No events found | Check date range matches your query |
| Script not found | Verify `${CLAUDE_PLUGIN_ROOT}` resolves correctly |
| `OUTLOOK_ICS_URLS` not set | Set the env var in shell profile: `export OUTLOOK_ICS_URLS="..."` |
| Was using `OUTLOOK_ICS_URL` | Rename to `OUTLOOK_ICS_URLS` (plural). The singular form is no longer supported |
| Calendar name shows "Calendar 1" | The ICS feed lacks `X-WR-CALNAME`; this is a cosmetic fallback |

## Error Handling

- If `OUTLOOK_ICS_URLS` is not set: Guide user to set the env var in their shell profile
- If uv not found: Install uv first
- If a URL fails to fetch: Skip it with error message, continue with remaining URLs
- If no events found: Confirm the date range and calendar permissions
- If parsing fails: Use `--debug` flag to see event counts per calendar
