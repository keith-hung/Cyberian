---
name: outlook-calendar
description: "Read Outlook (Microsoft 365) calendar events from the OneDrive Calendar-Export files written by a Power Automate flow. Use when user asks about meetings, schedule, calendar, appointments, availability, who is attending a meeting, or a meeting's Teams link."
---

# Outlook Calendar Reader

## Overview

Outlook can no longer publish calendars as ICS, so a Power Automate flow (run under the user's own account) exports the calendar to the user's OneDrive every 30 minutes. This skill reads those files locally:

| File | Range | Fields | Updated |
|------|-------|--------|---------|
| `Calendar-Export/calendar.json` | 7 days ago → 30 days ahead | full: subject, time, location, organizer, show-as, body preview, attendees with responses, meeting link | every 30 min, fully rewritten |
| `Calendar-Export/calendar-history.json` | past year | slim: subject, time, location, organizer, show-as, status | about once a day, merged; deleted meetings kept as `deleted` |

## When to Use

- "What meetings do I have today / this week / on 12/5?"
- "Am I free tomorrow afternoon?"
- "Who is attending the review meeting?" / "Send me the Teams link for ..."
- "What meetings did I have last month?" (history)

## Execution Steps

`calendar_reader.py` is stdlib-only; run it with `uv run` or `python3`.

```bash
R=${CLAUDE_PLUGIN_ROOT}/skills/outlook-calendar/calendar_reader.py

# Today (default range is today only)
uv run $R

# Date range (YYYY-MM-DD, inclusive)
uv run $R --start 2026-11-16 --end 2026-11-22

# Attendees, body preview and meeting link (snapshot window only)
uv run $R --start 2026-11-18 --details --format json

# Include cancelled meetings
uv run $R --start 2026-11-18 --end 2026-11-30 --include-cancelled

# Check file location and freshness
uv run $R --status
```

Always run `date` first to resolve "today", "tomorrow" or weekday names into dates.

### Options

| Option | Meaning |
|--------|---------|
| `--start`, `--end` | Date range, inclusive; default today |
| `--format table\|json` | Markdown table (default) or JSON |
| `--details` | Add `preview`, `attendees` (name, email, type, response) and `meetingUrl` — snapshot window only |
| `--include-cancelled` | Also list cancelled meetings (adds a Status column) |
| `--dir PATH` | Calendar-Export folder; overrides auto-detection |
| `--status` | Print folder, `generatedAt`, windows, counts and staleness only |

### How the two files are combined

- Inside the snapshot window, only `calendar.json` is used (it is current; history may lag up to a day).
- Before the snapshot window, `calendar-history.json` fills in.
- Only `active` meetings are listed by default; `deleted` (history only) is never listed.
- Multi-day all-day events are expanded into one row per day.
- All times are Taipei time (`+08:00`).

## Output Format

```
| Date | Time | Event | Location | Show As |
|---|---|---|---|---|
| 2026-11-18 (Wed) | 10:30-11:00 | Team Standup (recurring) | Microsoft Teams Meeting | busy |
| 2026-11-20 (Fri) | All day | Company Holiday | — | free |
```

When answering availability questions, treat `busy`, `oof` and `tentative` as occupied and `free` / `workingElsewhere` as available.

Warnings (stale data, dates outside the exported range) are printed to stderr, appended under the table, and included in JSON `warnings`. Relay them to the user.

## Configuration

| Variable | Required | Meaning |
|----------|----------|---------|
| `OUTLOOK_CALENDAR_DIR` | No | Path to the `Calendar-Export` folder. Without it the script searches `%OneDriveCommercial%`, `%OneDrive%`, `~/Library/CloudStorage/OneDrive-*/` (macOS), `/mnt/c/Users/*/OneDrive - */` (WSL) and `~/OneDrive*/` |

## Setup (once per user)

1. In OneDrive root create folder `Calendar-Export` with two files, `calendar.json` and `calendar-history.json`, each containing `{"events":[]}`.
2. Power Automate → **My flows** → **Import** → **Import Package (Legacy)** → upload `flow/calendar-export-flow.zip` from this skill.
3. Under Related resources pick (or create) the **Office 365 Outlook** and **OneDrive for Business** connections → **Import**.
4. Open the flow → **Save** → **Test** → **Manually**. Both files grow after the run; then it runs every 30 minutes.

`flow/make_package.py` regenerates the package if the flow definition changes.

## Sensitive Data

`calendar.json` contains body previews, attendee emails and meeting join links. Use them only to answer the user locally; do not paste them into chat channels, web pages or other public places, and do not modify either file.

## Troubleshooting

| Issue | Cause / Solution |
|-------|------------------|
| `Calendar-Export/calendar.json not found` (exit 2) | Flow not set up, OneDrive not synced, or non-standard path → set `OUTLOOK_CALENDAR_DIR` |
| `cannot read ...` | OneDrive online-only placeholder → ask the user to open the file once in Finder / File Explorer |
| `snapshot is stale` | The flow stopped (failed runs, expired connection, DLP) → ask the user to check run history in Power Automate |
| `dates ... are not covered by the export` | Range is beyond +30 days or older than one year; say so instead of claiming the user is free |
| Flow fails with `404 File or folder not found` | Setup step 1 files missing or not in OneDrive root |
| Flow fails with `Prefer outlook.timezone header not applied` | Time zone header ignored; the flow stops rather than write wrong times |
| Flow fails with `History file is not empty but parsed 0 events` | `calendar-history.json` was edited or corrupted; the flow stops rather than overwrite history |

## Legacy: ICS subscription

`ics_parser.py` still reads published ICS feeds via `OUTLOOK_ICS_URLS` (comma-separated) for tenants that still allow calendar publishing:

```bash
uv run ${CLAUDE_PLUGIN_ROOT}/skills/outlook-calendar/ics_parser.py --start 2026-11-18 --end 2026-11-18
```

Use it only when `OUTLOOK_ICS_URLS` is set and no `Calendar-Export` folder exists.
