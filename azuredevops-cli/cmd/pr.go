package cmd

import (
	"github.com/keith-hung/azuredevops-cli/internal/types"
	"github.com/spf13/cobra"
)

var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "Show details of a single pull request",
	Run: func(cmd *cobra.Command, args []string) {
		prID, _ := cmd.Flags().GetInt("id")

		if gf.Project == "" {
			ExitError("--project or AZDO_PROJECT is required", 3)
		}
		if gf.Repo == "" {
			ExitError("--repo or AZDO_REPO is required", 3)
		}
		if prID == 0 {
			ExitError("--id is required", 3)
		}

		c := NewClient(&gf)

		repoID, err := c.ResolveRepoID(gf.Project, gf.Repo)
		if err != nil {
			ExitErrorInfer(err.Error())
		}

		pr, err := c.GetPullRequest(gf.Project, repoID, prID)
		if err != nil {
			ExitErrorInfer(err.Error())
		}

		OutputJSON(types.PRDetailOutput{
			Success:     true,
			PullRequest: apiPRToOutput(*pr),
		}, gf.Pretty)
	},
}

func init() {
	prCmd.Flags().Int("id", 0, "Pull request ID (required)")
	rootCmd.AddCommand(prCmd)
}
