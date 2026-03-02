package session

import (
	"os"
)

// ClearSessionFile removes the session state file.
func ClearSessionFile(path string) {
	if path != "" {
		os.Remove(path)
	}
}
