# slip — ephemeral local secret broker

`slip` lets a user hand a secret to a command spawned by an AI coding agent
**without the secret value ever passing through the agent's readable context**
(its stdout, logs, or shell history).

The agent starts a short-lived daemon that blocks, waiting for the user to type
the secret in their own terminal. Once received, the daemon feeds the secret
into the target command's stdin, runs it, forwards the command's output and exit
code, then wipes the value and exits. The agent only ever sees the command's
real result — never the secret.

It was built for the `change-password` skill (so an agent can drive the password
portal without the password entering the conversation), but it is a generic
broker: any command that reads a secret from stdin works.

## Threat model — read this first

The **only** goal is to prevent *accidental* exposure of a secret into an
agent's readable context (stdout / logs / history).

`slip` is **not** designed to defend against a malicious or hijacked agent. The
agent runs under the same OS user and could read this process's memory or
connect to the socket directly. That is **accepted and out of scope**.

Consequently, by design, there is:

- no encryption and no cryptographic guarantee,
- no authentication or access control between same-user processes,
- only best-effort memory wiping (see below).

Do not use `slip` where the adversary is a same-user process. Use it to keep a
secret out of a *cooperating* agent's transcript.

## How it works

1. The agent runs `slip daemon --timeout <s> -- <cmd> [args...]`. The daemon
   listens on a Unix domain socket, prints a random 5-digit **ID** (and nothing
   else) to stdout, and blocks.
2. The agent tells the user the ID.
3. The user runs `slip set <ID>` **in their own terminal**, and types the secret
   at a hidden prompt (echo disabled, read from `/dev/tty`).
4. The daemon receives the value, pipes it into `<cmd>`'s stdin, closes stdin,
   and waits. `<cmd>`'s stdout, stderr, and exit code become the daemon's own.
5. The daemon zeroes the in-memory value, removes the socket, and exits.

One-shot: the daemon handles exactly one value and one command, then dies.

## Commands

```
slip daemon [--timeout <seconds>] -- <cmd> [args...]
    Start the broker (run by the agent). Prints only the ID to stdout, then
    blocks up to <seconds> (default 60) for a value. On a value, runs <cmd>
    with the secret on its stdin and exits with <cmd>'s exit code. On timeout
    or a signal (SIGINT/SIGTERM), prints an error to stderr, removes the
    socket, and exits non-zero.

slip set <ID>
    Send the secret to the daemon with that ID (run by the user). Reads the
    value with echo disabled; prints a brief confirmation ("sent to <ID>") to
    stderr. If no daemon is listening for that ID (already consumed, timed out,
    or never existed), prints a clear error to stderr and exits non-zero.

slip version
    Print the slip version.
```

### Socket location

Sockets live in a per-user runtime directory (mode `0700`), one file per ID
(mode `0600` on Unix):

- `$XDG_RUNTIME_DIR/slip/<ID>.sock` when `$XDG_RUNTIME_DIR` is set (typical on
  Linux),
- `%LOCALAPPDATA%\slip\<ID>.sock` on Windows, or
- `~/.slip/<ID>.sock` as a fallback (e.g. on macOS).

## Memory handling (best-effort only)

The secret is held as a `[]byte` and overwritten with zeros after the child
command has consumed it. This is **best-effort**: Go's garbage collector may
have already copied the value elsewhere, and that cannot be fully prevented
without `mlock`/`unsafe` tricks that are out of scope for the threat model
above. `mlock` is not used.

## Platform

Linux (including WSL2), macOS, and Windows. `slip` coordinates over a Unix domain
socket on every platform — Windows supports AF_UNIX since **Windows 10 1803 /
Server 2019** — and reads the secret from the terminal device (`/dev/tty` on Unix,
`CONIN$` on Windows). No third-party IPC library is used.

## Build

```bash
cd slip
go build -o slip .
```

Only dependency beyond the standard library is `golang.org/x/term` (for
echo-off terminal input).

Run the tests:

```bash
go test ./...            # unit tests (socket transport, ID, child piping, ...)
./test/e2e.sh            # end-to-end, Linux / macOS / WSL (drives real `slip set` via a PTY)
pwsh -File ./test/e2e.ps1  # end-to-end, Windows (PowerShell 7+)
```

The `test/e2e.*` scripts build the binary if needed, then check the full
daemon → `set` → child flow, the timeout path, and the no-daemon error — asserting
the secret never leaks into the daemon's output.

## End-to-end example

```console
# The agent runs:
$ slip daemon --timeout 30 -- ./migrate.sh
13515                      # <- only this ID is printed; the daemon now blocks

# The user runs in their own terminal:
$ slip set 13515
Enter value: ***           # echo off; the user types the DB password
sent to 13515

# The daemon receives the value -> pipes it into ./migrate.sh's stdin -> runs it
# -> forwards migrate.sh's output and exit code -> zeroes the value -> removes
# the socket -> exits.
# The agent only ever sees migrate.sh's output, never the password.
```
