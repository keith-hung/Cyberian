//go:build windows

package main

import "os"

// openControllingTTY opens the Windows console input buffer (CONIN$) so the
// secret can be read with echo disabled even when stdin is redirected — the
// Windows analogue of /dev/tty.
func openControllingTTY() (*os.File, error) {
	return os.OpenFile("CONIN$", os.O_RDWR, 0)
}
