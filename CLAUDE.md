# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Cyberian is a monorepo of workplace productivity CLI tools and a Claude Code plugin that exposes them as skills. It contains two Go CLI applications and a Claude Code plugin with four skills.

## Repository Structure

```
├── timecard-cli/     Go CLI — timesheet management (TCRS web scraping)
├── wedaka-cli/       Go CLI — clock-in/out attendance (REST API)
├── .claude/          Claude Code project config
│   ├── settings.local.json  User-specific env vars (gitignored, contains credentials)
│   └── settings.json.example  Template for settings.local.json
├── .claude-plugin/   Claude Code plugin
│   ├── plugin.json       Plugin manifest (4 skills: timecard, wedaka, jira, outlook-calendar)
│   ├── skills/           Skill definitions (SKILL.md per skill)
│   └── scripts/          Launcher scripts (.sh + .ps1) that auto-download CLI binaries
└── dev/              Development notes (gitignored)
```

## Build & Run

Both CLIs are zero-dependency Go modules (Go 1.25, stdlib only — no third-party imports).

```bash
# Build timecard-cli
cd timecard-cli && go build -o timecard .

# Build wedaka-cli
cd wedaka-cli && go build -o wedaka .

# Run commands directly
./timecard-cli/timecard <command> [flags]
./wedaka-cli/wedaka <command> [flags]
```

Build outputs (`timecard-cli/timecard`, `wedaka-cli/wedaka`) are gitignored.

There are no tests, linting, or CI pipelines configured.

## Architecture

### timecard-cli

Scrapes a legacy Java web app (TCRS) by parsing HTML responses. Key flow:

1. **Session** (`internal/session/`) — cookie-based auth with auto-restore and 25-min expiry
2. **HTTP client** (`internal/httpclient/`) — cookie jar, manual redirect following, session persistence to `.timecard-session.json`
3. **Parser** (`internal/parser/`) — regex-based HTML parsing for activities, timearray grids, form fields, project options
4. **Save** (`internal/session/save.go`) — atomic save: fetch page → reconstruct full form state → apply changes → POST

Commands: `projects`, `activities`, `timesheet`, `summary`, `save`, `version`

Config via env vars (set in `.claude/settings.local.json` or shell profile): `TIMECARD_BASE_URL`, `TIMECARD_USERNAME`, `TIMECARD_PASSWORD`

### wedaka-cli

REST API client for WeDaka attendance system. Straightforward request/response JSON.

Commands: `clock-in`, `clock-out`, `timelog`, `check-workday`, `version`

Config via env vars (set in `.claude/settings.local.json` or shell profile): `WEDAKA_API_URL`, `WEDAKA_USERNAME`, `WEDAKA_EMP_NO`, `WEDAKA_DEVICE_ID`

### Shared CLI patterns

Both CLIs follow the same conventions:
- **Manual flag parsing** — no flag library; `ParseGlobalFlags()` in `cmd/root.go` handles `--flag value` pairs with env var fallbacks
- **Subcommand extraction** — `extractSubcommand()` in `main.go` pulls the command from args (flags can appear before or after)
- **JSON-only output** — stdout for success, stderr for errors (`{"success":false,"error":"..."}`)
- **Exit codes** — 0=success, 1=general, 2=auth/config, 3=validation, 4=network

### Claude Code Plugin

The plugin (`plugin.json`) registers four skills that Claude Code activates based on keyword triggers. Each skill's SKILL.md contains the full usage instructions, commands, and examples.

Launcher scripts in `.claude-plugin/scripts/` auto-download platform-appropriate binaries from GitHub Releases on first run, caching them in `.cache/`. The jira launcher also auto-initializes `jira-cli` config from env vars.

All 13 env vars across 4 skills can be centrally configured in `.claude/settings.local.json` (copy from `settings.json.example`). Shell profile exports also work and take precedence over settings.json values.

## Key Conventions

- All CLI output is machine-parseable JSON — never print plain text to stdout
- Password input uses `--pass-stdin` (piped) rather than command-line flags
- The timecard save operation is atomic: it reconstructs the full HTML form state before applying changes, ensuring no data loss
- Note fields have character restrictions: `#$%^&*=+{}[]|?'"` are forbidden
- Entry indices are 0-9 (max 10 entries per week)
- Daily hours must not exceed 8 per day
- Only draft save is supported — never submit for approval
