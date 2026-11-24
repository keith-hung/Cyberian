---
name: outlook-calendar
description: "Read Outlook (Microsoft 365) calendar events from either a published ICS subscription or the OneDrive Calendar-Export files written by a Power Automate flow. Use when user asks about meetings, schedule, calendar, appointments, availability, who is attending a meeting, or a meeting's Teams link."
---

# Outlook Calendar Reader

## Overview

Two calendar sources are supported; each user sets up whichever their organization allows:

| Source | Use when | Script | Attendees, body preview, meeting link |
|--------|----------|--------|---------------------------------------|
| **ICS subscription** | The organization allows publishing the calendar from Outlook | `ics_parser.py` | Usually not included |
| **OneDrive export** | Publishing is blocked; a Power Automate flow exports the calendar to OneDrive | `calendar_reader.py` | Yes (`--details`) |

## When to Use

- "What meetings do I have today / this week / on 12/5?"
- "Am I free tomorrow afternoon?"
- "Who is attending the review meeting?" / "Send me the Teams link for ..." (export only)
- "What meetings did I have last month?"

## Choosing the Source

Always run `date` first to resolve "today", "tomorrow" or weekday names into dates.

1. `OUTLOOK_CALENDAR_SOURCE=ics` → ICS. `OUTLOOK_CALENDAR_SOURCE=export` → export.
2. Unset or `auto`: run `calendar_reader.py --status`. If it finds `Calendar-Export`, use the export (richer fields). Otherwise, if `OUTLOOK_ICS_URLS` is set, use ICS.
3. Neither configured → tell the user there are two setup options (see Setup) and ask which one their organization allows.

When the ICS source is in use and the user asks for attendees, responses or the meeting link, say the ICS feed does not provide them; do not guess.

## Source A: ICS Subscription

`ics_parser.py` fetches the published ICS feeds in `OUTLOOK_ICS_URLS` (comma-separated, one per calendar) on every query. Full reference (RRULE handling, manual parsing, debugging): [ics.md](ics.md).

```bash
I=${CLAUDE_PLUGIN_ROOT}/skills/outlook-calendar/ics_parser.py

uv run $I --start 2026-11-18 --end 2026-11-18
uv run $I --start 2026-11-16 --end 2026-11-22 --calendar "Work"   # filter by calendar name
uv run $I --start 2026-11-16 --end 2026-11-22 --format json
```

| Option | Meaning |
|--------|---------|
| `--start`, `--end` | Date range, inclusive (both required) |
| `--url URL` | ICS URL; repeatable; overrides `OUTLOOK_ICS_URLS` |
| `--calendar NAME` | Only calendars whose name contains NAME |
| `--format table\|json` | Markdown table (default) or JSON |
| `--debug` | Show event counts per calendar |

## Source B: OneDrive Export

A Power Automate flow (run under the user's own account) exports the calendar to the user's OneDrive every 30 minutes. `calendar_reader.py` reads those files locally:

| File | Range | Fields | Updated |
|------|-------|--------|---------|
| `Calendar-Export/calendar.json` | 7 days ago → 30 days ahead | full: subject, time, location, organizer, show-as, body preview, attendees with responses, meeting link | every 30 min, fully rewritten |
| `Calendar-Export/calendar-history.json` | past year | slim: subject, time, location, organizer, show-as, status | about once a day, merged; deleted meetings kept as `deleted` |

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

| Option | Meaning |
|--------|---------|
| `--start`, `--end` | Date range, inclusive; default today |
| `--format table\|json` | Markdown table (default) or JSON |
| `--details` | Add `preview`, `attendees` (name, email, type, response) and `meetingUrl` — snapshot window only |
| `--include-cancelled` | Also list cancelled meetings (adds a Status column) |
| `--dir PATH` | Calendar-Export folder; overrides auto-detection |
| `--status` | Print folder, `generatedAt`, windows, counts and staleness only |

How the two files are combined:

- Inside the snapshot window, only `calendar.json` is used (it is current; history may lag up to a day).
- Before the snapshot window, `calendar-history.json` fills in.
- Only `active` meetings are listed by default; `deleted` (history only) is never listed.
- Events covering several dates (multi-day all-day events, overnight meetings) are listed on every date they cover; timed ones show both dates in the Time column.
- An event that starts before the snapshot window but runs into it is taken from the snapshot only.
- All times are Taipei time (`+08:00`).

Warnings (stale data, dates outside the exported range) are printed to stderr, appended under the table, and included in JSON `warnings`. Relay them to the user.

`calendar.json` contains body previews, attendee emails and meeting join links. Use them only to answer the user locally; do not paste them into chat channels, web pages or other public places, and do not modify either file.

## Output Format

Both scripts print a Markdown table, for example:

```
| Date | Time | Event | Location | Show As |
|---|---|---|---|---|
| 2026-11-18 (Wed) | 10:30-11:00 | Team Standup (recurring) | Microsoft Teams Meeting | busy |
| 2026-11-20 (Fri) | All day | Company Holiday | — | free |
```

The ICS table has no Show As column (it adds a Calendar column when several calendars are configured), so treat every listed ICS event as occupied. On the export source, treat `busy`, `oof` and `tentative` as occupied and `free` / `workingElsewhere` as available. Dates outside the source's range are unknown, not free.

## Configuration

| Variable | Required | Meaning |
|----------|----------|---------|
| `OUTLOOK_CALENDAR_SOURCE` | No | `ics`, `export` or `auto` (default); see Choosing the Source |
| `OUTLOOK_ICS_URLS` | ICS only | Comma-separated published ICS URLs |
| `OUTLOOK_CALENDAR_DIR` | No | Path to the `Calendar-Export` folder. Without it the script searches `%OneDriveCommercial%`, `%OneDrive%`, `~/Library/CloudStorage/OneDrive-*/` (macOS), `/mnt/c/Users/*/OneDrive - */` (WSL) and `~/OneDrive*/` |

## Setup (once per user, pick one)

### Option A: ICS subscription

1. Outlook on the web → Settings → Calendar → **Shared calendars** → **Publish a calendar** → choose the calendar and **Can view all details** → **Publish**.
2. Copy the **ICS** link (not HTML) into `OUTLOOK_ICS_URLS`; separate several calendars with commas.

If publishing is unavailable or fails, the organization blocks it → use Option B.

### Option B: OneDrive export

1. Power Automate → **My flows** → **Import** → **Import Package (Legacy)** → upload `flow/calendar-export-flow.zip` from this skill.
2. Under Related resources pick (or create) the **Office 365 Outlook** and **OneDrive for Business** connections → **Import**.
3. Open the flow → **Save** → **Test** → **Manually**. The flow creates `Calendar-Export/` with both files in OneDrive root if they do not exist (only on a 404, so existing files are never replaced), then fills them; after that it runs every 30 minutes.

`flow/make_package.py` regenerates the package if the flow definition changes.

## Troubleshooting

| Issue | Cause / Solution |
|-------|------------------|
| ICS: `Provide --url, set OUTLOOK_ICS_URLS ...` | ICS not configured → set `OUTLOOK_ICS_URLS`, or set up the export |
| ICS: a URL fails to fetch | It is skipped with an error and the rest continue; the calendar may have been unpublished or its link reset → publish again and update `OUTLOOK_ICS_URLS`. More in [ics.md](ics.md) |
| Export: `Calendar-Export/calendar.json not found` (exit 2) | Flow not set up, OneDrive not synced, or non-standard path → set `OUTLOOK_CALENDAR_DIR` |
| Export: `cannot read ...` | OneDrive online-only placeholder → ask the user to open the file once in Finder / File Explorer |
| Export: `snapshot is stale` | The flow stopped (failed runs, expired connection, DLP), or this machine's OneDrive copy is behind → check run history in Power Automate and the sync client |
| Export: `dates ... are not covered by the export` | Range is beyond +30 days or older than one year; say so instead of claiming the user is free |
| Flow fails at `CreateSnapFile` / `CreateHistFile` | The flow could not create `Calendar-Export/` itself → create the folder with two files containing `{"events":[]}` in OneDrive root and run again |
| Flow fails with `Prefer outlook.timezone header not applied` | Time zone header ignored; the flow stops rather than write wrong times |
| Flow fails with `More than 1000 events in snapshot window` or `... in history quarter N` | One request returns at most 1000 events; the flow stops rather than write a partial file → the window must be split further in `flow/make_package.py` |
| Flow fails with `History file is not empty but parsed 0 events` | `calendar-history.json` was edited or corrupted; the flow stops rather than overwrite history |
