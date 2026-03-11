package cmd

import "github.com/spf13/cobra"

// Set via ldflags at build time.
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

type versionOutput struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		OutputJSON(versionOutput{
			Version:   Version,
			Commit:    Commit,
			BuildDate: BuildDate,
		}, gf.Pretty)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
