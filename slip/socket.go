package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// runtimeDir returns the per-user directory that holds slip sockets, creating
// it with 0700 permissions. It prefers $XDG_RUNTIME_DIR (a per-user, 0700 tmpfs
// on most Linux systems), then %LOCALAPPDATA% on Windows, and finally falls back
// to ~/.slip (e.g. on macOS). The directory's permissions are tightened on every
// call in case it pre-existed with looser bits; on Windows os.Chmod only toggles
// the read-only bit, but the chosen locations are already per-user.
func runtimeDir() (string, error) {
	var dir string
	switch {
	case os.Getenv("XDG_RUNTIME_DIR") != "":
		dir = filepath.Join(os.Getenv("XDG_RUNTIME_DIR"), "slip")
	case runtime.GOOS == "windows" && os.Getenv("LOCALAPPDATA") != "":
		dir = filepath.Join(os.Getenv("LOCALAPPDATA"), "slip")
	default:
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine a runtime directory: %w", err)
		}
		dir = filepath.Join(home, ".slip")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("cannot create runtime directory %s: %w", dir, err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return "", fmt.Errorf("cannot secure runtime directory %s: %w", dir, err)
	}
	return dir, nil
}

// socketPath returns the Unix socket path for a given ID within dir.
func socketPath(dir, id string) string {
	return filepath.Join(dir, id+".sock")
}
