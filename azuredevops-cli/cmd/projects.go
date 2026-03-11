package cmd

import (
	"github.com/keith-hung/azuredevops-cli/internal/types"
	"github.com/spf13/cobra"
)

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "List all projects in the collection",
	Run: func(cmd *cobra.Command, args []string) {
		c := NewClient(&gf)

		result, err := c.ListProjects()
		if err != nil {
			ExitErrorInfer(err.Error())
		}

		projects := make([]types.ProjectOutput, len(result.Value))
		for i, p := range result.Value {
			projects[i] = types.ProjectOutput{
				ID:          p.ID,
				Name:        p.Name,
				Description: p.Description,
				State:       p.State,
			}
		}

		OutputJSON(types.ProjectsOutput{
			Success:  true,
			Projects: projects,
			Count:    len(projects),
		}, gf.Pretty)
	},
}

func init() {
	rootCmd.AddCommand(projectsCmd)
}
