package cmd

import (
	"fmt"

	"github.com/keith-hung/azuredevops-cli/internal/types"
	"github.com/spf13/cobra"
)

var prRejectCmd = &cobra.Command{
	Use:   "pr-reject",
	Short: "Reject a pull request (vote = -10)",
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

		if err := c.VotePullRequest(gf.Project, repoID, prID, userID, -10); err != nil {
			ExitErrorInfer(err.Error())
		}

		OutputJSON(types.PRVoteOutput{
			Success:       true,
			PullRequestID: prID,
			Vote:          "rejected",
			Message:       fmt.Sprintf("PR %d rejected", prID),
		}, gf.Pretty)
	},
}

func init() {
	prRejectCmd.Flags().Int("id", 0, "Pull request ID (required)")
	rootCmd.AddCommand(prRejectCmd)
}
