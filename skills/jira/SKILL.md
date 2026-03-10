---
name: jira
description: "Use this skill when the user asks to interact with JIRA — including listing, viewing, creating, editing, moving, assigning, commenting on, or searching issues, as well as managing epics, sprints, and boards. Also use when the user mentions JIRA issue keys (e.g., PROJ-123), asks about project status, wants to perform any JIRA-related workflow, or needs to get/set custom fields (e.g., Reviewers, PR reviewer) via REST API."
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

## Custom Field Operations (REST API)

The `jira-cli` does not support custom fields. For custom field operations, use the JIRA REST API directly via `curl` with the same environment variables.

### 1. List Custom Fields

Find custom field IDs by name:
```bash
curl -s -u "$JIRA_USER_EMAIL:$JIRA_API_TOKEN" \
  "$JIRA_SERVER_URL/rest/api/3/field" \
  | python3 -c "
import sys, json
fields = json.load(sys.stdin)
for f in fields:
    if '<search_term>' in f.get('name','').lower():
        print(f'{f[\"id\"]}: {f[\"name\"]} (custom={f.get(\"custom\", False)})')
"
```

### 2. Get Custom Field Value

```bash
curl -s -u "$JIRA_USER_EMAIL:$JIRA_API_TOKEN" \
  "$JIRA_SERVER_URL/rest/api/3/issue/PROJ-123" \
  | python3 -c "
import sys, json
d = json.load(sys.stdin)
print(d['fields'].get('<customfield_id>', 'Not set'))
"
```

### 3. Search Users (for user-type fields)

Find a user's `accountId` by name or email:
```bash
curl -s -u "$JIRA_USER_EMAIL:$JIRA_API_TOKEN" \
  "$JIRA_SERVER_URL/rest/api/3/user/search?query=<name_or_email>" \
  | python3 -c "
import sys, json
for u in json.load(sys.stdin):
    print(f'accountId: {u[\"accountId\"]}, name: {u[\"displayName\"]}, email: {u.get(\"emailAddress\",\"N/A\")}')
"
```

### 4. Set Custom Field Value

**User-type field** (single user):
```bash
curl -s -u "$JIRA_USER_EMAIL:$JIRA_API_TOKEN" \
  -X PUT -H "Content-Type: application/json" \
  "$JIRA_SERVER_URL/rest/api/3/issue/PROJ-123" \
  -d '{"fields":{"<customfield_id>":{"accountId":"<account_id>"}}}'
```

**User-type field** (multi-user, e.g., Reviewers):
```bash
curl -s -u "$JIRA_USER_EMAIL:$JIRA_API_TOKEN" \
  -X PUT -H "Content-Type: application/json" \
  "$JIRA_SERVER_URL/rest/api/3/issue/PROJ-123" \
  -d '{"fields":{"<customfield_id>":[{"accountId":"<account_id>"}]}}'
```

**Text field**:
```bash
curl -s -u "$JIRA_USER_EMAIL:$JIRA_API_TOKEN" \
  -X PUT -H "Content-Type: application/json" \
  "$JIRA_SERVER_URL/rest/api/3/issue/PROJ-123" \
  -d '{"fields":{"<customfield_id>":"value"}}'
```

**Select field**:
```bash
curl -s -u "$JIRA_USER_EMAIL:$JIRA_API_TOKEN" \
  -X PUT -H "Content-Type: application/json" \
  "$JIRA_SERVER_URL/rest/api/3/issue/PROJ-123" \
  -d '{"fields":{"<customfield_id>":{"value":"Option Name"}}}'
```

### 5. Inspect Field Schema

Check a custom field's expected data type before setting it:
```bash
curl -s -u "$JIRA_USER_EMAIL:$JIRA_API_TOKEN" \
  "$JIRA_SERVER_URL/rest/api/3/field" \
  | python3 -c "
import sys, json
for f in json.load(sys.stdin):
    if f['id'] == '<customfield_id>':
        print(json.dumps(f.get('schema', {}), indent=2))
"
```

### Custom Field Workflow Example

To set a "Reviewers" field on an issue:
1. Find the field ID: search for `review` in custom fields
2. Search for the user to get their `accountId`
3. Set the field using the multi-user payload format

## Detailed Command Reference

For the full list of commands, flags, and advanced usage patterns, see [reference.md](reference.md).
