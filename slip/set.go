package main

import (
	"errors"
	"fmt"
	"net"
	"os"

	"golang.org/x/term"
)

// runSet implements `slip set <ID>`. It connects to the daemon for that ID
// first (so it can fail fast without prompting when no daemon is listening),
// reads the secret from the terminal with echo disabled, sends it, and prints a
// brief confirmation to stderr. The value is never echoed or printed.
func runSet(args []string) {
	if len(args) != 1 || args[0] == "" {
		fmt.Fprintln(os.Stderr, "slip set: usage: slip set <ID>")
		os.Exit(2)
	}
	id := args[0]

	dir, err := runtimeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "slip set: %v\n", err)
		os.Exit(1)
	}
	sockPath := socketPath(dir, id)

	// Connect before prompting so we don't ask for a secret we can't deliver.
	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "slip set: no daemon listening for ID %s (already consumed, timed out, or never existed)\n", id)
		os.Exit(1)
	}
	defer conn.Close()

	val, err := readSecret()
	if err != nil {
		fmt.Fprintf(os.Stderr, "slip set: %v\n", err)
		os.Exit(1)
	}
	if len(val) == 0 {
		zero(val)
		fmt.Fprintln(os.Stderr, "slip set: empty value, nothing sent")
		os.Exit(1)
	}

	if _, err := conn.Write(val); err != nil {
		zero(val)
		fmt.Fprintf(os.Stderr, "slip set: failed to send value: %v\n", err)
		os.Exit(1)
	}
	// Close now (not just via defer) so the daemon's ReadAll sees EOF and can
	// proceed. Closing a stream socket after a completed write still delivers the
	// buffered bytes on the local transports slip uses (AF_UNIX on Linux, macOS,
	// and Windows), so a plain Close is portable — no half-close needed.
	conn.Close()
	zero(val)
	fmt.Fprintf(os.Stderr, "sent to %s\n", id)
}

// readSecret reads a secret from the controlling terminal with echo disabled.
// It reads the terminal device directly (/dev/tty on Unix, CONIN$ on Windows;
// see openControllingTTY) rather than os.Stdin, so the prompt works even when
// the caller's stdin is redirected — matching slip's contract that the user
// types the value at their own terminal. If the terminal device cannot be
// opened it falls back to stdin, but only when stdin is itself a terminal.
func readSecret() ([]byte, error) {
	var fd int
	if tty, err := openControllingTTY(); err == nil {
		defer tty.Close()
		fd = int(tty.Fd())
	} else if term.IsTerminal(int(os.Stdin.Fd())) {
		fd = int(os.Stdin.Fd())
	} else {
		return nil, errors.New("no terminal available to read the secret (need an interactive TTY)")
	}

	fmt.Fprint(os.Stderr, "Enter value: ")
	b, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr) // ReadPassword consumes the newline; restore it
	if err != nil {
		return nil, fmt.Errorf("failed to read secret: %w", err)
	}
	return b, nil
}
