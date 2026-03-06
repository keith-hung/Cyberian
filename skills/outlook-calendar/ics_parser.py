#!/usr/bin/env python3
# /// script
# requires-python = ">=3.9"
# dependencies = ["python-dateutil"]
# ///
"""
ICS Parser for Outlook Calendar Skill

Parses ICS content and expands recurring events (RRULE) within a date range.
Supports multiple calendars via OUTLOOK_ICS_URLS (comma-separated).

Usage:
    # From env var (multiple calendars)
    uv run ics_parser.py --start 2025-12-01 --end 2025-12-31

    # From explicit URLs
    uv run ics_parser.py --url "https://..." --url "https://..." --start 2025-12-01 --end 2025-12-31

    # From stdin
    curl -s "https://..." | uv run ics_parser.py --start 2025-12-01 --end 2025-12-31

    # Filter by calendar name
    uv run ics_parser.py --start 2025-12-01 --end 2025-12-31 --calendar "Work"

    # Output formats
    uv run ics_parser.py --start 2025-12-01 --end 2025-12-31 --format json

Dependencies:
    - python-dateutil (auto-installed by uv)
"""

import argparse
import json
import os
import re
import sys
import urllib.request
from datetime import datetime, timedelta
from typing import Optional

# Try to import dateutil, provide fallback if not available
try:
    from dateutil.rrule import rrulestr
    HAS_DATEUTIL = True
except ImportError:
    HAS_DATEUTIL = False


def parse_ics_datetime(dt_line: str) -> Optional[datetime]:
    """
    Parse DTSTART/DTEND from ICS, handling all Outlook formats.

    Formats:
    - UTC: DTSTART:20251211T060000Z
    - With TZID: DTSTART;TZID=Taipei Standard Time:20250424T103000
    - All-day: DTSTART;VALUE=DATE:20251225
    """
    if not dt_line:
        return None

    # Extract datetime portion using regex
    match = re.search(r'(\d{8}T?\d{0,6}Z?)', dt_line)
    if not match:
        return None

    dt_str = match.group(1)
    try:
        if 'T' in dt_str:
            if dt_str.endswith('Z'):
                # UTC format: convert to Taipei (UTC+8)
                dt = datetime.strptime(dt_str, '%Y%m%dT%H%M%SZ')
                dt = dt + timedelta(hours=8)
            else:
                # Local time (TZID format): use as-is
                dt = datetime.strptime(dt_str, '%Y%m%dT%H%M%S')
        else:
            # All-day event
            dt = datetime.strptime(dt_str, '%Y%m%d')
        return dt
    except ValueError:
        return None


def parse_vevent(vevent_text: str) -> Optional[dict]:
    """Parse a single VEVENT block into a dictionary."""
    lines = vevent_text.strip().split('\n')
    event = {}

    # Handle line continuations (lines starting with space/tab)
    unfolded_lines = []
    for line in lines:
        if line.startswith(' ') or line.startswith('\t'):
            if unfolded_lines:
                unfolded_lines[-1] += line[1:]
        else:
            unfolded_lines.append(line)

    for line in unfolded_lines:
        line = line.strip()
        if line.startswith('SUMMARY'):
            event['summary'] = line.split(':', 1)[-1] if ':' in line else ''
        elif line.startswith('DTSTART'):
            event['dtstart_raw'] = line
            event['dtstart'] = parse_ics_datetime(line)
        elif line.startswith('DTEND'):
            event['dtend_raw'] = line
            event['dtend'] = parse_ics_datetime(line)
        elif line.startswith('RRULE'):
            event['rrule'] = line
        elif line.startswith('LOCATION'):
            event['location'] = line.split(':', 1)[-1] if ':' in line else ''
        elif line.startswith('DESCRIPTION'):
            event['description'] = line.split(':', 1)[-1] if ':' in line else ''
        elif line.startswith('STATUS'):
            event['status'] = line.split(':', 1)[-1] if ':' in line else ''
        elif line.startswith('UID'):
            event['uid'] = line.split(':', 1)[-1] if ':' in line else ''

    # Skip if no valid start time
    if not event.get('dtstart'):
        return None

    return event


def expand_rrule_dateutil(event: dict, range_start: datetime, range_end: datetime) -> list:
    """Expand recurring event using dateutil."""
    if not HAS_DATEUTIL:
        return []

    dtstart = event['dtstart']
    rrule_str = event['rrule']

    # Calculate duration
    duration = timedelta(hours=1)  # default
    if event.get('dtend'):
        duration = event['dtend'] - dtstart

    try:
        rule = rrulestr(rrule_str, dtstart=dtstart)
        occurrences = []
        for dt in rule.between(range_start, range_end, inc=True):
            occurrence = event.copy()
            occurrence['dtstart'] = dt
            occurrence['dtend'] = dt + duration
            occurrence['is_recurring'] = True
            occurrences.append(occurrence)
        return occurrences
    except Exception as e:
        # If RRULE parsing fails, return empty
        return []


def expand_rrule_fallback(event: dict, range_start: datetime, range_end: datetime) -> list:
    """
    Fallback RRULE expansion without dateutil.
    Supports: FREQ=WEEKLY with BYDAY and INTERVAL
    """
    dtstart = event['dtstart']
    rrule_str = event.get('rrule', '')

    # Parse RRULE components
    rrule_parts = {}
    for part in rrule_str.replace('RRULE:', '').split(';'):
        if '=' in part:
            key, value = part.split('=', 1)
            rrule_parts[key] = value

    freq = rrule_parts.get('FREQ', '')
    interval = int(rrule_parts.get('INTERVAL', 1))
    byday = rrule_parts.get('BYDAY', '')
    until_str = rrule_parts.get('UNTIL', '')
    count = int(rrule_parts.get('COUNT', 0))

    # Parse UNTIL if present
    until = None
    if until_str:
        until = parse_ics_datetime(f'DTSTART:{until_str}')

    # Calculate duration
    duration = timedelta(hours=1)
    if event.get('dtend'):
        duration = event['dtend'] - dtstart

    occurrences = []
    day_map = {'MO': 0, 'TU': 1, 'WE': 2, 'TH': 3, 'FR': 4, 'SA': 5, 'SU': 6}

    if freq == 'WEEKLY' and byday:
        target_days = [day_map[d.strip()] for d in byday.split(',') if d.strip() in day_map]

        current = range_start
        occurrence_count = 0
        while current <= range_end:
            if until and current > until:
                break
            if count and occurrence_count >= count:
                break

            if current.weekday() in target_days and current >= dtstart:
                # Check interval
                week_diff = (current - dtstart).days // 7
                if week_diff % interval == 0:
                    # Match the time from dtstart
                    dt = current.replace(
                        hour=dtstart.hour,
                        minute=dtstart.minute,
                        second=dtstart.second
                    )
                    if range_start <= dt <= range_end:
                        occurrence = event.copy()
                        occurrence['dtstart'] = dt
                        occurrence['dtend'] = dt + duration
                        occurrence['is_recurring'] = True
                        occurrences.append(occurrence)
                        occurrence_count += 1

            current += timedelta(days=1)

    elif freq == 'DAILY':
        current = max(dtstart, range_start)
        occurrence_count = 0
        while current <= range_end:
            if until and current > until:
                break
            if count and occurrence_count >= count:
                break

            day_diff = (current - dtstart).days
            if day_diff % interval == 0:
                dt = current.replace(
                    hour=dtstart.hour,
                    minute=dtstart.minute,
                    second=dtstart.second
                )
                occurrence = event.copy()
                occurrence['dtstart'] = dt
                occurrence['dtend'] = dt + duration
                occurrence['is_recurring'] = True
                occurrences.append(occurrence)
                occurrence_count += 1

            current += timedelta(days=1)

    return occurrences


def expand_rrule(event: dict, range_start: datetime, range_end: datetime) -> list:
    """Expand recurring event, using dateutil if available."""
    if HAS_DATEUTIL:
        result = expand_rrule_dateutil(event, range_start, range_end)
        if result:
            return result
    # Fallback
    return expand_rrule_fallback(event, range_start, range_end)


def extract_calendar_name(ics_content: str) -> str:
    """Extract calendar name from X-WR-CALNAME property."""
    match = re.search(r'X-WR-CALNAME:(.*)', ics_content)
    if match:
        return match.group(1).strip()
    return ''


def parse_ics(ics_content: str, range_start: datetime, range_end: datetime,
              calendar_name: str = '') -> list:
    """Parse ICS content and return events within the date range."""
    # Split into VEVENT blocks
    vevent_pattern = re.compile(r'BEGIN:VEVENT(.*?)END:VEVENT', re.DOTALL)
    vevent_blocks = vevent_pattern.findall(ics_content)

    all_events = []

    for block in vevent_blocks:
        event = parse_vevent(block)
        if not event:
            continue

        event['calendar'] = calendar_name

        if event.get('rrule'):
            # Recurring event: expand within range
            occurrences = expand_rrule(event, range_start, range_end)
            all_events.extend(occurrences)
        else:
            # Single event: check if within range
            event['is_recurring'] = False
            dtstart = event['dtstart']
            if range_start <= dtstart <= range_end:
                all_events.append(event)

    return all_events


def format_event_table(events: list, show_calendar: bool = False) -> str:
    """Format events as a markdown table."""
    if not events:
        return "No events found in the specified date range."

    lines = []
    if show_calendar:
        lines.append("| Date | Time | Event | Location | Calendar |")
        lines.append("|------|------|-------|----------|----------|")
    else:
        lines.append("| Date | Time | Event | Location |")
        lines.append("|------|------|-------|----------|")

    for event in events:
        dtstart = event['dtstart']
        dtend = event.get('dtend')

        date_str = dtstart.strftime('%Y-%m-%d (%a)')

        if dtend and dtstart.date() == dtend.date():
            time_str = f"{dtstart.strftime('%H:%M')}-{dtend.strftime('%H:%M')}"
        elif dtend:
            time_str = f"{dtstart.strftime('%H:%M')}-{dtend.strftime('%m/%d %H:%M')}"
        else:
            time_str = dtstart.strftime('%H:%M')

        # All-day event
        if dtstart.hour == 0 and dtstart.minute == 0:
            if not dtend or (dtend.hour == 0 and dtend.minute == 0):
                time_str = "All day"

        summary = event.get('summary', '(No title)')
        location = event.get('location', '')

        # Add recurring indicator
        if event.get('is_recurring'):
            summary = f"🔄 {summary}"

        if show_calendar:
            calendar = event.get('calendar', '')
            lines.append(f"| {date_str} | {time_str} | {summary} | {location} | {calendar} |")
        else:
            lines.append(f"| {date_str} | {time_str} | {summary} | {location} |")

    return '\n'.join(lines)


def format_event_json(events: list, show_calendar: bool = False) -> str:
    """Format events as JSON."""
    output = []
    for event in events:
        entry = {
            'date': event['dtstart'].strftime('%Y-%m-%d'),
            'start': event['dtstart'].strftime('%H:%M'),
            'end': event['dtend'].strftime('%H:%M') if event.get('dtend') else None,
            'summary': event.get('summary', ''),
            'location': event.get('location', ''),
            'is_recurring': event.get('is_recurring', False),
        }
        if show_calendar:
            entry['calendar'] = event.get('calendar', '')
        output.append(entry)
    return json.dumps(output, indent=2, ensure_ascii=False)


def fetch_ics(url: str) -> str:
    """Fetch ICS content from URL."""
    req = urllib.request.Request(url, headers={'User-Agent': 'Mozilla/5.0'})
    with urllib.request.urlopen(req, timeout=30) as response:
        return response.read().decode('utf-8')


def main():
    parser = argparse.ArgumentParser(
        description='Parse ICS and expand recurring events',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  %(prog)s --start 2025-12-01 --end 2025-12-31
  %(prog)s --url "https://..." --url "https://..." --start 2025-12-01 --end 2025-12-31
  %(prog)s --start 2025-12-01 --end 2025-12-31 --calendar "Work"
  %(prog)s --start 2025-12-01 --end 2025-12-31 --format json
  cat calendar.ics | %(prog)s --start 2025-12-01 --end 2025-12-31
        """
    )
    parser.add_argument('--url', action='append', dest='urls',
                        help='ICS URL to fetch (can be specified multiple times)')
    parser.add_argument('--start', required=True, help='Start date (YYYY-MM-DD)')
    parser.add_argument('--end', required=True, help='End date (YYYY-MM-DD)')
    parser.add_argument('--calendar', help='Filter by calendar name (substring match)')
    parser.add_argument('--format', choices=['table', 'json'], default='table',
                        help='Output format (default: table)')
    parser.add_argument('--debug', action='store_true', help='Show debug info')

    args = parser.parse_args()

    # Parse date range
    try:
        range_start = datetime.strptime(args.start, '%Y-%m-%d')
        range_end = datetime.strptime(args.end, '%Y-%m-%d').replace(hour=23, minute=59, second=59)
    except ValueError as e:
        print(f"Error: Invalid date format. Use YYYY-MM-DD. ({e})", file=sys.stderr)
        sys.exit(1)

    # Resolve ICS URLs: --url flags > OUTLOOK_ICS_URLS env var
    urls = args.urls or []
    if not urls:
        env_urls = os.environ.get('OUTLOOK_ICS_URLS', '')
        if env_urls:
            urls = [u.strip() for u in env_urls.split(',') if u.strip()]

    # Collect all events from all sources
    all_events = []

    if urls:
        for i, url in enumerate(urls):
            try:
                ics_content = fetch_ics(url)
            except Exception as e:
                print(f"Error: Failed to fetch ICS from URL {i + 1}. ({e})", file=sys.stderr)
                continue

            cal_name = extract_calendar_name(ics_content) or f"Calendar {i + 1}"

            if args.debug:
                vevent_count = ics_content.count('BEGIN:VEVENT')
                rrule_count = ics_content.count('RRULE:')
                print(f"Debug [{cal_name}]: Total VEVENTs: {vevent_count}", file=sys.stderr)
                print(f"Debug [{cal_name}]: Recurring events: {rrule_count}", file=sys.stderr)

            events = parse_ics(ics_content, range_start, range_end, cal_name)
            all_events.extend(events)

    elif not sys.stdin.isatty():
        ics_content = sys.stdin.read()
        cal_name = extract_calendar_name(ics_content) or 'Calendar'

        if args.debug:
            vevent_count = ics_content.count('BEGIN:VEVENT')
            rrule_count = ics_content.count('RRULE:')
            print(f"Debug [{cal_name}]: Total VEVENTs: {vevent_count}", file=sys.stderr)
            print(f"Debug [{cal_name}]: Recurring events: {rrule_count}", file=sys.stderr)

        events = parse_ics(ics_content, range_start, range_end, cal_name)
        all_events.extend(events)
    else:
        print("Error: Provide --url, set OUTLOOK_ICS_URLS env var, or pipe ICS content via stdin",
              file=sys.stderr)
        sys.exit(1)

    if args.debug:
        print(f"Debug: dateutil available: {HAS_DATEUTIL}", file=sys.stderr)

    # Filter by calendar name
    if args.calendar:
        filter_name = args.calendar.lower()
        all_events = [e for e in all_events if filter_name in e.get('calendar', '').lower()]

    # Sort by start time
    all_events.sort(key=lambda e: e['dtstart'])

    if args.debug:
        print(f"Debug: Events in range: {len(all_events)}", file=sys.stderr)

    # Determine if we should show calendar column
    unique_calendars = set(e.get('calendar', '') for e in all_events)
    show_calendar = len(unique_calendars) > 1 or args.calendar is not None

    # Output
    if args.format == 'json':
        print(format_event_json(all_events, show_calendar))
    else:
        print(format_event_table(all_events, show_calendar))


if __name__ == '__main__':
    main()
