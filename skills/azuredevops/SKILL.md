---
name: azuredevops
description: "Manage Azure DevOps projects, repositories, and pull requests. Trigger when user asks about: pull request, PR, code review, merge, Azure DevOps, TFS, DevOps repo, DevOps project."
user-invokable: true
argument-hint: "[action, e.g. 'list PRs', 'create PR', 'approve PR 123']"
---

# Azure DevOps Skill

Manage Azure DevOps Server projects, repositories, and pull requests via the `azuredevops` CLI.

## Trigger

Activate when user asks to:
- View Azure DevOps projects or repositories
- List, view, create, or update pull requests (PR)
- Approve or reject a PR (code review)
- Add comments or reviewers to a PR
- Merge a PR (via status update to "completed")

## Prerequisites

The following environment variables must be set:

| Variable | Required | Description |
|----------|----------|-------------|
| `AZDO_BASE_URL` | Yes | Server URL, e.g. `https://tfs.company.com:8080` |
| `AZDO_COLLECTION` | Yes | Collection name, e.g. `DefaultCollection` |
| `AZDO_DOMAIN` | No | AD domain name, e.g. `MYDOMAIN` — auto-prepended to username |
| `AZDO_USERNAME` | Yes | Username (plain, e.g. `jdoe`; or with domain, e.g. `DOMAIN\user`) |
| `AZDO_PASSWORD` | Yes | Password |
| `AZDO_PROJECT` | No | Default project name |
| `AZDO_REPO` | No | Default repository name |
| `AZDO_API_VERSION` | No | API version (default: `5.0`) |

If `AZDO_DOMAIN` is set and `AZDO_USERNAME` does not contain `\`, the CLI auto-combines them as `DOMAIN\username` for Basic Auth. You can also set `AZDO_USERNAME` directly to `DOMAIN\user` and omit `AZDO_DOMAIN`.

**Option A — Centralized config (recommended):**
Copy `.claude/settings.json.example` to `.claude/settings.local.json` and fill in the `AZDO_*` values.

**Option B — Shell profile:**
Export the variables in `~/.zshrc` or `~/.bashrc`.

The CLI binary is in `${CLAUDE_PLUGIN_ROOT}/scripts/`. Invoke via the platform-appropriate launcher:

| Platform | Command |
|----------|---------|
| Linux / macOS | `${CLAUDE_PLUGIN_ROOT}/scripts/azuredevops-launcher.sh <command> [flags]` |
| Windows (PowerShell) | `& "${env:CLAUDE_PLUGIN_ROOT}/scripts/azuredevops-launcher.ps1" <command> [flags]` |

## Commands

### List projects
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/azuredevops-launcher.sh projects
```
Returns all projects in the collection.

### List repositories
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/azuredevops-launcher.sh repos --project <ProjectName>
```
Returns all Git repositories in the specified project.

### List pull requests
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/azuredevops-launcher.sh prs --project <ProjectName> --repo <RepoName> [--status active|completed|abandoned|all]
```
Default status is `active`. The `--repo` flag accepts the repository name (not GUID).

### View pull request details
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/azuredevops-launcher.sh pr --project <ProjectName> --repo <RepoName> --id <PR_ID>
```
Returns full PR details including title, description, status, branches, and reviewers with vote status.

### Create pull request
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/azuredevops-launcher.sh pr-create --project <ProjectName> --repo <RepoName> --source <branch> --target <branch> --title "PR Title" [--description "Description"]
```
Branch names can omit `refs/heads/` prefix — it will be added automatically.

### Update pull request
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/azuredevops-launcher.sh pr-update --project <ProjectName> --repo <RepoName> --id <PR_ID> [--title "New Title"] [--description "New Desc"] [--status active|completed|abandoned]
```
At least one of `--title`, `--description`, or `--status` is required. Setting `--status completed` merges the PR.

### Approve pull request
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/azuredevops-launcher.sh pr-approve --project <ProjectName> --repo <RepoName> --id <PR_ID>
```
Casts an "Approved" vote (10) as the authenticated user.

### Reject pull request
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/azuredevops-launcher.sh pr-reject --project <ProjectName> --repo <RepoName> --id <PR_ID>
```
Casts a "Rejected" vote (-10) as the authenticated user.

### Add comment to pull request
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/azuredevops-launcher.sh pr-comment --project <ProjectName> --repo <RepoName> --id <PR_ID> --comment "Comment text"
```

### List PR reviewers
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/azuredevops-launcher.sh pr-reviewers --project <ProjectName> --repo <RepoName> --id <PR_ID>
```
Returns reviewers with their vote status: approved (10), approved with suggestions (5), no vote (0), waiting for author (-5), rejected (-10).

### Add reviewer to pull request
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/azuredevops-launcher.sh pr-add-reviewer --project <ProjectName> --repo <RepoName> --id <PR_ID> --reviewer <GUID_or_domain\username>
```
The `--reviewer` flag accepts a GUID or `DOMAIN\username` format.

### Version
```bash
${CLAUDE_PLUGIN_ROOT}/scripts/azuredevops-launcher.sh version
```

## Important Rules

- **Default project/repo**: If `AZDO_PROJECT` and `AZDO_REPO` are set, `--project` and `--repo` flags can be omitted
- **Repo names**: The CLI resolves repo names to GUIDs automatically — always use the human-readable name
- **Branch names**: Use short branch names (e.g., `main`, `feature/xyz`) — `refs/heads/` is prepended automatically
- **Vote semantics**: 10=approved, 5=approved with suggestions, 0=no vote, -5=waiting for author, -10=rejected
- **Merge via update**: To merge a PR, use `pr-update --status completed`
- **Abandon via update**: To close without merging, use `pr-update --status abandoned`
- **Authentication**: Uses IIS Basic Auth — ensure the server has Basic Auth enabled in IIS

## Workflow

### Review a PR
1. **List** active PRs: `prs --project X --repo Y`
2. **View** details: `pr --project X --repo Y --id 123`
3. **Comment** if needed: `pr-comment --project X --repo Y --id 123 --comment "Looks good"`
4. **Approve**: `pr-approve --project X --repo Y --id 123`

### Create and merge a PR
1. **Create**: `pr-create --project X --repo Y --source feature/abc --target main --title "Add feature"`
2. **Add reviewers**: `pr-add-reviewer --project X --repo Y --id 456 --reviewer DOMAIN\reviewer`
3. **Merge**: `pr-update --project X --repo Y --id 456 --status completed`

## Error Handling

| Exit Code | Meaning | Action |
|-----------|---------|--------|
| 0 | Success | — |
| 1 | General error | Check error message |
| 2 | Config/auth error | Verify environment variables or credentials |
| 3 | Validation error | Check input flags (missing project, repo, or ID) |
| 4 | Network error | Verify server URL is reachable |

Errors are JSON on stderr: `{"success":false,"error":"message"}`.
