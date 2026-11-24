---
name: clouddrive-fetch
description: "Download cloud-drive online-only (placeholder) files in place so they can be read, without pinning them. Use when reading a file under OneDrive, iCloud Drive, Google Drive, Dropbox or an rclone mount fails or looks empty: \"Resource deadlock avoided\", \"cannot read\", 0-byte or dataless files, Files On-Demand placeholders, or when the user asks to download, fetch or materialize online-only files. Works on macOS, Windows, WSL and Linux."
---

# Cloud Drive Fetch

## Overview

Sync clients keep "online-only" files as placeholders and download them on first read. Some processes cannot trigger that download (on macOS, Claude Code reads fail with `Resource deadlock avoided`), and agents cannot tell a placeholder from a real file. `clouddrive_fetch.py` finds the placeholders and reads them so the provider downloads them in place.

| Platform | Online-only when | Download |
|----------|------------------|----------|
| macOS (OneDrive, iCloud Drive, Google Drive, Dropbox) | `st_flags` has `SF_DATALESS` (`ls -lO` shows `dataless`) | enables the materialize I/O policy for itself (`setiopolicy_np`), then reads |
| Windows (OneDrive Files On-Demand, other Cloud Files providers) | attributes have `RECALL_ON_DATA_ACCESS`, `RECALL_ON_OPEN` or `OFFLINE` | reads |
| WSL (`/mnt/c/...`) and Linux (rclone mount) | heuristic: `st_size > 0` and `st_blocks == 0` (small local NTFS files and sparse files match too) | reads; a complete read counts as fetched |

Files are not set to "Always keep on this device"; the provider may free them again later. Do not pin them or change sync settings on the user's behalf.

## Execution Steps

`clouddrive_fetch.py` is stdlib-only; run it with `uv run` or `python3`.

```bash
F=${CLAUDE_PLUGIN_ROOT}/skills/clouddrive-fetch/clouddrive_fetch.py

# Download in place (folders are walked recursively)
uv run $F "<path>" ["<path>" ...]

# Only list online-only files
uv run $F --list "<folder>"

# Download and copy files or folders into another place (refuses to overwrite without --force)
uv run $F --copy [--force] "<src>" ["<src>" ...] "<dst>"

# Machine-readable summary
uv run $F --json "<path>"
```

Quote paths: cloud-drive folders usually contain spaces (`OneDrive - <Org>`).

### Options

| Option | Meaning |
|--------|---------|
| `--list` | List online-only files, download nothing |
| `--copy` | Copy SRC... (files or folders) to the last argument; DST must be an existing folder for several SRC or a folder SRC, which is copied as `DST/<folder name>/` |
| `--force` | With `--copy`, overwrite existing files |
| `-j`, `--jobs N` | Parallel downloads, default 4 |
| `--json` | Print `{fetched, online, copied, local, failed, missing, success}` instead of text lines |

Exit codes: 0 success, 1 some files failed or are missing (including files that vanished or could not be read during the scan), 3 bad arguments. A symlink given as an argument is resolved to its target.

## Guidance

- Fetch only what the task needs (a file or a small folder); a whole drive can be many GB. Run `--list` first on large folders and ask the user before downloading a lot.
- Run it before reading cloud-drive files that other skills depend on, for example the `outlook-calendar` export folder when its reader reports `cannot read` or `not found`.
- Downloading does not refresh a stale local copy; if the content itself is outdated, the sync client is behind (paused, asleep, offline) — tell the user.

## Troubleshooting

| Issue | Cause / Solution |
|-------|------------------|
| `failed ...: still online-only after read` | The provider did not download (offline, signed out, sync paused) → ask the user to check the sync client |
| `failed ...: Resource deadlock avoided` (macOS) | The I/O policy could not be enabled (warning on stderr) → run the script from a normal Terminal |
| WSL / Linux file listed as online-only but already local | Small NTFS files and sparse files report no allocated blocks; reading them is harmless and they are counted as fetched |
