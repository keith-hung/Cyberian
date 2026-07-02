#!/usr/bin/env bash
#
# verify-release.sh — pre-tag gate. Read-only: it changes nothing, it only reports.
# Run before `git tag`/`git push` (and it also runs as the first CI job in release.yml).
#
# Checks:
#   1. Version is identical across plugin.json, marketplace.json, and every launcher
#      (the 4 CLIs + slip, each .sh + .ps1).
#   2. CHANGELOG.md has a dated section for that version.
#   3. Every CLI/tool that ships a launcher is covered by BOTH build.sh and release.yml (P2 guard).
#   4. No forbidden tokens in tracked files (leak-guard list, if present).
#   5. Working tree is clean (warning only).
#
# Exit 0 = all hard checks pass; exit 1 = at least one hard check failed.
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
fail=0
ok()   { printf '  \033[32mOK\033[0m   %s\n' "$1"; }
bad()  { printf '  \033[31mFAIL\033[0m %s\n' "$1"; fail=1; }
warn() { printf '  \033[33mWARN\033[0m %s\n' "$1"; }

jv() { python3 -c "import json,sys;print(json.load(open(sys.argv[1]))['version'])" "$1"; }

echo "== 1. Version consistency =="
PV="$(jv .claude-plugin/plugin.json)"
echo "  plugin.json version: $PV"
MV="$(python3 -c "import json;d=json.load(open('.claude-plugin/marketplace.json'));print(next(p['version'] for p in d['plugins'] if p.get('name')=='cyberian'))")"
[ "$MV" = "$PV" ] \
  && ok "marketplace.json (cyberian) matches" || bad "marketplace.json cyberian version=$MV != $PV"
# CI-only: the pushed tag must match the version committed in the files.
if [ -n "${GITHUB_REF_NAME:-}" ]; then
  tagv="${GITHUB_REF_NAME#v}"
  [ "$tagv" = "$PV" ] && ok "tag $GITHUB_REF_NAME matches files" \
    || bad "tag $GITHUB_REF_NAME does not match file version v$PV"
fi
for cli in nouveau-timecard wedaka azuredevops chpw slip; do
  for f in "scripts/${cli}-launcher.sh" "scripts/${cli}-launcher.ps1"; do
    lv="$(grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' "$f" | head -1 | sed 's/^v//')"
    [ "$lv" = "$PV" ] && ok "$f ($lv)" || bad "$f is v$lv, expected v$PV"
  done
done

echo "== 2. CHANGELOG entry =="
if grep -qE "^## \[${PV//./\\.}\] - " CHANGELOG.md; then
  ok "CHANGELOG.md has a dated section for $PV"
else
  bad "CHANGELOG.md has no '## [$PV] - <date>' section"
fi

echo "== 3. CLI build coverage (P2 guard) =="
# Expected = every CLI dir that ships a launcher (jira excluded: upstream tool, no CLI dir here).
for cli in nouveau-timecard wedaka azuredevops chpw; do
  dir="${cli}-cli"
  [ -f "$dir/go.mod" ] || { bad "$dir/go.mod missing"; continue; }
  grep -q "\"$dir\"" scripts/build.sh && grep -q "$dir" .github/workflows/release.yml \
    && ok "$dir covered by build.sh + release.yml" \
    || bad "$dir NOT covered by build.sh and/or release.yml"
done
# slip: dir is "slip" (not slip-cli); the build lines emit slip_<os>_<arch>.
if [ -f "slip/go.mod" ]; then
  grep -q "slip_" scripts/build.sh && grep -q "slip_" .github/workflows/release.yml \
    && ok "slip covered by build.sh + release.yml" \
    || bad "slip NOT covered by build.sh and/or release.yml"
else
  bad "slip/go.mod missing"
fi

echo "== 4. Leak-guard token scan =="
TOKENS="dev/hooks/forbidden-tokens.txt"
if [ -f "$TOKENS" ]; then
  PAT="$(grep -vE '^[[:space:]]*(#|$)' "$TOKENS")"
  HITS="$(git grep -n -I -i -E -f <(printf '%s' "$PAT") -- ':!dev/' 2>/dev/null || true)"
  [ -z "$HITS" ] && ok "no forbidden tokens in tracked files" \
    || { bad "forbidden tokens found:"; printf '%s\n' "$HITS" | sed 's/^/       /'; }
else
  warn "no token list ($TOKENS) — skipped"
fi

echo "== 5. Working tree =="
[ -z "$(git status --porcelain)" ] && ok "clean" || warn "uncommitted changes present"

echo
if [ "$fail" -eq 0 ]; then
  echo "verify-release: PASS"
else
  echo "verify-release: FAIL — fix the above before tagging." >&2
fi
exit "$fail"
