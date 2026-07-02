#!/usr/bin/env bash
#
# bump-version.sh — bump the plugin version across every file that hardcodes it,
# in one shot, so releases never drift. Mirrors the "Version Bump Checklist" in
# CLAUDE.md. Does NOT commit, tag, or push — review the diff, then release manually.
#
# Usage:
#   ./scripts/bump-version.sh 0.3.1
#   ./scripts/bump-version.sh v0.3.1     # leading v is tolerated
#
# Touches: plugin.json, marketplace.json, the 4 CLI launchers (.sh + .ps1;
# jira-launcher tracks upstream and is intentionally skipped), README build/tag
# examples, and CHANGELOG.md (rolls [Unreleased] into a dated release section).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

NEW="${1:-}"
NEW="${NEW#v}"
if [[ ! "$NEW" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "usage: $0 X.Y.Z   (e.g. $0 0.3.1)" >&2
  exit 1
fi

DATE="$(date +%Y-%m-%d)"

ROOT="$ROOT" NEW="$NEW" DATE="$DATE" python3 - <<'PY'
import os, re, json, sys

root = os.environ["ROOT"]
new  = os.environ["NEW"]
date = os.environ["DATE"]
newv = "v" + new

# plugin.json is the source of truth for the current (OLD) version.
plugin = os.path.join(root, ".claude-plugin/plugin.json")
old = json.load(open(plugin, encoding="utf-8"))["version"]
oldv = "v" + old

if old == new:
    print(f"Already at {new}; nothing to do.")
    sys.exit(0)

changed = []

def edit(rel, subs):
    """Literal-replace each (old, new) pair in the file; record if it changed."""
    path = os.path.join(root, rel)
    with open(path, encoding="utf-8") as f:
        s = orig = f.read()
    for a, b in subs:
        s = s.replace(a, b)
    if s != orig:
        with open(path, "w", encoding="utf-8") as f:
            f.write(s)
        changed.append(rel)

# 1-2. plugin.json + marketplace.json (bare X.Y.Z, no leading v)
edit(".claude-plugin/plugin.json",      [(f'"version": "{old}"', f'"version": "{new}"')])
edit(".claude-plugin/marketplace.json", [(f'"version": "{old}"', f'"version": "{new}"')])

# 3-10. CLI launchers (jira tracks its own upstream version — skip it)
for cli in ["nouveau-timecard", "wedaka", "azuredevops", "chpw"]:
    edit(f"scripts/{cli}-launcher.sh",  [(f'VERSION="{oldv}"',     f'VERSION="{newv}"')])
    edit(f"scripts/{cli}-launcher.ps1", [(f'$Version = "{oldv}"',  f'$Version = "{newv}"')])

# 11. README build/tag examples (targeted, not a blind global replace)
edit("README.md", [
    (f"build.sh {oldv}",        f"build.sh {newv}"),
    (f"git tag {oldv}",         f"git tag {newv}"),
    (f"git push origin {oldv}", f"git push origin {newv}"),
])

# 12. CHANGELOG.md — roll [Unreleased] into a dated [X.Y.Z] section, leaving a
# fresh empty [Unreleased] on top (Keep-a-Changelog convention).
cl = os.path.join(root, "CHANGELOG.md")
with open(cl, encoding="utf-8") as f:
    text = f.read()
m = re.search(r"^## \[Unreleased\]\s*\n(.*?)(?=^## \[|\Z)", text, re.M | re.S)
if not m:
    print("WARN: no [Unreleased] section in CHANGELOG.md — skipped.", file=sys.stderr)
else:
    body = m.group(1).strip("\n")
    if not body.strip():
        print("WARN: [Unreleased] had no notes — created an empty release section; "
              "add release notes before tagging.", file=sys.stderr)
    block = f"## [Unreleased]\n\n## [{new}] - {date}\n\n"
    if body.strip():
        block += body.rstrip() + "\n\n"
    text = text[:m.start()] + block + text[m.end():]
    with open(cl, "w", encoding="utf-8") as f:
        f.write(text)
    changed.append("CHANGELOG.md")

print(f"Bumped {old} -> {new}")
print("Changed files:")
for c in changed:
    print("  " + c)
print("\nNext:")
print("  1. Review the diff (esp. CHANGELOG.md release notes).")
print("  2. ./scripts/verify-release.sh")
print(f"  3. git commit, then: git tag {newv} && git push origin {newv}")
PY
