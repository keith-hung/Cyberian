# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Cyberian is a monorepo of workplace productivity CLI tools and a Claude Code plugin that exposes them as skills. It contains three Go CLI applications and a Claude Code plugin with five skills.

## Repository Structure

```
├── timecard-cli/     Go CLI — timesheet management (TimeCard web scraping)
├── wedaka-cli/       Go CLI — clock-in/out attendance (REST API)
├── azuredevops-cli/  Go CLI — Azure DevOps Server projects, repos & PRs (REST API)
├── .claude/          Claude Code project config
│   ├── settings.local.json  User-specific env vars (gitignored, contains credentials)
│   └── settings.json.example  Template for settings.local.json
├── .claude-plugin/   Claude Code plugin metadata
│   ├── plugin.json       Plugin manifest
│   └── marketplace.json  Marketplace manifest
├── skills/           Skill definitions (SKILL.md per skill: timecard, wedaka, jira, outlook-calendar, azuredevops)
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
- The timecard save operation is atomic: it reconstructs the full HTML form state before applying changes, ensuring no data loss
- Note fields have character restrictions: `#$%^&*=+{}[]|?'"` are forbidden
- Entry indices are 0-9 (max 10 entries per week)
- Daily hours must not exceed 8 per day
- Only draft save is supported — never submit for approval

## Version Bump Checklist

When bumping the version (e.g., `v0.2.2` → `v0.2.3`), update the following files **in order**:

1. `.claude-plugin/plugin.json` — `"version": "X.Y.Z"`
2. `.claude-plugin/marketplace.json` — `"version": "X.Y.Z"`
3. `scripts/timecard-launcher.sh` — `VERSION="vX.Y.Z"`
4. `scripts/timecard-launcher.ps1` — `$Version = "vX.Y.Z"`
5. `scripts/wedaka-launcher.sh` — `VERSION="vX.Y.Z"`
6. `scripts/wedaka-launcher.ps1` — `$Version = "vX.Y.Z"`
7. `scripts/azuredevops-launcher.sh` — `VERSION="vX.Y.Z"`
8. `scripts/azuredevops-launcher.ps1` — `$Version = "vX.Y.Z"`
9. `README.md` — build/tag examples in the "從原始碼建置" section
10. `CHANGELOG.md` — add new version entry at the top

After committing, tag and push to trigger the release workflow:
```bash
git tag vX.Y.Z
git push origin vX.Y.Z
```
