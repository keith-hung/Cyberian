---
name: jira
description: "Use this skill when the user asks to interact with JIRA — including listing, viewing, creating, editing, moving, assigning, commenting on, or searching issues, as well as managing epics, sprints, and boards. Also use when the user mentions JIRA issue keys (e.g., PROJ-123), asks about project status, or wants to perform any JIRA-related workflow."
user-invokable: true
argument-hint: "[action or issue key, e.g. 'list', 'PROJ-123', 'create bug']"
---

# JIRA Skill — jira-cli Integration

You have access to `jira` CLI (https://github.com/ankitpokhrel/jira-cli) via the launcher script.
Use it to interact with JIRA issues, epics, sprints, and boards via shell commands.

## Prerequisites

The following environment variables must be set:

| Variable | Required | Description |
|----------|----------|-------------|
| `JIRA_API_TOKEN` | **Yes** | JIRA API token |
| `JIRA_SERVER_URL` | **Yes** | JIRA instance URL (e.g. `https://company.atlassian.net`) |
| `JIRA_USER_EMAIL` | **Yes** | Login email for authentication |
| `JIRA_PROJECT` | No | Default project key |
| `JIRA_BOARD` | No | Default board name |

**Option A — Centralized config (recommended):**
Copy `.claude/settings.json.example` to `.claude/settings.local.json` and fill in the `JIRA_*` values.

**Option B — Shell profile:**
Export the variables in `~/.zshrc` or `~/.bashrc`.

The launcher auto-initializes jira-cli config on first run when the required env vars are set.

The CLI binary is in `${CLAUDE_PLUGIN_ROOT}/scripts/`. Invoke via the launcher:
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/jira-launcher.sh <command> [flags]
```

## Core Principles

1. **Always use `--plain` flag** when you need to parse output programmatically
2. **Use `--raw` flag** when you need structured JSON data
3. **Use `--no-input` flag** for non-interactive create/edit operations
4. **Never guess issue keys** — always confirm with the user or look them up first
5. **Present results in formatted tables** when displaying lists to the user

## Handling User Requests

When the user provides `$ARGUMENTS`:

- **Issue key** (e.g., `PROJ-123`): View the issue details
- **`list`** or no arguments: List recent issues in the current project
- **`create`**: Guide through issue creation
- **`search <query>`**: Search issues using JQL or filters
- **Other text**: Interpret intent and map to appropriate jira-cli commands

## Common Workflows

### 1. List Issues
```bash
# List issues in current project
${CLAUDE_PLUGIN_ROOT}/scripts/jira-launcher.sh issue list --plain

# Filter by status
${CLAUDE_PLUGIN_ROOT}/scripts/jira-launcher.sh issue list -s "In Progress" --plain

# Filter by assignee (current user)
${CLAUDE_PLUGIN_ROOT}/scripts/jira-launcher.sh issue list -a$(${CLAUDE_PLUGIN_ROOT}/scripts/jira-launcher.sh me --plain) --plain

# Custom JQL query
${CLAUDE_PLUGIN_ROOT}/scripts/jira-launcher.sh issue list --jql "project = PROJ AND status = 'To Do' ORDER BY priority DESC" --plain
```

### 2. View Issue Details
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/jira-launcher.sh issue view PROJ-123 --plain --comments 5
```

### 3. Create Issue
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/jira-launcher.sh issue create \
  -t Bug \
  -s "Summary of the issue" \
  -b "Detailed description" \
  -y High \
  -l "backend" \
  -P PROJ-100 \
  --no-input
```

### 4. Edit Issue
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/jira-launcher.sh issue edit PROJ-123 -s "Updated summary" --no-input
```

### 5. Move/Transition Issue
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/jira-launcher.sh issue move PROJ-123 "In Progress"
${CLAUDE_PLUGIN_ROOT}/scripts/jira-launcher.sh issue move PROJ-123 "Done" -R "Done"
```

### 6. Assign Issue
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/jira-launcher.sh issue assign PROJ-123 username
```

### 7. Add Comment
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/jira-launcher.sh issue comment add PROJ-123 "This is a comment" --no-input
```

### 8. Search with JQL
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/jira-launcher.sh issue list --jql "text ~ 'search term' AND project = PROJ" --plain
```

## Output Formatting

When presenting JIRA data to the user:
- Use **tables** for issue lists (columns: Key, Type, Status, Priority, Summary, Assignee)
- Use **structured blocks** for single issue details
- Highlight **status** and **priority** clearly
- Include links where relevant: `https://<domain>.atlassian.net/browse/ISSUE-KEY`

## Error Handling

- If a command fails, check the error message and suggest fixes
- Common issues: expired token, wrong project key, invalid status name
- For permission errors, inform the user they may lack JIRA project access

## Detailed Command Reference

For the full list of commands, flags, and advanced usage patterns, see [reference.md](reference.md).
