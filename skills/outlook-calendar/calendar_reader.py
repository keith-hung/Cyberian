#!/usr/bin/env python3
# /// script
# requires-python = ">=3.9"
# dependencies = []
# ///
"""
Calendar reader for the Outlook Calendar skill.

Reads the JSON files written by the Power Automate "calendar export" flow into
OneDrive `Calendar-Export/`:

    calendar.json          snapshot, -7..+30 days, full fields, rewritten every 30 min
    calendar-history.json  slim fields, past year, merged about once a day

Inside the snapshot window only the snapshot is used (history may still hold
events deleted since its last merge); before the snapshot window, history fills in.

Usage:
    uv run calendar_reader.py                                   # today
    uv run calendar_reader.py --start 2026-11-01 --end 2026-11-30
    uv run calendar_reader.py --start 2026-11-18 --details --format json
    uv run calendar_reader.py --status                          # file freshness only

Exit codes: 0=success, 2=config (files not found), 3=validation (bad args)
"""

import argparse
import glob
import json
import os
import platform
import sys
from datetime import date, datetime, timedelta, timezone

TW = timezone(timedelta(hours=8))
SNAPSHOT = "calendar.json"
HISTORY = "calendar-history.json"
FOLDER = "Calendar-Export"
SNAPSHOT_STALE = timedelta(minutes=90)
HISTORY_STALE = timedelta(hours=30)
RECURRING_TYPES = {"occurrence", "exception"}


def candidate_dirs() -> list:
    """Possible Calendar-Export locations, most specific first."""
    env = os.environ.get("OUTLOOK_CALENDAR_DIR")
    if env:
        return [env]
    dirs = []
    for var in ("OneDriveCommercial", "OneDrive"):
        if os.environ.get(var):
            dirs.append(os.path.join(os.environ[var], FOLDER))
    home = os.path.expanduser("~")
    # macOS: business OneDrive is "OneDrive-<Org>", personal is "OneDrive-Personal"
    dirs += sorted(glob.glob(os.path.join(home, "Library", "CloudStorage", "OneDrive-*", FOLDER)))
    # WSL: business OneDrive is "OneDrive - <Org>"
    if "microsoft" in platform.release().lower():
        dirs += sorted(glob.glob(f"/mnt/c/Users/*/OneDrive - */{FOLDER}"))
        dirs += sorted(glob.glob(f"/mnt/c/Users/*/OneDrive/{FOLDER}"))
    dirs += sorted(glob.glob(os.path.join(home, "OneDrive*", FOLDER)))
    return dirs


def find_dir(explicit: str = None) -> str:
    for d in ([explicit] if explicit else candidate_dirs()):
        if d and os.path.isfile(os.path.join(d, SNAPSHOT)):
            return d
    return None


def load(path: str) -> dict:
    if not os.path.isfile(path):
        return None
    try:
        with open(path, encoding="utf-8") as f:
            return json.load(f)
    except OSError as e:
        # OneDrive "online-only" placeholders can fail to read until downloaded
        raise SystemExit(error(f"cannot read {path}: {e}. If it is an online-only OneDrive file, "
                               "open it once in Finder / File Explorer to download it.", 2))
    except json.JSONDecodeError as e:
        raise SystemExit(error(f"invalid JSON in {path}: {e}", 2))


def error(msg: str, code: int) -> int:
    print(json.dumps({"success": False, "error": msg}, ensure_ascii=False), file=sys.stderr)
    return code


def parse_ts(value: str) -> datetime:
    return datetime.fromisoformat(value).astimezone(TW)


def freshness(doc: dict, limit: timedelta, now: datetime) -> dict:
    if not doc:
        return {"present": False}
    generated = parse_ts(doc["generatedAt"]) if doc.get("generatedAt") else None
    info = {
        "present": True,
        "generatedAt": doc.get("generatedAt"),
        "windowStart": doc.get("windowStart"),
        "windowEnd": doc.get("windowEnd"),
        "count": len(doc.get("events", [])),
    }
    info["stale"] = generated is None or now - generated > limit
    return info


def expand_days(ev: dict) -> list:
    """Yield (day, ev) rows; multi-day all-day events become one row per day."""
    start, end = parse_ts(ev["start"]), parse_ts(ev["end"])
    if not ev.get("allDay"):
        return [(start.date(), ev)]
    days, d = [], start.date()
    last = max(start.date(), (end - timedelta(seconds=1)).date())  # end is exclusive
    while d <= last:
        days.append((d, ev))
        d += timedelta(days=1)
    return days


def collect(snap: dict, hist: dict, first: date, last: date, include_cancelled: bool) -> list:
    snap_start = parse_ts(snap["windowStart"]) if snap and snap.get("windowStart") else None
    events = []
    for ev in (snap or {}).get("events", []):
        events.append(dict(ev, source="snapshot"))
    for ev in (hist or {}).get("events", []):
        # Inside the snapshot window the snapshot is authoritative
        if snap_start and parse_ts(ev["start"]) >= snap_start:
            continue
        events.append(dict(ev, source="history"))

    wanted = {"active", "cancelled"} if include_cancelled else {"active"}
    rows = []
    for ev in events:
        if ev.get("status") not in wanted:
            continue
        for day, e in expand_days(ev):
            if first <= day <= last:
                rows.append((day, e))
    rows.sort(key=lambda r: (r[0], not r[1].get("allDay"), r[1]["start"]))
    return rows


def slim_attendees(attendees) -> list:
    out = []
    for a in attendees or []:
        email = a.get("emailAddress") or {}
        out.append({"name": email.get("name"), "email": email.get("address"),
                    "type": a.get("type"), "response": (a.get("status") or {}).get("response")})
    return out


def to_record(day: date, ev: dict, details: bool) -> dict:
    start, end = parse_ts(ev["start"]), parse_ts(ev["end"])
    rec = {
        "date": day.isoformat(),
        "weekday": day.strftime("%a"),
        "allDay": bool(ev.get("allDay")),
        "start": None if ev.get("allDay") else start.strftime("%H:%M"),
        "end": None if ev.get("allDay") else end.strftime("%H:%M"),
        "subject": ev.get("subject") or "",
        "location": ev.get("location") or "",
        "organizer": ev.get("organizer") or "",
        "showAs": ev.get("showAs"),
        "status": ev.get("status"),
        "recurring": ev.get("type") in RECURRING_TYPES,
        "online": bool(ev.get("online")),
        "sensitivity": ev.get("sensitivity"),
        "source": ev["source"],
    }
    if details and ev["source"] == "snapshot":
        rec["preview"] = ev.get("preview") or ""
        rec["attendees"] = slim_attendees(ev.get("attendees"))
        rec["meetingUrl"] = ev.get("meetingUrl") or ""
    return rec


def cell(text: str) -> str:
    return (text or "").replace("|", "\\|").replace("\n", " ").strip() or "—"


def format_table(records: list, include_cancelled: bool) -> str:
    if not records:
        return "No events found in the specified date range."
    head = ["Date", "Time", "Event", "Location", "Show As"] + (["Status"] if include_cancelled else [])
    lines = ["| " + " | ".join(head) + " |", "|" + "|".join("---" for _ in head) + "|"]
    for r in records:
        when = "All day" if r["allDay"] else f'{r["start"]}-{r["end"]}'
        subject = r["subject"] + (" (recurring)" if r["recurring"] else "")
        row = [f'{r["date"]} ({r["weekday"]})', when, cell(subject), cell(r["location"]), r["showAs"] or "—"]
        if include_cancelled:
            row.append(r["status"])
        lines.append("| " + " | ".join(row) + " |")
    return "\n".join(lines)


def parse_date(value: str) -> date:
    try:
        return date.fromisoformat(value)
    except ValueError:
        raise SystemExit(error(f"invalid date '{value}', expected YYYY-MM-DD", 3))


def main() -> int:
    p = argparse.ArgumentParser(description="Read Outlook calendar from the OneDrive Calendar-Export JSON files")
    p.add_argument("--start", help="Start date YYYY-MM-DD (default: today)")
    p.add_argument("--end", help="End date YYYY-MM-DD (default: same as --start)")
    p.add_argument("--format", choices=["table", "json"], default="table")
    p.add_argument("--details", action="store_true",
                   help="Include body preview, attendees and meeting link (snapshot window only; json output)")
    p.add_argument("--include-cancelled", action="store_true", help="Also list cancelled meetings")
    p.add_argument("--dir", help="Calendar-Export folder (default: $OUTLOOK_CALENDAR_DIR or auto-detect)")
    p.add_argument("--status", action="store_true", help="Only report file location and freshness")
    args = p.parse_args()

    now = datetime.now(TW)
    first = parse_date(args.start) if args.start else now.date()
    last = parse_date(args.end) if args.end else first
    if last < first:
        return error("--end is earlier than --start", 3)

    folder = find_dir(args.dir)
    if not folder:
        return error("Calendar-Export/calendar.json not found. Set OUTLOOK_CALENDAR_DIR or check that the "
                     "Power Automate calendar export flow is running and OneDrive is synced "
                     "(or use the ICS source: OUTLOOK_ICS_URLS). "
                     f"Searched: {candidate_dirs() if not args.dir else [args.dir]}", 2)

    snap = load(os.path.join(folder, SNAPSHOT))
    hist = load(os.path.join(folder, HISTORY))
    status = {"dir": folder,
              "snapshot": freshness(snap, SNAPSHOT_STALE, now),
              "history": freshness(hist, HISTORY_STALE, now)}

    warnings = []
    if status["snapshot"].get("stale"):
        warnings.append(f"snapshot is stale (generatedAt {status['snapshot'].get('generatedAt')}); "
                        "check that the Power Automate flow is still running")
    if status["history"].get("present") and status["history"].get("stale"):
        warnings.append(f"history is stale (generatedAt {status['history'].get('generatedAt')})")
    if snap and snap.get("windowEnd") and last >= parse_ts(snap["windowEnd"]).date():
        warnings.append(f"dates on or after {parse_ts(snap['windowEnd']).date()} are not covered by the export")
    oldest = hist.get("windowStart") if hist else None
    if oldest and first < parse_ts(oldest).date():
        warnings.append(f"dates before {parse_ts(oldest).date()} are not covered by the export")

    if args.status:
        print(json.dumps(dict(status, warnings=warnings), ensure_ascii=False, indent=2))
        return 0

    rows = collect(snap, hist, first, last, args.include_cancelled)
    records = [to_record(d, e, args.details) for d, e in rows]
    for w in warnings:
        print(f"warning: {w}", file=sys.stderr)
    if args.format == "json":
        print(json.dumps({"range": {"start": first.isoformat(), "end": last.isoformat()},
                          "generatedAt": status["snapshot"].get("generatedAt"),
                          "warnings": warnings, "events": records}, ensure_ascii=False, indent=2))
    else:
        print(format_table(records, args.include_cancelled))
        for w in warnings:
            print(f"\n> Warning: {w}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
