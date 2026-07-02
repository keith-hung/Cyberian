//go:build !windows

package main

import "os"

// openControllingTTY opens the process's controlling terminal so the secret can
// be read with echo disabled, independent of whether stdin is redirected.
func openControllingTTY() (*os.File, error) {
	return os.OpenFile("/dev/tty", os.O_RDWR, 0)
}
