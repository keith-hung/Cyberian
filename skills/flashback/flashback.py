#!/usr/bin/env python3
# /// script
# requires-python = ">=3.9"
# dependencies = []
# ///
"""
flashback.py -- the dumb helper for the "flashback" skill.

Reads local Claude Code transcripts (~/.claude/projects/*/*.jsonl) and does pure
volume + arithmetic work so the agent does not have to parse tens of thousands of
raw JSONL lines. It contains ZERO project-name / keyword / classification logic:
attribution is the agent's job and lives in SKILL.md, not here.

Two subcommands:

  distill    Scan the transcript root, parse each session .jsonl, and emit a JSON
             array of compact per-session digest records to stdout.

  aggregate  Given the digests plus an agent-produced {session_id -> project} label
             map, sum active minutes into hours per (project x period). Pure,
             deterministic arithmetic.

Standard library only (JSON is stdlib); provisioned by `uv run` via the PEP 723
header above. Runs on native Windows, WSL, macOS, and Linux -- no jq/bash/awk.

Usage:
    uv run flashback.py distill [--root ~/.claude/projects] [--from YYYY-MM-DD]
        [--to YYYY-MM-DD] [--gap-minutes 30] [--tz +08:00]
        [--project-dir <encoded-dir-name>]... [--samples 5]

    uv run flashback.py aggregate --digests <digests.json|-> --labels <labels.json>
        [--period day|week|month] [--calibration '{"Beta":2.0}'] [--tz +08:00]
        [--format table|json]

Exit codes: 0 success, 1 general, 2 config, 3 validation, 4 (reserved) network.
"""

import argparse
import json
import os
import re
import sys
from datetime import date, datetime, timedelta, timezone

# --- constants -------------------------------------------------------------

# Full honesty caveat (SKILL.md section 8). Shown in the human table footer.
CAVEAT_FULL = (
    "This is active AI-interaction time, not real work hours. It cannot see "
    "meetings, reading/writing, IDE/coding-without-AI, or thinking, so it "
    "systematically under-counts."
)

# Short caveat used verbatim in the JSON output object (section 9).
CAVEAT_SHORT = (
    "active interaction time != real work hours; excludes meetings/non-AI/idle"
)

# A sample_user_messages candidate is dropped if its (stripped) text starts with
# any of these injection / system markers -- they are never genuine human turns.
INJECTION_PREFIXES = (
    "Base directory for this skill",
    "<system-reminder>",
    "<local-command-stdout>",
    "<command-name>",
    "<command-message>",
    "<local-command-caveat>",
    "Caveat:",
    "[Request interrupted",
    "This session is being continued",
    # hook-context markers
    "<user-prompt-submit-hook>",
    "<session-start-hook>",
    "<hook-",
)

SAMPLE_MAX_CHARS = 120

# Catch-all for hook / IDE / system context wrappers that arrive as user turns,
# e.g. <system-reminder>, <task-notification>, <ide_opened_file>,
# <local-command-caveat>, <user-prompt-submit-hook>. Genuine human turns almost
# never open with a lowercase XML-ish tag; erring toward exclusion is safe for a
# redaction step (privacy over completeness).
_SYSTEM_TAG_RE = re.compile(r"^<[a-z][\w-]*[ >]")


# --- timezone / timestamp helpers ------------------------------------------

def parse_tz(value):
    """Parse a '+08:00' / '-05:00' / '+0800' offset into a tzinfo."""
    s = (value or "").strip()
    m = re.match(r"^([+-])(\d{2}):?(\d{2})$", s)
    if not m:
        raise ValueError("invalid --tz offset %r (expected e.g. +08:00)" % value)
    sign = 1 if m.group(1) == "+" else -1
    hours = int(m.group(2))
    minutes = int(m.group(3))
    return timezone(sign * timedelta(hours=hours, minutes=minutes))


def normalize_ts(raw, target_tz):
    """
    Normalize a transcript timestamp like '2026-06-29T07:21:09.530Z' to an aware
    datetime in target_tz.

    Python 3.9's datetime.fromisoformat rejects the trailing 'Z' and is finicky
    about fractional seconds, so we strip fractional seconds and convert 'Z' to
    '+00:00' by hand before parsing.
    """
    if not isinstance(raw, str):
        return None
    s = raw.strip()
    if not s:
        return None
    # 'Z' -> explicit UTC offset
    if s.endswith("Z"):
        s = s[:-1] + "+00:00"
    # strip fractional seconds (the '.530' in '...09.530+00:00')
    s = re.sub(r"\.\d+", "", s)
    try:
        dt = datetime.fromisoformat(s)
    except ValueError:
        return None
    if dt.tzinfo is None:
        dt = dt.replace(tzinfo=timezone.utc)
    return dt.astimezone(target_tz)


# --- distill ----------------------------------------------------------------

def _most_common(counter):
    """Return the key with the highest count (ties: first inserted)."""
    best_key = None
    best_n = -1
    for key, n in counter.items():
        if n > best_n:
            best_key = key
            best_n = n
    return best_key


def _leaf_from_encoded(dir_name):
    """Best-effort project leaf from an encoded dir name when no cwd is present."""
    name = dir_name.lstrip("-")
    parts = [p for p in name.split("-") if p]
    return parts[-1] if parts else dir_name


def _extract_user_texts(record):
    """
    Return candidate genuine-human text strings from a 'user' record.

    Accepts message.content as a plain string, or the 'text' parts of a content
    list. Ignores tool_result and any non-text parts.
    """
    if record.get("type") != "user":
        return []
    message = record.get("message")
    if not isinstance(message, dict):
        return []
    content = message.get("content")
    texts = []
    if isinstance(content, str):
        texts.append(content)
    elif isinstance(content, list):
        for part in content:
            if isinstance(part, dict) and part.get("type") == "text":
                t = part.get("text")
                if isinstance(t, str):
                    texts.append(t)
    return texts


def _redact_sample(text):
    """Collapse whitespace and truncate; return None if it is an injection marker."""
    stripped = text.strip()
    if not stripped:
        return None
    for prefix in INJECTION_PREFIXES:
        if stripped.startswith(prefix):
            return None
    if _SYSTEM_TAG_RE.match(stripped):
        return None
    collapsed = " ".join(stripped.split())
    if not collapsed:
        return None
    if len(collapsed) > SAMPLE_MAX_CHARS:
        collapsed = collapsed[: SAMPLE_MAX_CHARS - 3].rstrip() + "..."
    return collapsed


def digest_session(file_path, encoded_dir, session_id, target_tz, gap_minutes,
                   sample_limit):
    """Parse one session .jsonl and return its digest record (or None on failure)."""
    timestamps = []            # list of aware datetimes
    cwd_counter = {}
    branch_counter = {}
    ai_title = None
    message_count = 0
    samples = []

    try:
        fh = open(file_path, "r", encoding="utf-8", errors="replace")
    except OSError as exc:
        sys.stderr.write("note: skipping unreadable file %s (%s)\n"
                         % (file_path, exc))
        return None

    with fh:
        for line in fh:
            line = line.strip()
            if not line:
                continue
            try:
                record = json.loads(line)
            except (ValueError, TypeError):
                # malformed / partial JSONL line -- skip it
                continue
            if not isinstance(record, dict):
                continue

            ts = record.get("timestamp")
            if isinstance(ts, str):
                dt = normalize_ts(ts, target_tz)
                if dt is not None:
                    timestamps.append(dt)

            cwd = record.get("cwd")
            if isinstance(cwd, str) and cwd:
                cwd_counter[cwd] = cwd_counter.get(cwd, 0) + 1

            branch = record.get("gitBranch")
            if isinstance(branch, str) and branch:
                branch_counter[branch] = branch_counter.get(branch, 0) + 1

            if ai_title is None:
                title = record.get("aiTitle")
                if isinstance(title, str) and title.strip():
                    ai_title = title.strip()

            if "message" in record:
                message_count += 1

            if len(samples) < sample_limit:
                for text in _extract_user_texts(record):
                    if len(samples) >= sample_limit:
                        break
                    redacted = _redact_sample(text)
                    if redacted:
                        samples.append(redacted)

    # --- active-time math ---------------------------------------------------
    timestamps.sort()
    active_by_date_f = {}      # local-date string -> float minutes
    for earlier, later in zip(timestamps, timestamps[1:]):
        gap_min = (later - earlier).total_seconds() / 60.0
        if 0 <= gap_min <= gap_minutes:
            key = earlier.strftime("%Y-%m-%d")
            active_by_date_f[key] = active_by_date_f.get(key, 0.0) + gap_min

    active_by_date = {}
    for key in sorted(active_by_date_f):
        active_by_date[key] = int(round(active_by_date_f[key]))
    active_minutes = sum(active_by_date.values())

    # primary local date: the date with the most active minutes (ties: earliest)
    date_local = None
    if active_by_date:
        best_key = None
        best_val = -1
        for key in sorted(active_by_date):
            if active_by_date[key] > best_val:
                best_val = active_by_date[key]
                best_key = key
        date_local = best_key
    elif timestamps:
        date_local = timestamps[0].strftime("%Y-%m-%d")

    start_local = timestamps[0].strftime("%Y-%m-%dT%H:%M") if timestamps else None
    end_local = timestamps[-1].strftime("%Y-%m-%dT%H:%M") if timestamps else None

    project_dir = _most_common(cwd_counter) if cwd_counter else ""
    if project_dir:
        dir_leaf = os.path.basename(project_dir.rstrip("/\\")) or project_dir
    else:
        dir_leaf = _leaf_from_encoded(encoded_dir)
    git_branch = _most_common(branch_counter) if branch_counter else ""

    return {
        "session_id": session_id,
        "file": encoded_dir + "/" + os.path.basename(file_path),
        "project_dir": project_dir,
        "dir_leaf": dir_leaf,
        "git_branch": git_branch,
        "date_local": date_local,
        "start_local": start_local,
        "end_local": end_local,
        "active_minutes": active_minutes,
        "active_by_date": active_by_date,
        "message_count": message_count,
        "sample_user_messages": samples,
        "session_title": ai_title,
    }


def iter_session_files(root, project_dirs):
    """
    Yield (encoded_dir_name, file_path) for every */*.jsonl under root.

    Symlinked dirs and permission errors are skipped with a short stderr note.
    """
    try:
        entries = list(os.scandir(root))
    except OSError as exc:
        sys.stderr.write("note: cannot scan root %s (%s)\n" % (root, exc))
        return
    for entry in sorted(entries, key=lambda e: e.name):
        try:
            is_dir = entry.is_dir()
        except OSError as exc:
            sys.stderr.write("note: skipping %s (%s)\n" % (entry.path, exc))
            continue
        if not is_dir:
            continue
        if project_dirs and entry.name not in project_dirs:
            continue
        try:
            sub_entries = list(os.scandir(entry.path))
        except OSError as exc:
            sys.stderr.write("note: skipping dir %s (%s)\n" % (entry.path, exc))
            continue
        for sub in sorted(sub_entries, key=lambda e: e.name):
            if not sub.name.endswith(".jsonl"):
                continue
            try:
                if not sub.is_file():
                    continue
            except OSError as exc:
                sys.stderr.write("note: skipping %s (%s)\n" % (sub.path, exc))
                continue
            yield entry.name, sub.path


def cmd_distill(args):
    try:
        target_tz = parse_tz(args.tz)
    except ValueError as exc:
        sys.stderr.write(json.dumps({"success": False, "error": str(exc)}) + "\n")
        return 3

    root = os.path.expanduser(args.root)
    if not os.path.isdir(root):
        sys.stderr.write(json.dumps(
            {"success": False, "error": "root not found: %s" % root}) + "\n")
        return 2

    project_dirs = set(args.project_dir or [])
    date_from = args.date_from
    date_to = args.date_to

    digests = []
    for encoded_dir, file_path in iter_session_files(root, project_dirs):
        session_id = os.path.basename(file_path)[: -len(".jsonl")]
        try:
            digest = digest_session(
                file_path, encoded_dir, session_id, target_tz,
                args.gap_minutes, args.samples,
            )
        except Exception as exc:  # never let one bad file kill the whole run
            sys.stderr.write("note: skipping %s (%s)\n" % (file_path, exc))
            continue
        if digest is None:
            continue

        primary = digest["date_local"]
        if date_from or date_to:
            if primary is None:
                continue
            if date_from and primary < date_from:
                continue
            if date_to and primary > date_to:
                continue
        digests.append(digest)

    sys.stdout.write(json.dumps(digests, ensure_ascii=False, indent=2) + "\n")
    return 0


# --- aggregate --------------------------------------------------------------

def period_key(date_str, period):
    """Map a 'YYYY-MM-DD' local date to a period bucket key."""
    if period == "day":
        return date_str
    try:
        y, m, d = (int(x) for x in date_str.split("-"))
        the_date = date(y, m, d)
    except (ValueError, TypeError):
        return date_str
    if period == "month":
        return "%04d-%02d" % (y, m)
    # week -> ISO year + week
    iso = the_date.isocalendar()
    return "%04d-W%02d" % (iso[0], iso[1])


def load_digests(spec):
    """Load the distill output from a file path or '-' (stdin)."""
    if spec == "-":
        data = sys.stdin.read()
    else:
        with open(spec, "r", encoding="utf-8") as fh:
            data = fh.read()
    parsed = json.loads(data)
    if not isinstance(parsed, list):
        raise ValueError("digests must be a JSON array")
    return parsed


def load_labels(spec):
    """Load the {session_id -> label} map from a file path or inline JSON."""
    if os.path.isfile(spec):
        with open(spec, "r", encoding="utf-8") as fh:
            data = fh.read()
    else:
        data = spec
    parsed = json.loads(data)
    if not isinstance(parsed, dict):
        raise ValueError("labels must be a JSON object")
    return parsed


def label_mapping(label):
    """
    Normalize a label value into {project: fraction}.

    A string -> {string: 1.0}; a dict -> its fractions; anything else / missing
    -> {'(unassigned)': 1.0} so nothing silently vanishes.
    """
    if isinstance(label, str):
        return {label: 1.0}
    if isinstance(label, dict):
        out = {}
        for proj, frac in label.items():
            try:
                out[proj] = float(frac)
            except (TypeError, ValueError):
                continue
        if out:
            return out
    return {"(unassigned)": 1.0}


def cmd_aggregate(args):
    try:
        parse_tz(args.tz)  # validate only; digest dates are already local
    except ValueError as exc:
        sys.stderr.write(json.dumps({"success": False, "error": str(exc)}) + "\n")
        return 3

    try:
        digests = load_digests(args.digests)
    except (OSError, ValueError) as exc:
        sys.stderr.write(json.dumps(
            {"success": False, "error": "cannot load digests: %s" % exc}) + "\n")
        return 2

    try:
        labels = load_labels(args.labels)
    except (OSError, ValueError) as exc:
        sys.stderr.write(json.dumps(
            {"success": False, "error": "cannot load labels: %s" % exc}) + "\n")
        return 2

    calibration = {}
    if args.calibration:
        try:
            parsed = json.loads(args.calibration)
            if not isinstance(parsed, dict):
                raise ValueError("calibration must be a JSON object")
            for proj, mult in parsed.items():
                calibration[proj] = float(mult)
        except (ValueError, TypeError) as exc:
            sys.stderr.write(json.dumps(
                {"success": False,
                 "error": "invalid --calibration: %s" % exc}) + "\n")
            return 3

    period = args.period
    # minutes accumulated per (period_key, project)
    buckets = {}
    all_dates = set()

    for digest in digests:
        if not isinstance(digest, dict):
            continue
        session_id = digest.get("session_id")
        by_date = digest.get("active_by_date") or {}
        if not isinstance(by_date, dict):
            continue
        mapping = label_mapping(labels.get(session_id))
        for date_str, minutes in by_date.items():
            try:
                minutes = float(minutes)
            except (TypeError, ValueError):
                continue
            all_dates.add(date_str)
            pkey = period_key(date_str, period)
            for proj, frac in mapping.items():
                key = (pkey, proj)
                buckets[key] = buckets.get(key, 0.0) + minutes * frac

    has_calibration = bool(calibration)
    rows = []
    raw_hours = {}
    cal_hours = {}
    for (pkey, proj), minutes in buckets.items():
        active_hours = round(minutes / 60.0, 1)
        raw_hours[(pkey, proj)] = active_hours
        row = {"period": pkey, "project": proj, "active_hours": active_hours}
        if has_calibration:
            mult = calibration.get(proj, 1.0)
            calibrated = round(minutes / 60.0 * mult, 1)
            cal_hours[(pkey, proj)] = calibrated
            row["calibrated_hours"] = calibrated
        rows.append(row)

    rows.sort(key=lambda r: (r["period"], r["project"]))

    if all_dates:
        range_from = min(all_dates)
        range_to = max(all_dates)
    else:
        range_from = None
        range_to = None

    params = {
        "gap_minutes": args.gap_minutes,
        "period": period,
        "tz": args.tz,
    }

    if args.format == "json":
        obj = {
            "range": {"from": range_from, "to": range_to},
            "params": params,
            "rows": rows,
            "caveats": [CAVEAT_SHORT],
        }
        sys.stdout.write(json.dumps(obj, ensure_ascii=False, indent=2) + "\n")
        return 0

    # --- markdown table -----------------------------------------------------
    periods = sorted({pkey for (pkey, _proj) in buckets})
    projects = sorted({proj for (_pkey, proj) in buckets})

    out_lines = []
    out_lines.append("### Active hours (raw)")
    out_lines.append("")
    out_lines.append(_render_pivot(raw_hours, periods, projects))

    if has_calibration:
        out_lines.append("")
        out_lines.append("### Calibrated hours")
        mult_desc = ", ".join(
            "%s x%s" % (p, calibration[p]) for p in sorted(calibration))
        out_lines.append("")
        out_lines.append("_Multipliers: %s (raw shown above)_" % mult_desc)
        out_lines.append("")
        out_lines.append(_render_pivot(cal_hours, periods, projects))

    range_desc = "%s..%s" % (range_from or "?", range_to or "?")
    out_lines.append("")
    out_lines.append(
        "_Params: gap<=%smin, period=%s, range %s, tz %s_"
        % (args.gap_minutes, period, range_desc, args.tz))
    out_lines.append("_Caveat: %s_" % CAVEAT_FULL)

    sys.stdout.write("\n".join(out_lines) + "\n")
    return 0


def _render_pivot(hours_lookup, periods, projects):
    """Render a projects-as-rows x periods-as-columns markdown pivot table."""
    header = "| Project | " + " | ".join(periods) + " | Total |"
    sep = "|" + "---|" * (len(periods) + 2)
    lines = [header, sep]
    for proj in projects:
        cells = []
        total = 0.0
        for pkey in periods:
            val = hours_lookup.get((pkey, proj))
            if val:
                total += val
                cells.append("%.1f" % val)
            else:
                cells.append("")
        lines.append("| " + proj + " | " + " | ".join(cells)
                     + " | %.1f |" % total)
    # totals row
    tcells = []
    grand = 0.0
    for pkey in periods:
        col_sum = sum(hours_lookup.get((pkey, p), 0.0) for p in projects)
        grand += col_sum
        tcells.append("%.1f" % col_sum)
    lines.append("| **Total** | " + " | ".join(tcells) + " | %.1f |" % grand)
    return "\n".join(lines)


# --- CLI --------------------------------------------------------------------

def build_parser():
    parser = argparse.ArgumentParser(
        prog="flashback.py",
        description="Dumb helper for the flashback skill: distill transcripts "
                    "and aggregate active time. No classification logic.",
    )
    sub = parser.add_subparsers(dest="command")

    p_distill = sub.add_parser(
        "distill",
        help="Scan transcripts and emit per-session digest records (JSON array).",
    )
    p_distill.add_argument(
        "--root", default="~/.claude/projects",
        help="transcript root (default: ~/.claude/projects)")
    p_distill.add_argument(
        "--from", dest="date_from", default=None,
        help="inclusive start local date YYYY-MM-DD (filter by primary date)")
    p_distill.add_argument(
        "--to", dest="date_to", default=None,
        help="inclusive end local date YYYY-MM-DD (filter by primary date)")
    p_distill.add_argument(
        "--gap-minutes", type=float, default=30.0,
        help="active-time gap threshold in minutes (default: 30)")
    p_distill.add_argument(
        "--tz", default="+08:00", help="local tz offset (default: +08:00)")
    p_distill.add_argument(
        "--project-dir", action="append", default=None,
        help="restrict to specific encoded project dir names (repeatable)")
    p_distill.add_argument(
        "--samples", type=int, default=5,
        help="N sample user messages per session (default: 5)")
    p_distill.set_defaults(func=cmd_distill)

    p_agg = sub.add_parser(
        "aggregate",
        help="Sum digest active time into hours per (project x period).",
    )
    p_agg.add_argument(
        "--digests", required=True,
        help="distill output: file path or '-' for stdin")
    p_agg.add_argument(
        "--labels", required=True,
        help="{session_id: project} map: file path or inline JSON")
    p_agg.add_argument(
        "--period", choices=["day", "week", "month"], default="day",
        help="period granularity (default: day)")
    p_agg.add_argument(
        "--calibration", default=None,
        help="optional per-project multipliers, e.g. '{\"Beta\":2.0}'")
    p_agg.add_argument(
        "--tz", default="+08:00", help="local tz offset (default: +08:00)")
    p_agg.add_argument(
        "--format", choices=["table", "json"], default="table",
        help="output format (default: table)")
    # Metadata-only echo so the JSON params / table footer can report the gap
    # threshold used at distill time (not carried in the digest records).
    p_agg.add_argument(
        "--gap-minutes", type=float, default=30.0,
        help="gap threshold to echo in params/footer (default: 30)")
    p_agg.set_defaults(func=cmd_aggregate)

    return parser


def main(argv=None):
    parser = build_parser()
    args = parser.parse_args(argv)
    if not getattr(args, "command", None):
        parser.print_help(sys.stderr)
        return 1
    return args.func(args)


if __name__ == "__main__":
    sys.exit(main())
