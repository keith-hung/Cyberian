package cmd

import (
	"github.com/keith-hung/azuredevops-cli/internal/types"
	"github.com/spf13/cobra"
)

// Version info set via ldflags at build time.
var (
	Version   = "dev"
	Commit    = "unknown"
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
