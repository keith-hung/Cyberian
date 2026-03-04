package cmd

import "github.com/keith-hung/azuredevops-cli/internal/types"

// Version info set via ldflags at build time.
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

// RunVersion prints version information.
func RunVersion(gf *GlobalFlags) {
	OutputJSON(types.VersionOutput{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
	}, gf.Pretty)
}
