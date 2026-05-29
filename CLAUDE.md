# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Cyberian is a monorepo of workplace productivity CLI tools and a Claude Code plugin that exposes them as skills. It contains four Go CLI applications and a Claude Code plugin with six skills.

## Repository Structure

```
├── timecard-cli/     Go CLI — timesheet management (legacy TimeCard web scraping)
├── nouveau-timecard-cli/ Go CLI — timesheet for the rebuilt Smart Timecard (draft only)
├── wedaka-cli/       Go CLI — clock-in/out attendance (REST API)
├── azuredevops-cli/  Go CLI — Azure DevOps Server projects, repos & PRs (REST API)
├── .claude/          Claude Code project config
│   ├── settings.local.json  User-specific env vars (gitignored, contains credentials)
│   └── settings.json.example  Template for settings.local.json
├── .claude-plugin/   Claude Code plugin metadata
│   ├── plugin.json       Plugin manifest
│   └── marketplace.json  Marketplace manifest
├── skills/           Skill definitions (SKILL.md per skill: timecard, nouveau-timecard, wedaka, jira, outlook-calendar, azuredevops)
├── scripts/          Launcher scripts (.sh + .ps1) + build script
└── dev/              Development notes (gitignored)
```

## Build & Run

All CLIs are Go modules (Go 1.25) using [cobra](https://github.com/spf13/cobra) for command-line parsing. Each CLI has built-in `--help` for all commands.

```bash
# Build timecard-cli
cd timecard-cli && go build -o timecard .

# Build wedaka-cli
cd wedaka-cli && go build -o wedaka .

# Build azuredevops-cli
cd azuredevops-cli && go build -o azuredevops .

# Run commands directly
./timecard-cli/timecard <command> [flags]
./wedaka-cli/wedaka <command> [flags]
./azuredevops-cli/azuredevops <command> [flags]
```

Build outputs (`timecard-cli/timecard`, `wedaka-cli/wedaka`, `azuredevops-cli/azuredevops`) are gitignored.

There are no tests, linting, or CI pipelines configured.

## Architecture

### timecard-cli

Scrapes a legacy Java web app (TimeCard) by parsing HTML responses. Key flow:

1. **Session** (`internal/session/`) — cookie-based auth with auto-restore and 25-min expiry
2. **HTTP client** (`internal/httpclient/`) — cookie jar, manual redirect following, session persistence to `.timecard-session.json`
3. **Parser** (`internal/parser/`) — regex-based HTML parsing for activities, timearray grids, form fields, project options
4. **Save** (`internal/session/save.go`) — atomic save: fetch page → reconstruct full form state → apply changes → POST

Commands: `projects`, `activities`, `timesheet`, `summary`, `save`, `version`

Config via env vars (set in `.claude/settings.local.json` or shell profile): `TIMECARD_BASE_URL`, `TIMECARD_USERNAME`, `TIMECARD_PASSWORD`

### nouveau-timecard-cli

Targets the rebuilt Smart Timecard (智慧工時系統), an ASP.NET Core Razor Pages app. Key flow:

1. **Session** (`internal/session/`) — LDAP-backed form login carrying the antiforgery token + cookie, with auto-restore and 25-min expiry
2. **HTTP client** (`internal/httpclient/`) — cookie jar, manual redirect control, session persistence to `.nouveau-timecard-session.json`
3. **Parser** (`internal/parser/`) — finds the JS *assignment* of `serverTimeRows` / `activityHtmlCache` (string-aware bracket matching, ignoring Alpine references) and decodes HTML entities
4. **Save** (`internal/session/save.go`) — atomic draft save: fetch month → overlay changes → POST `handler=SaveBoth`; `sync-leave` fetches BPM leave and fills the 休假 activity

Commands: `projects`, `activities`, `timesheet`, `save`, `sync-leave`, `version`

Draft save only — there is intentionally NO submit command. Reuses the timecard env vars: `TIMECARD_BASE_URL`, `TIMECARD_USERNAME`, `TIMECARD_PASSWORD` (optional `TIMECARD_INSECURE`).

### wedaka-cli

REST API client for WeDaka attendance system. Straightforward request/response JSON.

Commands: `clock-in`, `clock-out`, `timelog`, `check-workday`, `version`

Config via env vars (set in `.claude/settings.local.json` or shell profile): `WEDAKA_API_URL`, `WEDAKA_USERNAME`, `WEDAKA_EMP_NO`, `WEDAKA_DEVICE_ID`

### azuredevops-cli

REST API client for Azure DevOps Server (on-premises, IIS Basic Auth). Manages projects, repositories, and pull requests.

Commands: `projects`, `repos`, `prs`, `my-prs`, `pr`, `pr-create`, `pr-update`, `pr-approve`, `pr-reject`, `pr-comment`, `pr-comments`, `pr-attachment`, `pr-reviewers`, `pr-add-reviewer`, `version`

Config via env vars: `AZDO_BASE_URL`, `AZDO_COLLECTION`, `AZDO_DOMAIN` (optional), `AZDO_USERNAME`, `AZDO_PASSWORD`, `AZDO_PROJECT` (optional), `AZDO_REPO` (optional), `AZDO_API_VERSION` (optional, default: 5.0-preview.1), `AZDO_INSECURE` (optional, skip TLS verification)

### Shared CLI patterns

All CLIs follow the same conventions:
- **Cobra-based CLI** — `cobra` handles subcommand routing, flag parsing, and auto-generated `--help`; global flags are `PersistentFlags` on the root command with env var defaults
- **JSON-only output** — stdout for success, stderr for errors (`{"success":false,"error":"..."}`)
- **Exit codes** — 0=success, 1=general, 2=auth/config, 3=validation, 4=network

### Claude Code Plugin

The plugin (`plugin.json`) registers five skills that Claude Code activates based on keyword triggers. Each skill's SKILL.md contains the full usage instructions, commands, and examples.

Launcher scripts in `scripts/` auto-download platform-appropriate binaries from GitHub Releases on first run, caching them in `.cache/`. The jira launcher also auto-initializes `jira-cli` config from env vars.

All 22 env vars across 5 skills can be centrally configured in `.claude/settings.local.json` (copy from `settings.json.example`). Shell profile exports also work and take precedence over settings.json values.

## Key Conventions

- All CLI output is machine-parseable JSON — never print plain text to stdout
- Password input uses `--pass-stdin` (piped) rather than command-line flags
- Both timecard CLIs save atomically: they reconstruct the full form state before applying changes, ensuring no data loss
- Only draft save is supported — never submit for approval (both timecard CLIs)
- Legacy `timecard-cli` only (old TimeCard backend): note fields forbid `#$%^&*=+{}[]|?'"`; entry indices are 0-9 (max 10 entries per week); daily hours must not exceed 8 per day
- `nouveau-timecard-cli` instead follows the rebuilt backend's own validation: hours in 0.5 increments, description ≤ 100 chars, project/activity required. The new draft path enforces no per-day cap and no forbidden-character rule, so those legacy limits are intentionally not ported (porting the per-day cap would also break leave sync on days that already have work)

## Version Bump Checklist

When bumping the version (e.g., `v0.2.2` → `v0.2.3`), update the following files **in order**:

1. `.claude-plugin/plugin.json` — `"version": "X.Y.Z"`
2. `.claude-plugin/marketplace.json` — `"version": "X.Y.Z"`
3. `scripts/timecard-launcher.sh` — `VERSION="vX.Y.Z"`
4. `scripts/timecard-launcher.ps1` — `$Version = "vX.Y.Z"`
5. `scripts/nouveau-timecard-launcher.sh` — `VERSION="vX.Y.Z"`
6. `scripts/nouveau-timecard-launcher.ps1` — `$Version = "vX.Y.Z"`
7. `scripts/wedaka-launcher.sh` — `VERSION="vX.Y.Z"`
8. `scripts/wedaka-launcher.ps1` — `$Version = "vX.Y.Z"`
9. `scripts/azuredevops-launcher.sh` — `VERSION="vX.Y.Z"`
10. `scripts/azuredevops-launcher.ps1` — `$Version = "vX.Y.Z"`
11. `README.md` — build/tag examples in the "從原始碼建置" section
12. `CHANGELOG.md` — add new version entry at the top

> Note: `scripts/jira-launcher.{sh,ps1}` track jira-cli's own upstream version, not the plugin version — do not bump them here.

After committing, tag and push to trigger the release workflow:
```bash
git tag vX.Y.Z
git push origin vX.Y.Z
```

## Adding a New CLI

`.github/workflows/release.yml` builds each CLI with a separate hardcoded `Build <name>-cli` step (it is NOT auto-discovery). When adding a new CLI, you MUST add a matching build step, or the release will silently ship without that CLI's binaries and its launcher will fail to download them. Mirror the existing steps:

- ldflags inject `${MODULE}/cmd.Version`, `.Commit`, `.BuildDate` (module read from the CLI's `go.mod`)
- output name `<name>-cli_${GOOS}_${GOARCH}` (append `.exe` for windows) — must match the launcher's download URL

Also add the new CLI's launcher (`.sh` + `.ps1`) to the Version Bump Checklist above.
