package cmd

import "github.com/keith-hung/timecard-cli/internal/types"

// Version info injected at build time via ldflags.
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

// RunVersion prints version info as JSON.
func RunVersion(gf *GlobalFlags) {
	OutputJSON(types.VersionOutput{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
	}, gf.Pretty)
}
