package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

// runDaemon implements `slip daemon [--timeout N] -- <cmd> [args...]`.
//
// It listens on a Unix socket, prints only the socket's ID to stdout, then
// blocks until either a value arrives, the timeout elapses, or a signal is
// received. On a value it runs <cmd> with the value on its stdin and exits with
// <cmd>'s exit code. Every exit path removes the socket file first.
func runDaemon(args []string) {
	// Split "<flags> -- <cmd> [args...]" on the first bare "--".
	flagArgs, cmdArgs := splitDoubleDash(args)

	fs := flag.NewFlagSet("slip daemon", flag.ExitOnError)
	timeout := fs.Int("timeout", 60, "seconds to wait for a value before giving up")
	// flag writes usage/errors to stderr by default, so parsing never taints stdout.
	_ = fs.Parse(flagArgs)

	if len(cmdArgs) == 0 {
		fmt.Fprintln(os.Stderr, "slip daemon: missing command; usage: slip daemon [--timeout N] -- <cmd> [args...]")
		os.Exit(2)
	}
	if *timeout <= 0 {
		fmt.Fprintln(os.Stderr, "slip daemon: --timeout must be a positive number of seconds")
		os.Exit(2)
	}

	dir, err := runtimeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "slip daemon: %v\n", err)
		os.Exit(1)
	}

	id, ln, sockPath, err := listenOnFreeID(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "slip daemon: %v\n", err)
		os.Exit(1)
	}

	// cleanup closes the listener and removes the socket file. Safe to call more
	// than once. Because we exit via os.Exit (to forward the child's exit code),
	// deferred cleanup would not run, so every exit path calls this explicitly.
	cleanup := func() {
		ln.Close()
		os.Remove(sockPath)
	}

	// Handle termination signals so a Ctrl-C or kill still removes the socket.
	// os.Interrupt is portable (SIGINT on Unix, Ctrl+C on Windows); SIGTERM is
	// additionally honored where the OS delivers it.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Only the ID goes to stdout, telling the agent which ID to give the user.
	fmt.Println(id)

	// The first non-empty payload received becomes the secret.
	valCh := make(chan []byte, 1)
	go acceptLoop(ln, valCh)

	select {
	case val := <-valCh:
		code := runChild(cmdArgs, val, os.Stdout, os.Stderr)
		zero(val)
		cleanup()
		os.Exit(code)
	case <-time.After(time.Duration(*timeout) * time.Second):
		fmt.Fprintf(os.Stderr, "slip daemon: timed out after %ds waiting for a value on ID %s\n", *timeout, id)
		cleanup()
		os.Exit(1)
	case s := <-sigCh:
		fmt.Fprintf(os.Stderr, "slip daemon: received %s, aborting on ID %s\n", s, id)
		cleanup()
		os.Exit(1)
	}
}

// splitDoubleDash splits args on the first bare "--". Everything before it is
// returned as before (flags); everything after as after (the command). When no
// "--" is present, all args are flags and after is nil.
func splitDoubleDash(args []string) (before, after []string) {
	for i, a := range args {
		if a == "--" {
			return args[:i], args[i+1:]
		}
	}
	return args, nil
}

// listenOnFreeID generates random IDs until it finds one whose socket path is
// free, then listens on it (socket file mode 0600). A pre-existing socket file
// is reused only if nothing is currently listening there (stale from a crashed
// daemon); otherwise a different ID is tried.
func listenOnFreeID(dir string) (id string, ln net.Listener, sockPath string, err error) {
	var lastErr error
	for attempt := 0; attempt < 20; attempt++ {
		id, err = genID()
		if err != nil {
			return "", nil, "", err
		}
		sockPath = socketPath(dir, id)

		if _, statErr := os.Stat(sockPath); statErr == nil {
			if c, dialErr := net.Dial("unix", sockPath); dialErr == nil {
				c.Close() // a live daemon owns this ID; try another
				continue
			}
			os.Remove(sockPath) // stale socket, safe to reclaim
		}

		ln, err = net.Listen("unix", sockPath)
		if err != nil {
			// A busy ID is expected and worth retrying, but a structural
			// problem (e.g. the path exceeds the ~108-byte sun_path limit, or
			// the directory is unwritable) will fail identically every time.
			// Remember it so the caller sees the real cause, not just "no ID".
			lastErr = err
			continue
		}
		if err = os.Chmod(sockPath, 0o600); err != nil {
			ln.Close()
			os.Remove(sockPath)
			return "", nil, "", fmt.Errorf("cannot secure socket %s: %w", sockPath, err)
		}
		return id, ln, sockPath, nil
	}
	if lastErr != nil {
		return "", nil, "", fmt.Errorf("could not listen on a socket in %s: %w", dir, lastErr)
	}
	return "", nil, "", errors.New("could not allocate a free socket ID after 20 attempts")
}

// acceptLoop accepts connections and forwards the first non-empty payload on
// valCh. Each connection is drained in its own goroutine so a stalled client
// cannot block a later, well-behaved one. It returns when the listener closes.
func acceptLoop(ln net.Listener, valCh chan<- []byte) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return // listener closed
		}
		go func(c net.Conn) {
			defer c.Close()
			b, err := io.ReadAll(c)
			if err != nil || len(b) == 0 {
				return
			}
			select {
			case valCh <- b:
			default: // a value was already delivered; ignore extras
			}
		}(conn)
	}
}

// runChild spawns cmdArgs, writes val to its stdin and then closes it, forwards
// the child's stdout/stderr to the given writers, and returns the child's exit
// code. val is never written to stdout/stderr. On a failure to start or wait,
// it reports to stderr and returns 1.
func runChild(cmdArgs []string, val []byte, stdout, stderr io.Writer) int {
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		fmt.Fprintf(stderr, "slip daemon: cannot open child stdin: %v\n", err)
		return 1
	}
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(stderr, "slip daemon: cannot start %s: %v\n", cmdArgs[0], err)
		return 1
	}
	// Write the secret, then close stdin so the child sees EOF.
	if _, err := stdin.Write(val); err != nil {
		fmt.Fprintf(stderr, "slip daemon: cannot write to child stdin: %v\n", err)
	}
	stdin.Close()

	err = cmd.Wait()
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	fmt.Fprintf(stderr, "slip daemon: %s failed: %v\n", cmdArgs[0], err)
	return 1
}

// zero overwrites b in place. This is best-effort only: Go's garbage collector
// may already have copied the value elsewhere, and preventing that would need
// mlock/unsafe tricks that are out of scope for slip's threat model.
func zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
