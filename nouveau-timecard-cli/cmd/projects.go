package cmd

import (
	"github.com/keith-hung/nouveau-timecard-cli/internal/types"
	"github.com/spf13/cobra"
)

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "List the projects you can fill time against",
	Run: func(cmd *cobra.Command, args []string) {
		year, month, err := monthFromFlags(cmd)
		if err != nil {
			ExitError(err.Error(), 3)
		}

		sess, err := NewSession(&gf)
		if err != nil {
			ExitError(err.Error(), 1)
		}

		projects, err := sess.GetProjects(year, month)
		if err != nil {
			ExitError("Failed to get projects: "+err.Error(), classifyError(err))
		}

		OutputJSON(types.ProjectsOutput{Projects: projects, Count: len(projects)}, gf.Pretty)
	},
}

func init() {
	addMonthFlags(projectsCmd)
	rootCmd.AddCommand(projectsCmd)
}
