---
name: change-password
description: "Change your on-prem AD password. Trigger when user asks about: 改密碼, change password, AD password, 密碼過期, domain password. Two paths, auto-selected: domain-joined Windows (local, instant) or the off-network self-service portal (SMS OTP)."
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

## Path B — off-network self-service portal (SMS OTP)

Two steps with a human OTP in the middle. Requires `CHPW_BASE_URL` (and usually
`CHPW_USERNAME`) configured — see Prerequisites.

1. **login** — verifies the current password; the server texts a 6-digit OTP
   (valid 120s) to the registered phone.

   ```bash
   printf '%s\n' "$OLD_PASSWORD" | \
     ${CLAUDE_PLUGIN_ROOT}/scripts/chpw-launcher.sh login --pass-stdin --user "<USERNAME>"
   ```

2. Ask the user for the OTP they just received.

3. **submit** — sends the new password + OTP. Must complete within ~2 minutes.

   ```bash
   printf '%s\n' "$NEW_PASSWORD" | \
     ${CLAUDE_PLUGIN_ROOT}/scripts/chpw-launcher.sh submit --pass-stdin --otp "<OTP>"
   ```

Output: `{"success":true,"message":"password changed"}` or a structured error
(`validation:` for wrong OTP / weak password / expired session; `authentication:`
for a wrong current password at the login step).

Windows PowerShell uses `chpw-launcher.ps1` instead of `.sh`.

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
