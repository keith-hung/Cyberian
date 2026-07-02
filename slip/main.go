// Command slip is an ephemeral local secret broker.
//
// slip lets a user hand a secret to a command spawned by an AI coding agent
// without the secret value ever passing through the agent's readable context
// (its stdout, logs, or history). The agent starts a short-lived daemon that
// blocks waiting for the user to type the secret in their own terminal; once
// received, the daemon feeds the secret into the target command's stdin, runs
// it, forwards the command's output and exit code, then wipes the value and
// exits.
//
// Threat model: the ONLY goal is to prevent accidental exposure of the secret
// into the agent's readable context. slip is NOT designed to defend against a
// malicious or hijacked agent — the agent runs as the same OS user and could
// read this process's memory or connect to the socket directly. That is
// accepted and out of scope. There are deliberately no cryptographic
// guarantees and no access control between same-user processes.
//
// Platform: Linux (including WSL2), macOS, and Windows. slip coordinates over a
// Unix domain socket on every platform — Windows supports AF_UNIX since Windows
// 10 1803 / Server 2019 — and reads the secret from the terminal device
// (/dev/tty on Unix, CONIN$ on Windows).
package main

import (
	"fmt"
	"os"
)

const usage = `slip — ephemeral local secret broker

Usage:
  slip daemon [--timeout <seconds>] -- <cmd> [args...]
        Start the broker (run by the agent). Prints a short numeric ID to
        stdout, then blocks up to <seconds> (default 60) for a value. When the
        value arrives it is piped into <cmd>'s stdin; <cmd>'s stdout, stderr,
        and exit code are forwarded as slip's own. The secret is never printed.

  slip set <ID>
        Send the secret to the daemon with that ID (run by the user in their
        own terminal). Reads the value with echo disabled; nothing is echoed.

  slip version
        Print the slip version.`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	switch os.Args[1] {
	case "daemon":
		runDaemon(os.Args[2:])
	case "set":
		runSet(os.Args[2:])
	case "version", "-v", "--version":
		fmt.Println(version)
	case "help", "-h", "--help":
		fmt.Println(usage)
	default:
		fmt.Fprintf(os.Stderr, "slip: unknown command %q\n\n%s\n", os.Args[1], usage)
		os.Exit(2)
	}
}
