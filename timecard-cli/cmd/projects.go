package cmd

import (
	"github.com/keith-hung/timecard-cli/internal/types"
	"github.com/spf13/cobra"
)

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "List available projects",
	Run: func(cmd *cobra.Command, args []string) {
		sess, err := NewSession(&gf)
		if err != nil {
			ExitError(err.Error(), 1)
		}
		if err := sess.EnsureAuth(); err != nil {
			ExitError("Authentication failed: "+err.Error(), 2)
		}

		projects, err := sess.GetProjects("")
		if err != nil {
			ExitError("Failed to get projects: "+err.Error(), 1)
		}

		OutputJSON(types.ProjectsOutput{
			Projects: projects,
			Count:    len(projects),
		}, gf.Pretty)
	},
}

func init() {
	rootCmd.AddCommand(projectsCmd)
}
