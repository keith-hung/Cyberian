package cmd

import (
	"github.com/keith-hung/timecard-cli/internal/types"
	"github.com/spf13/cobra"
)

// Version info injected at build time via ldflags.
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		OutputJSON(types.VersionOutput{
			Version:   Version,
			Commit:    Commit,
			BuildDate: BuildDate,
		}, gf.Pretty)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
