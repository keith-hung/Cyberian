---
name: jira
description: "Use this skill when the user asks to interact with JIRA — including listing, viewing, creating, editing, moving, assigning, commenting on, or searching issues, as well as managing epics, sprints, and boards. Also use when the user mentions JIRA issue keys (e.g., PROJ-123), asks about project status, or wants to perform any JIRA-related workflow."
user-invokable: true
argument-hint: "[action or issue key, e.g. 'list', 'PROJ-123', 'create bug']"
---

# JIRA Skill — jira-cli Integration

You have access to `jira` CLI (https://github.com/ankitpokhrel/jira-cli) on the user's system.
Use it to interact with JIRA issues, epics, sprints, and boards via shell commands.

## Prerequisites

- `jira` CLI must be installed and configured (`jira init` completed)
- User must have valid JIRA authentication (API token or PAT)

Before executing any command, verify the CLI is available:
```
jira me
```
If this fails, inform the user that jira-cli is not configured and guide them to run `jira init`.

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
jira issue list --plain

# Filter by status
jira issue list -s "In Progress" --plain

# Filter by assignee (current user)
jira issue list -a$(jira me --plain) --plain

# Custom JQL query
jira issue list --jql "project = PROJ AND status = 'To Do' ORDER BY priority DESC" --plain
```

### 2. View Issue Details
```bash
# View issue with comments
jira issue view PROJ-123 --plain --comments 5
```

### 3. Create Issue
```bash
# Create with all fields (non-interactive)
jira issue create \
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
jira issue edit PROJ-123 -s "Updated summary" --no-input
```

### 5. Move/Transition Issue
```bash
# Move issue to a new status
jira issue move PROJ-123 "In Progress"
jira issue move PROJ-123 "Done" -R "Done"
```

### 6. Assign Issue
```bash
jira issue assign PROJ-123 username
```

### 7. Add Comment
```bash
jira issue comment add PROJ-123 "This is a comment" --no-input
```

### 8. Search with JQL
```bash
jira issue list --jql "text ~ 'search term' AND project = PROJ" --plain
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
- If jira-cli is not installed, provide installation instructions from the reference

## Detailed Command Reference

For the full list of commands, flags, and advanced usage patterns, see [reference.md](reference.md).
