# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Cyberian is a monorepo of workplace productivity CLI tools and a Claude Code plugin that exposes them as skills. It contains five Go CLI applications and a Claude Code plugin with six skills.

> **Retired:** The legacy `timecard` skill was removed from the plugin (it kept mis-triggering against the rebuilt system). Its `timecard-cli/` Go source, launchers history, and build instructions are retained for reference, but it is NOT exposed as a skill, NOT built by `release.yml`/`build.sh`, and has no launcher script. For timesheet work, always use `nouveau-timecard`. Do not re-add a `skills/timecard/` directory.

## Repository Structure

```
├── timecard-cli/     Go CLI — legacy TimeCard web scraping (RETIRED from plugin; source kept, not built/shipped)
├── nouveau-timecard-cli/ Go CLI — timesheet for the rebuilt Smart Timecard (draft only)
├── wedaka-cli/       Go CLI — clock-in/out attendance (REST API)
├── azuredevops-cli/  Go CLI — Azure DevOps Server projects, repos & PRs (REST API)
├── chpw-cli/         Go CLI — change on-prem AD password via the self-service portal (interactive -i, or two-step; app/SMS OTP)
├── slip/             Go — ephemeral local secret broker (cross-platform; single package, no cobra). Hands a secret from the user's terminal into a spawned command's stdin without it entering the agent's context. Used by the change-password skill's B2 path
├── .claude/          Claude Code project config
│   ├── settings.local.json  User-specific env vars (gitignored, contains credentials)
│   └── settings.json.example  Template for settings.local.json
├── .claude-plugin/   Claude Code plugin metadata
│   ├── plugin.json       Plugin manifest
│   └── marketplace.json  Marketplace manifest
├── skills/           Skill definitions (SKILL.md per skill: nouveau-timecard, wedaka, jira, outlook-calendar, azuredevops, change-password)
├── scripts/          Launcher scripts (.sh + .ps1) + build script
└── dev/              Development notes (gitignored)
```

## Build & Run

All CLIs are Go modules (Go 1.25) using [cobra](https://github.com/spf13/cobra) for command-line parsing. Each CLI has built-in `--help` for all commands.

```bash
# Build nouveau-timecard-cli
cd nouveau-timecard-cli && go build -o nouveau-timecard .

# Build wedaka-cli
cd wedaka-cli && go build -o wedaka .

# Build azuredevops-cli
cd azuredevops-cli && go build -o azuredevops .

# Build chpw-cli
cd chpw-cli && go build -o chpw .

# Run commands directly
./nouveau-timecard-cli/nouveau-timecard <command> [flags]
./wedaka-cli/wedaka <command> [flags]
./azuredevops-cli/azuredevops <command> [flags]
./chpw-cli/chpw <command> [flags]

# Retired (source kept, not shipped): cd timecard-cli && go build -o timecard .
```

Build outputs (`nouveau-timecard-cli/nouveau-timecard`, `wedaka-cli/wedaka`, `azuredevops-cli/azuredevops`, `chpw-cli/chpw`) are gitignored.

There are no tests, linting, or CI pipelines configured.

## Architecture

### timecard-cli (RETIRED — source kept for reference, not exposed as a skill or shipped)

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

### chpw-cli

HTML form-based HTTP client for the off-network self-service AD password-change portal (antiforgery token + form POST, like nouveau-timecard-cli). One flag-driven command instead of subcommands:

- one-shot interactive: `chpw -i` (prompts for current password, OTP, and new password — for a human at a terminal)
- two-step for automation: `chpw` (step 1: posts current password via `--pass-stdin`, triggers the OTP, persists `.chpw-session.json`, prints a `next` command) then `chpw --continue --otp <code>` (step 2: submits the new password via `--pass-stdin`)
- `--method APP|SMS` (default APP: i-daka/Email; SMS: mobile text)

Commands: (default) start / -i interactive / --continue / version

Config via env vars: `CHPW_BASE_URL`, `CHPW_USERNAME` (optional), `CHPW_INSECURE` (optional, skip TLS verification). Password is only ever supplied via `--pass-stdin`; the session file stores cookies and a form token, never a password.

The `change-password` skill also covers a second, CLI-free path: it runs `skills/change-password/local-change.ps1 -Detect` first and, when the machine is domain-joined with a reachable DC, uses the local ADSI change (domain/user auto-detected from the session, overridable via `-Domain`/`-User`); otherwise it falls back to the chpw portal automatically.

### slip (ephemeral local secret broker)

`slip` is deliberately unlike the other CLIs: a single `package main` (no cobra, no
`internal/`, no JSON-only convention) with only `golang.org/x/term` beyond stdlib.
It is a broker/wrapper, not an API client, so it forwards a spawned command's
stdout/stderr/exit code verbatim. It is **cross-platform**: it coordinates over a
Unix domain socket on every OS (Windows supports AF_UNIX since Windows 10 1803 /
Server 2019) and reads the secret from the terminal device (`/dev/tty` on Unix,
`CONIN$` on Windows — see `slip/tty_unix.go` / `slip/tty_windows.go`).

- `slip daemon --timeout <s> -- <cmd> [args...]` — listens on a per-user socket
  (`$XDG_RUNTIME_DIR/slip/<ID>.sock` → `%LOCALAPPDATA%\slip` on Windows → `~/.slip`),
  prints ONLY a random 5-digit ID to stdout, blocks; on a value it pipes it into
  `<cmd>`'s stdin, runs it, forwards output/exit code, zeroes the value
  (best-effort), removes the socket, exits. One-shot.
- `slip set <ID>` — reads a secret from the terminal device with echo off and sends
  it to the daemon. The secret is never echoed or printed.

Threat model: prevents ONLY accidental exposure of a secret into an agent's readable
context (stdout/logs). It does NOT defend against a malicious same-user process
(which could read memory or the socket). No crypto, no access control by design.

Its version symbol is `main.version` (not `cmd.Version`), its module dir is `slip`
(not `slip-cli`), and its release artifact is `slip_<os>_<arch>[.exe]`. The
`change-password` skill's B2 path drives it to keep the AD password out of the
agent's context. Per-platform tests live in `slip/test/e2e.sh` and `slip/test/e2e.ps1`.

### Shared CLI patterns

All CLIs follow the same conventions:
- **Cobra-based CLI** — `cobra` handles subcommand routing, flag parsing, and auto-generated `--help`; global flags are `PersistentFlags` on the root command with env var defaults
- **JSON-only output** — stdout for success, stderr for errors (`{"success":false,"error":"..."}`)
- **Exit codes** — 0=success, 1=general, 2=auth/config, 3=validation, 4=network

### Claude Code Plugin

The plugin (`plugin.json`) registers six skills that Claude Code activates based on keyword triggers. Each skill's SKILL.md contains the full usage instructions, commands, and examples.

Launcher scripts in `scripts/` auto-download platform-appropriate binaries from GitHub Releases on first run, caching them in `.cache/`. The jira launcher also auto-initializes `jira-cli` config from env vars.

All 25 env vars across 6 skills can be centrally configured in `.claude/settings.local.json` (copy from `settings.json.example`). Shell profile exports also work and take precedence over settings.json values.

## Key Conventions

- All CLI output is machine-parseable JSON — never print plain text to stdout
- Password input uses `--pass-stdin` (piped) rather than command-line flags
- Both timecard CLIs save atomically: they reconstruct the full form state before applying changes, ensuring no data loss
- Only draft save is supported — never submit for approval (both timecard CLIs)
- Legacy `timecard-cli` only (old TimeCard backend): note fields forbid `#$%^&*=+{}[]|?'"`; entry indices are 0-9 (max 10 entries per week); daily hours must not exceed 8 per day
- `nouveau-timecard-cli` instead follows the rebuilt backend's own validation: hours in 0.5 increments, description ≤ 100 chars, project/activity required. The new draft path enforces no per-day cap and no forbidden-character rule, so those legacy limits are intentionally not ported (porting the per-day cap would also break leave sync on days that already have work)

## Public Repo Safety (Leak-Guard)

This repo (`github.com/keith-hung/Cyberian`) is **public**; once content is pushed
(and indexed/cached) a leak is effectively irreversible. Before ANY commit or push:

- **No real project/client names or internal identifiers** may appear in git-tracked
  files — not even as illustrative examples in SKILL.md, docstrings, help text, README,
  or CHANGELOG. Genericize to neutral placeholders (`Alpha` / `Beta` / `web-app`).
- **Judgment rule (grep cannot enforce this):** treat ANY company or client name as
  suspect; a **financial-industry** name is a strong red flag. When you see one, stop and
  genericize it rather than committing it.
- If a leak sits in an **unpushed** commit, **AMEND** it (do not add a follow-up commit)
  so the leaky content never enters pushed history. `dev/` is gitignored — real names may
  stay there.

A local **leak-guard hook** (`dev/hooks/leak-guard.sh`, gitignored) blocks `git commit` /
`git push` when a tracked file matches a **known-token** list (`dev/hooks/forbidden-tokens.txt`,
gitignored — the only place those tokens are stored; never commit them). Caveats: it only
fires for git commands **Claude runs via Bash** (not manual terminal commits) and only
catches the **known** list — the judgment rule above is still yours to apply.

## Version Bump Checklist

When bumping the version (e.g., `v0.2.2` → `v0.2.3`), update the following files **in order**:

1. `.claude-plugin/plugin.json` — `"version": "X.Y.Z"`
2. `.claude-plugin/marketplace.json` — `"version": "X.Y.Z"`
3. `scripts/nouveau-timecard-launcher.sh` — `VERSION="vX.Y.Z"`
4. `scripts/nouveau-timecard-launcher.ps1` — `$Version = "vX.Y.Z"`
5. `scripts/wedaka-launcher.sh` — `VERSION="vX.Y.Z"`
6. `scripts/wedaka-launcher.ps1` — `$Version = "vX.Y.Z"`
7. `scripts/azuredevops-launcher.sh` — `VERSION="vX.Y.Z"`
8. `scripts/azuredevops-launcher.ps1` — `$Version = "vX.Y.Z"`
9. `scripts/chpw-launcher.sh` — `VERSION="vX.Y.Z"`
10. `scripts/chpw-launcher.ps1` — `$Version = "vX.Y.Z"`
11. `scripts/slip-launcher.sh` — `VERSION="vX.Y.Z"`
12. `scripts/slip-launcher.ps1` — `$Version = "vX.Y.Z"`
13. `README.md` — build/tag examples in the "從原始碼建置" section
14. `CHANGELOG.md` — add new version entry at the top

> `scripts/bump-version.sh X.Y.Z` performs items 1-12 (the 4 CLI launchers + slip,
> each `.sh` + `.ps1`), 13, and 14 automatically — run it instead of editing by
> hand, then review the diff and `scripts/verify-release.sh`.

> Note: the legacy `scripts/timecard-launcher.{sh,ps1}` were removed when the timecard skill was retired — no version bump applies.

> Note: `scripts/jira-launcher.{sh,ps1}` track jira-cli's own upstream version, not the plugin version — do not bump them here.

After committing, tag and push to trigger the release workflow:
```bash
git tag vX.Y.Z
git push origin vX.Y.Z
```

## Adding a New CLI

`.github/workflows/release.yml` and `scripts/build.sh` build the CLIs from a
hardcoded `CLIS` list (it is NOT auto-discovery); `scripts/verify-release.sh`
enforces that the two lists stay in sync. When adding a new CLI, add its dir to the
`CLIS` list in BOTH files, or the release will silently ship without that CLI's
binaries and its launcher will fail to download them. Mirror the existing entries:

- ldflags inject `${MODULE}/cmd.Version`, `.Commit`, `.BuildDate` (module read from the CLI's `go.mod`)
- output name `<name>-cli_${GOOS}_${GOARCH}` (append `.exe` for windows) — must match the launcher's download URL

Also add the new CLI's launcher (`.sh` + `.ps1`) to the Version Bump Checklist
above, to `scripts/bump-version.sh`, and to the launcher/coverage loops in
`scripts/verify-release.sh`.

**Exception — `slip`:** it is not a `*-cli` module, so it is NOT in the `CLIS`
list. It has its own build lines (version symbol `main.version`, output
`slip_<os>_<arch>[.exe]`, dir `slip`) in both `release.yml` and `build.sh`. It IS
cross-platform (built for windows too) and ships both `.sh` + `.ps1` launchers, so
it rides the standard launcher loops in `verify-release.sh` / `bump-version.sh`;
only its build-coverage check in `verify-release.sh` is dedicated (dir is `slip`,
not `slip-cli`). A new `*-cli` should follow the loop pattern, not slip's build lines.
