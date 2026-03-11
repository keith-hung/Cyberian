package cmd

import (
	"fmt"

	"github.com/keith-hung/azuredevops-cli/internal/types"
	"github.com/spf13/cobra"
)

var prApproveCmd = &cobra.Command{
	Use:   "pr-approve",
	Short: "Approve a pull request (vote = 10)",
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

		userID, err := c.GetCurrentUserID()
		if err != nil {
			ExitErrorInfer(err.Error())
		}

		if err := c.VotePullRequest(gf.Project, repoID, prID, userID, 10); err != nil {
			ExitErrorInfer(err.Error())
		}

		OutputJSON(types.PRVoteOutput{
			Success:       true,
			PullRequestID: prID,
			Vote:          "approved",
			Message:       fmt.Sprintf("PR %d approved", prID),
		}, gf.Pretty)
	},
}

func init() {
	prApproveCmd.Flags().Int("id", 0, "Pull request ID (required)")
	rootCmd.AddCommand(prApproveCmd)
}
