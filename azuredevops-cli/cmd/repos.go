package cmd

import (
	"strings"

	"github.com/keith-hung/azuredevops-cli/internal/types"
	"github.com/spf13/cobra"
)

var reposCmd = &cobra.Command{
	Use:   "repos",
	Short: "List all repositories in a project",
	Run: func(cmd *cobra.Command, args []string) {
		if gf.Project == "" {
			ExitError("--project or AZDO_PROJECT is required", 3)
		}

		c := NewClient(&gf)

		result, err := c.ListRepos(gf.Project)
		if err != nil {
			ExitErrorInfer(err.Error())
		}

		repos := make([]types.RepoOutput, len(result.Value))
		for i, r := range result.Value {
			defaultBranch := r.DefaultBranch
			defaultBranch = strings.TrimPrefix(defaultBranch, "refs/heads/")
			repos[i] = types.RepoOutput{
				ID:            r.ID,
				Name:          r.Name,
				DefaultBranch: defaultBranch,
				RemoteURL:     r.RemoteURL,
				Size:          r.Size,
			}
		}

		OutputJSON(types.ReposOutput{
			Success: true,
			Project: gf.Project,
			Repos:   repos,
			Count:   len(repos),
		}, gf.Pretty)
	},
}

func init() {
	rootCmd.AddCommand(reposCmd)
}
