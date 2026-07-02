#!/usr/bin/env bash
#
# slip end-to-end test — Linux / macOS / WSL.
#
# Exercises the real binary: the daemon prints an ID, `slip set` types a secret at
# the terminal (driven here through a pseudo-terminal), the secret is piped into
# the child, and the child's stdout + exit code are forwarded while the secret
# never appears in the daemon's own output.
#
# Requires python3 — the `slip set` prompt reads the terminal device with echo off
# and cannot be fed from a plain pipe, so a PTY is used to drive it. The slip
# binary is built automatically if `go` is available, or pass its path as $1.
#
# Usage: ./test/e2e.sh [path-to-slip-binary]
set -u

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SLIP_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

# 1. Locate or build the binary.
SLIP="${1:-}"
if [ -z "$SLIP" ]; then
  if [ -x "$SLIP_DIR/slip" ]; then
    SLIP="$SLIP_DIR/slip"
  elif command -v go >/dev/null 2>&1; then
    echo "building slip..." >&2
    ( cd "$SLIP_DIR" && go build -o slip . ) || { echo "build failed" >&2; exit 1; }
    SLIP="$SLIP_DIR/slip"
  else
    echo "error: no slip binary and no 'go' to build one; pass the binary path as \$1" >&2
    exit 1
  fi
fi
command -v python3 >/dev/null 2>&1 || { echo "error: python3 is required for this test" >&2; exit 1; }

# Short runtime dir keeps the AF_UNIX socket path under the ~108-byte limit.
RUNTIME="$(mktemp -d "${TMPDIR:-/tmp}/slip-e2e.XXXXXX")"
export XDG_RUNTIME_DIR="$RUNTIME"
trap 'rm -rf "$RUNTIME"' EXIT

pass=0; fail=0
ok() { echo "  PASS $1"; pass=$((pass+1)); }
no() { echo "  FAIL $1"; fail=$((fail+1)); }

echo "== Test A: happy path (daemon -> real 'slip set' via PTY -> child) =="
if SLIP="$SLIP" python3 - <<'PY'
import os, pty, select, subprocess, sys, time

slip = os.environ["SLIP"]
secret = "s3cr3t-value"  # 12 chars; the child reports only the length

d = subprocess.Popen(
    [slip, "daemon", "--timeout", "20", "--",
     "sh", "-c", 'read x; printf "child-len=%s\\n" "${#x}"; exit 5'],
    stdout=subprocess.PIPE, stderr=subprocess.PIPE)
sid = d.stdout.readline().decode().strip()
if not sid.isdigit():
    print("  daemon did not print a numeric ID:", repr(sid)); sys.exit(1)

pid, fd = pty.fork()
if pid == 0:  # child: run the real `slip set` attached to a PTY
    os.execv(slip, [slip, "set", sid]); os._exit(127)

buf = b""; typed = False; deadline = time.time() + 5
while time.time() < deadline:
    r, _, _ = select.select([fd], [], [], 0.2)
    if fd in r:
        try:
            data = os.read(fd, 1024)
        except OSError:
            break
        if not data:
            break
        buf += data
        if not typed and b"Enter value:" in buf:
            os.write(fd, (secret + "\n").encode()); typed = True
os.waitpid(pid, 0)

rc = d.wait()
out = d.stdout.read().decode(); err = d.stderr.read().decode()
ptyout = buf.decode(errors="replace")
problems = []
if rc != 5:
    problems.append(f"daemon exit={rc}, want 5")
if f"child-len={len(secret)}" not in out:
    problems.append(f"child did not receive the secret (stdout={out!r})")
if secret in out or secret in err or secret in ptyout:
    problems.append("secret LEAKED into output")
if problems:
    for p in problems: print("  -", p)
    sys.exit(1)
print(f"  daemon exit={rc}, child stdout={out.strip()!r}, secret not leaked")
sys.exit(0)
PY
then ok "happy path"; else no "happy path"; fi

echo "== Test B: timeout with no value =="
OUT="$(mktemp)"; ERR="$(mktemp)"
"$SLIP" daemon --timeout 1 -- sh -c 'echo should-not-run' >"$OUT" 2>"$ERR"
rc=$?
id="$(head -n1 "$OUT")"
sock="$RUNTIME/slip/$id.sock"
if [ "$rc" -ne 0 ] && [ -n "$id" ] && ! grep -q should-not-run "$OUT" && [ ! -S "$sock" ]; then
  ok "timeout (exit=$rc, only ID on stdout, socket cleaned)"
else
  no "timeout (exit=$rc, id='$id', socket present=$([ -S "$sock" ] && echo yes || echo no))"
fi
rm -f "$OUT" "$ERR"

echo "== Test C: 'slip set' with no daemon =="
if err="$("$SLIP" set 00000 2>&1)"; then
  no "no-daemon set should fail but exited 0"
else
  case "$err" in
    *"no daemon listening"*) ok "no-daemon set errors clearly and exits non-zero";;
    *) no "no-daemon set: unexpected message: $err";;
  esac
fi

echo
echo "Result: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
