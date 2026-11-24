#!/usr/bin/env python3
# /// script
# requires-python = ">=3.9"
# dependencies = []
# ///
"""
clouddrive-fetch -- download cloud-drive "online-only" files in place, without pinning them.

Works with sync clients that keep placeholders on disk and download on first read:
OneDrive, iCloud Drive, Google Drive, Dropbox (macOS File Provider / Windows Cloud Files),
OneDrive under WSL (/mnt/c/...), and rclone mounts on Linux.

Usage:
    uv run clouddrive_fetch.py PATH...              download in place; folders are walked recursively
    uv run clouddrive_fetch.py --list PATH...       only list online-only files, download nothing
    uv run clouddrive_fetch.py --copy SRC... DST    download SRC and copy it to DST (a file, or a folder for several SRC)
    add --json for a machine-readable summary on stdout

How online-only files are detected and downloaded:
    macOS    st_flags has SF_DATALESS. Some processes (e.g. Claude Code) have the
             materialize-dataless-files I/O policy off, so reads fail with "Resource deadlock
             avoided"; the tool turns it on for itself (setiopolicy_np) before reading.
    Windows  file attributes have RECALL_ON_DATA_ACCESS / RECALL_ON_OPEN / OFFLINE.
    Linux    st_blocks == 0 while st_size > 0 (WSL drvfs placeholders, rclone VFS without cache).
Reading the whole file makes the provider download it. Files are NOT set to "Always keep on
this device"; the provider may free them again later.

Exit codes: 0=success, 1=some files failed or are missing, 3=validation (bad args)
"""

import argparse
import concurrent.futures
import ctypes
import json
import os
import shutil
import sys

CHUNK = 1 << 20

# macOS
SF_DATALESS = 0x40000000
IOPOL_TYPE_VFS_MATERIALIZE_DATALESS_FILES = 3
IOPOL_SCOPE_PROCESS = 0
IOPOL_MATERIALIZE_DATALESS_FILES_ON = 2

# Windows
FILE_ATTRIBUTE_OFFLINE = 0x00001000
FILE_ATTRIBUTE_RECALL_ON_OPEN = 0x00040000
FILE_ATTRIBUTE_RECALL_ON_DATA_ACCESS = 0x00400000
WIN_ONLINE_ONLY = FILE_ATTRIBUTE_OFFLINE | FILE_ATTRIBUTE_RECALL_ON_OPEN | FILE_ATTRIBUTE_RECALL_ON_DATA_ACCESS


def enable_materialize() -> None:
    """macOS only: allow this process to trigger downloads of dataless files."""
    if sys.platform != "darwin":
        return
    libc = ctypes.CDLL(None, use_errno=True)
    rc = libc.setiopolicy_np(IOPOL_TYPE_VFS_MATERIALIZE_DATALESS_FILES, IOPOL_SCOPE_PROCESS,
                             IOPOL_MATERIALIZE_DATALESS_FILES_ON)
    if rc != 0:
        print(f"warning: setiopolicy_np failed: {os.strerror(ctypes.get_errno())}", file=sys.stderr)


def is_online_only(path: str) -> bool:
    st = os.lstat(path)
    if sys.platform == "darwin":
        return bool(st.st_flags & SF_DATALESS)
    if sys.platform == "win32":
        return bool(st.st_file_attributes & WIN_ONLINE_ONLY)
    return st.st_size > 0 and st.st_blocks == 0


class Run:
    def __init__(self, as_json: bool):
        self.as_json = as_json
        self.result = {"fetched": [], "online": [], "copied": [], "local": 0, "failed": [], "missing": []}

    def say(self, line: str, err: bool = False) -> None:
        if not self.as_json:
            print(line, file=sys.stderr if err else sys.stdout)

    def fail(self, path: str, error: str) -> None:
        self.result["failed"].append({"path": path, "error": error})
        self.say(f"failed   {path}: {error}", err=True)

    def iter_files(self, paths):
        for path in paths:
            if os.path.islink(path):
                continue
            if os.path.isdir(path):
                for root, _, files in os.walk(path):
                    for name in files:
                        full = os.path.join(root, name)
                        if not os.path.islink(full):
                            yield full
            elif os.path.exists(path):
                yield path
            else:
                self.result["missing"].append(path)
                self.say(f"missing  {path}", err=True)

    def ok(self) -> bool:
        return not self.result["failed"] and not self.result["missing"]

    def finish(self, summary: str) -> int:
        if self.as_json:
            print(json.dumps(dict(self.result, success=self.ok()), ensure_ascii=False, indent=2))
        else:
            print(summary, file=sys.stderr if summary.endswith("online-only") else sys.stdout)
        return 0 if self.ok() else 1


def materialize(path: str):
    """Read the whole file so the provider downloads it. Returns an error string or None."""
    try:
        with open(path, "rb") as fh:
            while fh.read(CHUNK):
                pass
        return "still online-only after read" if is_online_only(path) else None
    except OSError as exc:
        return exc.strerror or str(exc)


def fetch(run: Run, paths: list, jobs: int) -> int:
    online = []
    for path in run.iter_files(paths):
        if is_online_only(path):
            online.append(path)
        else:
            run.result["local"] += 1
    with concurrent.futures.ThreadPoolExecutor(max_workers=jobs) as pool:
        for path, error in zip(online, pool.map(materialize, online)):
            if error:
                run.fail(path, error)
            else:
                run.result["fetched"].append(path)
                run.say(f"fetched  {path}")
    return run.finish(f"done: {len(run.result['fetched'])} fetched, {run.result['local']} already local, "
                      f"{len(run.result['failed'])} failed")


def list_online(run: Run, paths: list) -> int:
    for path in run.iter_files(paths):
        if is_online_only(path):
            run.result["online"].append(path)
            run.say(path)
        else:
            run.result["local"] += 1
    return run.finish(f"{len(run.result['online'])} online-only")


def copy(run: Run, sources: list, dst: str) -> int:
    for src in sources:
        target = os.path.join(dst, os.path.basename(src)) if os.path.isdir(dst) else dst
        try:
            shutil.copyfile(src, target)
            run.result["copied"].append({"from": src, "to": target})
            run.say(f"copied   {src} -> {target}")
        except OSError as exc:
            run.fail(src, exc.strerror or str(exc))
    return run.finish(f"done: {len(run.result['copied'])} copied, {len(run.result['failed'])} failed")


def main() -> int:
    parser = argparse.ArgumentParser(
        prog="clouddrive-fetch",
        description="Download cloud-drive online-only files in place without pinning them.")
    mode = parser.add_mutually_exclusive_group()
    mode.add_argument("--list", action="store_true", help="only list online-only files")
    mode.add_argument("--copy", action="store_true", help="copy SRC... to the last argument")
    parser.add_argument("-j", "--jobs", type=int, default=4, help="parallel downloads (default 4)")
    parser.add_argument("--json", action="store_true", help="print a JSON summary instead of text lines")
    parser.add_argument("paths", nargs="+", metavar="PATH")
    args = parser.parse_args()

    if args.copy:
        if len(args.paths) < 2:
            parser.error("--copy needs SRC... DST")
        if len(args.paths) > 2 and not os.path.isdir(args.paths[-1]):
            parser.exit(3, f"clouddrive-fetch: {args.paths[-1]} must be an existing folder "
                           "when copying several files\n")

    enable_materialize()
    run = Run(args.json)
    if args.copy:
        return copy(run, args.paths[:-1], args.paths[-1])
    if args.list:
        return list_online(run, args.paths)
    return fetch(run, args.paths, max(1, args.jobs))


if __name__ == "__main__":
    sys.exit(main())
