---
name: change-password
description: "Change your on-prem AD password. Trigger when user asks about: 改密碼, change password, AD password, 密碼過期, domain password. Two paths, auto-selected: domain-joined Windows (local, instant) or the off-network self-service portal (OTP via app/SMS)."
user-invokable: true
argument-hint: "[scenario, e.g. 'domain' or 'off-network']"
---

# Change Password Skill

Change the user's on-prem Active Directory password. The directory is on-prem
with one-way sync to Azure (Password Hash Sync, no writeback), so the password
MUST be changed against on-prem AD — never a cloud page.

## Trigger

Activate when the user wants to change (not "forgot"/reset) their AD password:
改密碼 / change password / AD 密碼 / 密碼過期 / domain password.

## Routing (automatic)

Choose the path automatically from a detection probe; do not ask the user unless
they explicitly request a specific path.

**Step 1 — detect** (run via `powershell.exe`; on WSL wrap the script path with `wslpath -w`):

```bash
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "$(wslpath -w "${CLAUDE_PLUGIN_ROOT}/skills/change-password/local-change.ps1")" -Detect
```

Prints JSON, e.g. `{"domainJoined":true,"userIsDomain":true,"dcReachable":true,"domain":"...","user":"...","adViable":true}`.
If `powershell.exe` is missing or errors (e.g. macOS), treat it as `adViable=false`.
(`-ExecutionPolicy Bypass` lets the local script run even under a Restricted machine
policy; it affects only this invocation, not the machine.)

**Step 2 — route on `adViable`:**

| Result | Path |
|--------|------|
| `adViable == true` (domain-joined + domain account + DC reachable) | **Path A — local AD** |
| otherwise (non-domain / DC unreachable / no PowerShell) | **Path B — chpw portal** |

For Path B, if `CHPW_BASE_URL` is not set, stop and tell the user neither path is
available and what to configure. The user may override the automatic choice by
asking for a specific path.

## Presenting commands (make them copy-paste-ready)

Any command you give the user to run MUST work when pasted verbatim — no editing:

- **Resolve the path.** Expand `${CLAUDE_PLUGIN_ROOT}` to its real absolute path
  before showing the command (run `echo "${CLAUDE_PLUGIN_ROOT}"` if unsure). In a
  source checkout the target is the built binary, e.g. `<repo>/chpw-cli/chpw`. Never
  leave `${CLAUDE_PLUGIN_ROOT}`, `$(...)`, or `~` unexpanded in what you hand over.
- **Use a real invocable name.** The launcher's absolute path
  (`.../scripts/chpw-launcher.sh`) or the built binary's absolute path — never a bare
  `chpw` (it is not on `PATH`).
- **No leftover placeholders.** Fill `--user` with the actual account (from
  `CHPW_USERNAME`, or ask the user once — the username is not a secret), OR omit
  `--user` for `-i` so the tool prompts. Do not leave `<USERNAME>` for the user to
  edit. (`<OTP>` / the password are entered at prompts or a later step — not edited
  into the command.)
- **Ensure `CHPW_BASE_URL` is present for the shell that will run it.** If it is
  already exported in the user's shell, nothing to do; otherwise prefix it inline:
  `CHPW_BASE_URL="<portal-url>" .../chpw-launcher.sh ...`. Put the URL in the command
  you hand the user — never in a tracked file.

The snippets below use `${CLAUDE_PLUGIN_ROOT}` / `<USERNAME>` as placeholders for
readability; always resolve them before presenting.

## Path A — domain-joined (local, no OTP)

Run the PowerShell script; pipe old then new password (two lines) to stdin.
Domain and user are auto-detected from the logged-in session — no flags needed.
Never put passwords on the command line.

```bash
printf '%s\n%s\n' "$OLD" "$NEW" | powershell.exe -NoProfile -ExecutionPolicy Bypass -File \
  "$(wslpath -w "${CLAUDE_PLUGIN_ROOT}/skills/change-password/local-change.ps1")"
```

Optionally override the target with `-Domain <name> -User <name>` (e.g. to change a
different domain account). Output: `{"success":true}` or `{"success":false,"error":"..."}`.
Common failures: wrong old password, password-policy violation, or DC unreachable.

## Path B — off-network self-service portal (OTP)

The portal sends an OTP (via the i-daka app / email, or SMS) after you submit your
current password. Two ways to run it, by who is at the keyboard. **Default to B1** so
the password stays in the user's own terminal; use B2 only when the user explicitly
opts into the agent handling the password.

### B1 — the user runs it (interactive, recommended)

Give the user this command to run in THEIR OWN terminal (passwords are typed at hidden
prompts and never enter this conversation). Present it fully resolved per "Presenting
commands" above — absolute path, `--user` filled (or omitted so it prompts), no
placeholders, `CHPW_BASE_URL` ensured — so they can paste and run it as-is:

```bash
${CLAUDE_PLUGIN_ROOT}/scripts/chpw-launcher.sh -i --user <USERNAME> --method APP
```

It walks through: `Current password:` (hidden) → OTP sent → `Enter OTP:` →
`New password:` / `Confirm new password:` (hidden) → `{"success":true,"message":"password changed"}`.
`--method` selects delivery: `APP` (i-daka/Email, default) or `SMS`. `-i` requires a
terminal; the user can omit `--user` and it will prompt. By default `-i` asks for the
new password twice (confirmation); pass `--no-confirm` to enter it once.

### B2 — agent-driven (two-step; only after the user opts in)

Step 1 — sends the OTP and prints the next command:

```bash
printf '%s\n' "$OLD_PASSWORD" | \
  ${CLAUDE_PLUGIN_ROOT}/scripts/chpw-launcher.sh --user "<USERNAME>" --method APP --pass-stdin
```

Then ask the user for the OTP they received. Step 2 — within ~120s, same directory:

```bash
printf '%s\n' "$NEW_PASSWORD" | \
  ${CLAUDE_PLUGIN_ROOT}/scripts/chpw-launcher.sh --continue --pass-stdin --otp "<OTP>"
```

Output: `{"success":true,"message":"password changed"}` or a structured error
(`validation:` for wrong OTP / weak password / expired session; `authentication:` for a
wrong current password at step 1). Windows PowerShell uses `chpw-launcher.ps1` instead of `.sh`.

## Prerequisites (Path B)

- `CHPW_BASE_URL` — portal base URL (required)
- `CHPW_USERNAME` — default username (optional)
- `CHPW_INSECURE` — skip TLS verification, dev only (optional)

**Option A — Centralized config (recommended):** copy `.claude/settings.json.example`
to `.claude/settings.local.json` and fill in the `CHPW_*` values.
**Option B — Shell profile:** export the variables in `~/.zshrc` / `~/.bashrc`.

## After changing (both paths)

Remind the user to update every place that caches the old password, or a stale
cached password will keep retrying and can lock the AD account:

- Outlook / mail clients
- VPN saved credentials
- Mapped network drives / shares
- Enterprise Wi-Fi (802.1x)
- Phone mail / Teams

The on-prem change syncs to Azure via Password Hash Sync in about 2 minutes; no
cloud action is needed.

## Security notes

- Passwords are passed only via stdin — never as flags, never echoed, never written to files.
- The session file (`.chpw-session.json`) holds cookies + a form token only, never a password.
- The portal URL and any AD details live in env/settings, never in tracked files.
- The user enters passwords in their own terminal (Path A / Path B1); the agent only handles the password when the user explicitly chooses the two-step agent path (B2).
