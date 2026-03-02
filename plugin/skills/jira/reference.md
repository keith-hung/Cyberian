# jira-cli Command Reference

Full reference for all `jira` CLI commands and flags.
Source: https://github.com/ankitpokhrel/jira-cli

> **Note:** The binary is managed by the launcher script (`jira-launcher.sh` on Linux/macOS, `jira-launcher.ps1` on Windows). No manual installation needed.

## Initial Setup

**Linux / macOS:**
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/jira-launcher.sh init
```

**Windows (PowerShell):**
```powershell
& "${env:CLAUDE_PLUGIN_ROOT}/scripts/jira-launcher.ps1" init
```

Prompts for:
- JIRA server URL (e.g., `https://company.atlassian.net`)
- Login email
- API token (generate at https://id.atlassian.com/manage-profile/security/api-tokens)
- Default project, board

## Environment Variables

| Variable | Description |
|----------|-------------|
| `JIRA_API_TOKEN` | API token for authentication |
| `JIRA_AUTH_TYPE` | Auth type: `basic` (default), `bearer`, `mtls` |
| `JIRA_CONFIG_FILE` | Path to config file |

---

## Issue Commands

### `jira issue list`

List issues in the project.

| Flag | Short | Description |
|------|-------|-------------|
| `--status` | `-s` | Filter by status |
| `--priority` | `-y` | Filter by priority |
| `--assignee` | `-a` | Filter by assignee |
| `--reporter` | `-r` | Filter by reporter |
| `--label` | `-l` | Filter by label |
| `--type` | `-t` | Filter by issue type |
| `--resolution` | `-R` | Filter by resolution |
| `--watching` | `-w` | Filter by watching user |
| `--project` | `-p` | Specify project key |
| `--created` | | Filter by creation date (e.g., `-10d`) |
| `--updated` | | Filter by update date |
| `--jql` | `-q` | Raw JQL query |
| `--order-by` | | Order by field (default: `created`) |
| `--reverse` | | Reverse sort order |
| `--history` | | Show recently viewed issues |
| `--plain` | | Plain output (no UI) |
| `--raw` | | JSON output |
| `--csv` | | CSV output |
| `--table` | | Table output |
| `--columns` | | Specify columns |
| `--paginate` | | Page size |

**Examples:**
```bash
# Issues assigned to me, in progress
jira issue list -a$(jira me --plain) -s "In Progress" --plain

# High priority bugs created in last 7 days
jira issue list -t Bug -y High --created -7d --plain

# Custom JQL
jira issue list --jql "project = PROJ AND fixVersion = '2.0'" --plain

# CSV export
jira issue list --csv > issues.csv
```

### `jira issue view`

View a single issue.

```bash
jira issue view ISSUE-KEY [--plain] [--raw] [--comments N]
```

| Flag | Description |
|------|-------------|
| `--plain` | Plain text output |
| `--raw` | JSON output |
| `--comments` | Number of comments to show |

### `jira issue create`

Create a new issue.

| Flag | Short | Description |
|------|-------|-------------|
| `--type` | `-t` | Issue type (Bug, Story, Task, etc.) |
| `--summary` | `-s` | Issue summary/title |
| `--body` | `-b` | Issue description |
| `--priority` | `-y` | Priority (Highest, High, Medium, Low, Lowest) |
| `--label` | `-l` | Labels (repeatable) |
| `--component` | `-C` | Components (repeatable) |
| `--parent` | `-P` | Parent issue/epic key |
| `--assignee` | `-a` | Assignee username |
| `--fix-version` | | Fix version |
| `--custom` | | Custom fields (key=value) |
| `--template` | | Path to body template file |
| `--no-input` | | Non-interactive mode |

**Examples:**
```bash
# Create a bug
jira issue create -t Bug -s "Login button broken" -b "Steps to reproduce..." -y High --no-input

# Create a story under an epic
jira issue create -t Story -s "Add user profile page" -P PROJ-50 --no-input

# Create with custom fields
jira issue create -t Task -s "Setup CI" --custom "team=Platform,story-points=5" --no-input
```

### `jira issue edit`

Edit an existing issue.

```bash
jira issue edit ISSUE-KEY [flags]
```

Supports same flags as create: `-s`, `-y`, `-l`, `-C`, `-b`, `--fix-version`, `--no-input`

### `jira issue move`

Transition an issue to a new status.

```bash
jira issue move ISSUE-KEY STATUS [flags]
```

| Flag | Short | Description |
|------|-------|-------------|
| `--comment` | | Add transition comment |
| `--resolution` | `-R` | Set resolution (for Done transitions) |
| `--assignee` | `-a` | Change assignee during transition |

**Examples:**
```bash
jira issue move PROJ-123 "In Progress"
jira issue move PROJ-123 "Done" -R "Done" --comment "Completed in PR #456"
```

### `jira issue assign`

```bash
jira issue assign ISSUE-KEY USERNAME
jira issue assign ISSUE-KEY   # Assign to self (interactive)
```

### `jira issue link`

```bash
# Link two issues
jira issue link ISSUE-1 ISSUE-2 "Blocks"

# Add remote link (URL)
jira issue link remote ISSUE-KEY "https://example.com" "Link text"
```

Common link types: `Blocks`, `is blocked by`, `Cloners`, `Duplicate`, `Relates`

### `jira issue unlink`

```bash
jira issue unlink ISSUE-1 ISSUE-2
```

### `jira issue clone`

```bash
jira issue clone ISSUE-KEY [-s "New summary"] [-y Priority] [-a assignee]
```

| Flag | Short | Description |
|------|-------|-------------|
| `--summary` | `-s` | Override summary |
| `--priority` | `-y` | Override priority |
| `--assignee` | `-a` | Override assignee |
| `--replace` | `-H` | Find and replace in description (FIND:REPLACE) |

### `jira issue delete`

```bash
jira issue delete ISSUE-KEY [--cascade]
```

`--cascade` deletes subtasks too.

### `jira issue comment add`

```bash
jira issue comment add ISSUE-KEY "Comment body" [--no-input]
```

| Flag | Description |
|------|-------------|
| `--internal` | Internal/service desk comment |
| `--template` | Path to template file |
| `--no-input` | Non-interactive mode |

### `jira issue worklog add`

```bash
jira issue worklog add ISSUE-KEY "2h 30m" [--comment "Work notes"] [--no-input]
```

---

## Epic Commands

### `jira epic list`

```bash
jira epic list [--plain] [--raw] [--table]
```

Supports same filters as issue list: `-s`, `-y`, `-a`, `-l`, `--jql`, etc.

### `jira epic create`

```bash
jira epic create -n "Epic Name" -s "Epic summary" [-y Priority] [-l label] [-b body] [--no-input]
```

### `jira epic add`

Add issues to an epic.

```bash
jira epic add EPIC-KEY ISSUE-1 ISSUE-2 ISSUE-3
```

### `jira epic remove`

Remove issues from their epic.

```bash
jira epic remove ISSUE-1 ISSUE-2
```

---

## Sprint Commands

### `jira sprint list`

```bash
jira sprint list [--table] [--plain] [--raw]
```

| Flag | Description |
|------|-------------|
| `--current` | Show current sprint only |
| `--prev` | Show previous sprint |
| `--next` | Show next sprint |
| `--state` | Filter by state (active, closed, future) |

### `jira sprint add`

Add issues to a sprint.

```bash
jira sprint add SPRINT_ID ISSUE-1 ISSUE-2
```

---

## Other Commands

### `jira project list`

```bash
jira project list [--plain]
```

### `jira board list`

```bash
jira board list [--plain]
```

### `jira release list`

```bash
jira release list [--project PROJECT_KEY] [--plain]
```

### `jira open`

Open an issue or project in the browser.

```bash
jira open ISSUE-KEY
jira open PROJECT-KEY
```

### `jira me`

Show the current authenticated user.

```bash
jira me [--plain]
```

---

## Output Format Flags

| Flag | Description | Use Case |
|------|-------------|----------|
| `--plain` | Simple text layout | Scripting, parsing |
| `--raw` | Full JSON response | Programmatic processing |
| `--csv` | CSV format | Spreadsheet export |
| `--table` | Formatted table | Terminal display |
| (none) | Interactive TUI | Manual browsing |

## JQL Quick Reference

JQL (JIRA Query Language) is used with `--jql` / `-q` flag.

```
# Basic field comparison
project = "PROJ"
status = "In Progress"
assignee = currentUser()
priority in (High, Highest)

# Text search
summary ~ "login"
text ~ "error message"

# Date functions
created >= -7d
updated >= startOfWeek()
duedate < endOfMonth()

# Combining conditions
project = PROJ AND status != Done AND assignee = currentUser()
(priority = High OR priority = Highest) AND created >= -30d

# Ordering
ORDER BY priority DESC, created ASC
```
