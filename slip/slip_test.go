package main

import (
	"bytes"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestRuntimeDirUsesXDG(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", tmp)

	dir, err := runtimeDir()
	if err != nil {
		t.Fatalf("runtimeDir: %v", err)
	}
	want := filepath.Join(tmp, "slip")
	if dir != want {
		t.Fatalf("dir = %q, want %q", dir, want)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat %s: %v", dir, err)
	}
	if !info.IsDir() {
		t.Fatalf("%s is not a directory", dir)
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		t.Fatalf("perm = %o, want 700", perm)
	}
}

func TestSocketPath(t *testing.T) {
	if got := socketPath("/run/slip", "13515"); got != "/run/slip/13515.sock" {
		t.Fatalf("socketPath = %q", got)
	}
}

func TestGenIDFormat(t *testing.T) {
	for i := 0; i < 200; i++ {
		id, err := genID()
		if err != nil {
			t.Fatalf("genID: %v", err)
		}
		if len(id) != 5 {
			t.Fatalf("id = %q, want 5 digits", id)
		}
		n, err := strconv.Atoi(id)
		if err != nil {
			t.Fatalf("id %q not numeric: %v", id, err)
		}
		if n < 10000 || n > 99999 {
			t.Fatalf("id %d out of range", n)
		}
	}
}

func TestSplitDoubleDash(t *testing.T) {
	cases := []struct {
		in     []string
		before []string
		after  []string
	}{
		{[]string{"--timeout", "30", "--", "cmd", "a"}, []string{"--timeout", "30"}, []string{"cmd", "a"}},
		{[]string{"--", "cmd"}, []string{}, []string{"cmd"}},
		{[]string{"--timeout", "5"}, []string{"--timeout", "5"}, nil},
		{[]string{"--", "cmd", "--", "x"}, []string{}, []string{"cmd", "--", "x"}},
	}
	for _, c := range cases {
		before, after := splitDoubleDash(c.in)
		if !equal(before, c.before) || !equal(after, c.after) {
			t.Fatalf("splitDoubleDash(%v) = %v, %v; want %v, %v", c.in, before, after, c.before, c.after)
		}
	}
}

func TestRunChildPipesStdinAndForwardsExit(t *testing.T) {
	var out, errb bytes.Buffer
	code := runChild([]string{"sh", "-c", "cat; exit 7"}, []byte("hunter2"), &out, &errb)
	if code != 7 {
		t.Fatalf("exit code = %d, want 7", code)
	}
	if out.String() != "hunter2" {
		t.Fatalf("stdout = %q, want %q", out.String(), "hunter2")
	}
	if errb.Len() != 0 {
		t.Fatalf("unexpected stderr: %q", errb.String())
	}
}

func TestRunChildMissingCommand(t *testing.T) {
	var out, errb bytes.Buffer
	code := runChild([]string{"slip-no-such-binary-xyz"}, []byte("x"), &out, &errb)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if errb.Len() == 0 {
		t.Fatalf("expected an error message on stderr")
	}
}

// TestAcceptLoopDeliversFirstPayload exercises the socket half of the daemon:
// a client connects, sends the value, half-closes, and the value arrives on the
// channel — the same path `slip set` drives.
func TestAcceptLoopDeliversFirstPayload(t *testing.T) {
	dir := t.TempDir()
	_, ln, sockPath, err := listenOnFreeID(dir)
	if err != nil {
		t.Fatalf("listenOnFreeID: %v", err)
	}
	defer ln.Close()
	defer os.Remove(sockPath)

	// Socket file must be 0600.
	if info, err := os.Stat(sockPath); err != nil {
		t.Fatalf("stat socket: %v", err)
	} else if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("socket perm = %o, want 600", perm)
	}

	valCh := make(chan []byte, 1)
	go acceptLoop(ln, valCh)

	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	if _, err := conn.Write([]byte("s3cret")); err != nil {
		t.Fatalf("write: %v", err)
	}
	conn.(*net.UnixConn).CloseWrite()

	select {
	case v := <-valCh:
		if string(v) != "s3cret" {
			t.Fatalf("value = %q, want %q", v, "s3cret")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for value")
	}
}

// TestAcceptLoopIgnoresEmptyPayload ensures a stray, empty connection does not
// falsely trigger the command; the real value delivered afterwards wins.
func TestAcceptLoopIgnoresEmptyPayload(t *testing.T) {
	dir := t.TempDir()
	_, ln, sockPath, err := listenOnFreeID(dir)
	if err != nil {
		t.Fatalf("listenOnFreeID: %v", err)
	}
	defer ln.Close()
	defer os.Remove(sockPath)

	valCh := make(chan []byte, 1)
	go acceptLoop(ln, valCh)

	// Empty connection: connect and close without sending.
	empty, err := net.Dial("unix", sockPath)
	if err != nil {
		t.Fatalf("dial empty: %v", err)
	}
	empty.Close()

	// Real value.
	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	conn.Write([]byte("real"))
	conn.(*net.UnixConn).CloseWrite()

	select {
	case v := <-valCh:
		if string(v) != "real" {
			t.Fatalf("value = %q, want %q", v, "real")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for value")
	}
}

func TestZero(t *testing.T) {
	b := []byte("secret")
	zero(b)
	for i, c := range b {
		if c != 0 {
			t.Fatalf("byte %d = %d, want 0", i, c)
		}
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
