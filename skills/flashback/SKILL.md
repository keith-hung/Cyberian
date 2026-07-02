---
name: flashback
description: >
  Estimate how many active hours went to which project from your local Claude Code
  transcripts — human table + JSON, fully local. Use when the user asks to reconstruct
  work hours / effort per project or per period for a timesheet (工時填報), "how long did
  I spend on X", a worklog, or time attribution.
---

# Flashback

Reconstruct **how many active hours went to which project, per period (day/week/month)**
from the local Claude Code transcripts (`~/.claude/projects/*/*.jsonl`), and emit a human
table plus a machine-readable JSON. Fully local, decoupled from any timecard system.

This skill is your work-life flashing before your eyes, at timesheet o'clock: filling a
timesheet triggers a *flashback* of everything you did that month, so this skill replays
your transcript reel and tallies it for you.

## Overview

Attribution is the hard part, and it is user/company-specific, so it stays with the agent.
The Python helper (`flashback.py`) is deliberately dumb: it only scans files, parses JSONL,
does timestamp/active-time arithmetic, and — once you hand it a `{session_id -> project}`
label map — sums the hours. **The helper never classifies and knows no project names.** All
judgment (grouping sessions into projects, splitting mixed sessions, honesty caveats) lives
here in the prompt.

## When to Use

- User asks to reconstruct or estimate work hours / effort spent per project for a period.
- User asks "how long did I spend on X" or wants a worklog for a timesheet (工時填報).
- User wants time attribution across projects from their Claude Code history.

## Prerequisites

**Prerequisite**: [uv](https://docs.astral.sh/uv/) must be installed. The helper is a
PEP 723, standard-library-only Python script; `uv` provisions the interpreter on first run.

**This skill adds no environment variables.** Defaults you can rely on:

| Flag | Default | Meaning |
|------|---------|---------|
| `--root` | `~/.claude/projects` | Where the transcript `.jsonl` files live |
| `--tz` | `+08:00` | Local timezone offset used for date bucketing |
| `--gap-minutes` | `30` | Active-time gap threshold (larger gaps count as "away") |
| `--period` | `day` | Aggregation granularity (`day` \| `week` \| `month`) |

## Method & Step Sequence

Follow these steps in order. Do not skip the attribution confirmation (step 3).

1. **Distill** — run `flashback.py distill` for the requested date range. This scans the
   transcripts and emits one compact digest record per session (NOT the raw content). Save
   the JSON array to a temp file so you can feed it to `aggregate` later.
2. **Read the digests** — read the distilled records. Each carries the signals you need to
   attribute: `dir_leaf`, `session_title`, `sample_user_messages`, `git_branch`, and the
   `active_minutes` / `active_by_date` volume.
3. **Attribute (agent judgment)** — assign each session to a project:
   - **If the user gave project names and/or keywords**, use them as the seed: map each
     session to one of those projects using the digest signals plus the keywords as hints.
   - **If the user gave nothing**, propose a grouping yourself from the digest context
     (`dir_leaf`, `session_title`, `sample_user_messages`, `git_branch`) and **confirm it
     with the user once** before aggregating (e.g. "I see dirs A, B, C — group A+B as
     project X?").
   - The directory (`dir_leaf`) is just **one signal**, not a hard rule — one folder can
     hold several projects. If a session clearly spans two projects, split its active
     minutes proportionally via the split form of the labels map and say so.
4. **Build the labels map** — write a `{session_id -> project | split}` JSON to a temp file
   (in your scratchpad / working temp dir). A value is either a string (`"Alpha"`) or a split
   object (`{"Alpha": 0.5, "Beta": 0.5}`). Sessions you leave out land under `"(unassigned)"`
   so nothing silently vanishes.
5. **Aggregate** — run `flashback.py aggregate` with the digests + labels map to produce the
   human table and the JSON. Always print the table with its params/caveat footer.

## Subcommands

### `distill` — scan transcripts, emit per-session digests

```bash
uv run ${CLAUDE_PLUGIN_ROOT}/skills/flashback/flashback.py distill \
  --from 2026-06-01 \
  --to   2026-06-30 \
  > /path/to/scratchpad/digests.json
```

All flags:

```bash
uv run ${CLAUDE_PLUGIN_ROOT}/skills/flashback/flashback.py distill \
  [--root ~/.claude/projects]           # default ~/.claude/projects
  [--from YYYY-MM-DD] [--to YYYY-MM-DD]  # inclusive local-date range (by session primary date)
  [--gap-minutes 30]                     # active-time gap threshold, default 30
  [--tz +08:00]                          # local tz offset, default +08:00
  [--project-dir <encoded-dir-name>]...  # optional: restrict to specific project dirs (repeatable)
  [--samples 5]                          # N sample user messages per session, default 5
# -> stdout: JSON array of digest records
```

Each digest record looks like:

```json
{
  "session_id": "<uuid>",
  "file": "<project-dir>/<uuid>.jsonl",
  "project_dir": "/home/you/repos/web-app",
  "dir_leaf": "web-app",
  "git_branch": "main",
  "date_local": "2026-06-29",
  "start_local": "2026-06-29T15:21",
  "end_local": "2026-06-29T18:44",
  "active_minutes": 366,
  "active_by_date": {"2026-06-29": 366},
  "message_count": 2720,
  "sample_user_messages": ["..."],
  "session_title": "..."
}
```

### `aggregate` — sum hours per (project x period)

```bash
uv run ${CLAUDE_PLUGIN_ROOT}/skills/flashback/flashback.py aggregate \
  --digests /path/to/scratchpad/digests.json \
  --labels  /path/to/scratchpad/labels.json \
  --period  day \
  --format  table
```

All flags:

```bash
uv run ${CLAUDE_PLUGIN_ROOT}/skills/flashback/flashback.py aggregate \
  --digests <digests.json|->                  # distill output (file or stdin)
  --labels  <labels.json>                     # {session_id: "Alpha"} or {session_id: {"Alpha":0.5,"Beta":0.5}}
  [--period day|week|month]                   # default day
  [--calibration '{"Beta":2.0}']               # optional per-project multipliers; raw value still shown
  [--tz +08:00]                               # default +08:00
  [--format table|json]                       # default table
  [--gap-minutes 30]                          # echoed in params/footer only, default 30
# -> stdout: markdown table (default) or the JSON spec object
```

- `day` keys are `YYYY-MM-DD`, `week` keys are ISO weeks (e.g. `2026-W27`), `month` keys are
  `YYYY-MM`.
- With `--calibration`, both `active_hours` (raw) and `calibrated_hours` are shown; the raw
  value is never hidden.
- Run `aggregate` twice (`--format table`, then `--format json`) when you need both the
  human table and the downstream JSON.

## Mandatory Rules (enforce every run)

- **Always print the caveat.** Never present the numbers as billable hours. Print verbatim:
  > This is active AI-interaction time, not real work hours. It cannot see meetings,
  > reading/writing, IDE/coding-without-AI, or thinking, so it systematically under-counts.
- **State the known under-count bias.** Interaction-heavy work (drafting/analysis with the
  model) tracks real hours well; read/write/meeting-heavy work is badly under-counted —
  expect roughly **2x** under on those. Offer `--calibration` if the user has a trusted total
  for a known period to back out a per-project multiplier.
- **Note the meetings gap.** Meeting-only projects never appear in transcripts, so they are
  invisible here. *Suggest* supplementing from a calendar source (the `outlook-calendar`
  skill), but do **NOT** do it in this skill — it is out of scope.
- **Keep descriptive notes neutral and defensible.** Output can reach a manager/HR if fed
  into a timesheet, so any note must be neutral and defensible. Avoid loaded or self-
  incriminating wording; default to neutral descriptions.
- **100% local. Never upload transcript content anywhere.** Keep only project names, hours,
  and short sample snippets in output — never dump raw conversations.
- **The helper never classifies.** All project attribution is your judgment (optionally
  seeded by the user); the helper only scans, parses, and sums.

## Worked Example

Request: "Reconstruct my June hours for the Alpha and Beta projects."

**1. Distill the range:**

```bash
uv run ${CLAUDE_PLUGIN_ROOT}/skills/flashback/flashback.py distill \
  --from 2026-06-01 --to 2026-06-30 \
  > /path/to/scratchpad/digests.json
```

**2-3. Read the digests and attribute.** The user named Alpha and Beta, so seed with those.
Suppose one session under `dir_leaf: Alpha-billing` clearly spans both projects — split it.

**4. Write the labels map** to `/path/to/scratchpad/labels.json`:

```json
{
  "a1b2c3d4-...": "Alpha",
  "e5f6g7h8-...": "Beta",
  "9i0j1k2l-...": {"Alpha": 0.5, "Beta": 0.5}
}
```

**5. Aggregate** (table, then JSON):

```bash
uv run ${CLAUDE_PLUGIN_ROOT}/skills/flashback/flashback.py aggregate \
  --digests /path/to/scratchpad/digests.json \
  --labels  /path/to/scratchpad/labels.json \
  --period  day --format table

uv run ${CLAUDE_PLUGIN_ROOT}/skills/flashback/flashback.py aggregate \
  --digests /path/to/scratchpad/digests.json \
  --labels  /path/to/scratchpad/labels.json \
  --period  day --format json
```

**Example table output** (params + caveat footer required):

```
| Project | 2026-06-10 | 2026-06-11 | Total |
|---------|-----------:|-----------:|------:|
| Alpha   |        2.8 |        3.1 |   5.9 |
| Beta    |        3.2 |        0.0 |   3.2 |

params: gap_minutes=30, period=day, range=2026-06-01..2026-06-30, tz=+08:00
caveat: This is active AI-interaction time, not real work hours. It cannot see meetings,
reading/writing, IDE/coding-without-AI, or thinking, so it systematically under-counts.
```

**Example JSON output:**

```json
{
  "range": {"from": "2026-06-01", "to": "2026-06-30"},
  "params": {"gap_minutes": 30, "period": "day", "tz": "+08:00"},
  "rows": [
    {"period": "2026-06-10", "project": "Alpha", "active_hours": 2.8},
    {"period": "2026-06-10", "project": "Beta", "active_hours": 3.2}
  ],
  "caveats": ["active interaction time != real work hours; excludes meetings/non-AI/idle"]
}
```

When `--calibration` is applied, each row also carries `calibrated_hours` alongside the raw
`active_hours`.
