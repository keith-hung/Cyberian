package cmd

import (
	"github.com/keith-hung/azuredevops-cli/internal/types"
	"github.com/spf13/cobra"
)

var prReviewersCmd = &cobra.Command{
	Use:   "pr-reviewers",
	Short: "List reviewers of a pull request",
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

		result, err := c.ListReviewers(gf.Project, repoID, prID)
		if err != nil {
			ExitErrorInfer(err.Error())
		}

		reviewers := make([]types.ReviewerOutput, len(result.Value))
		for i, r := range result.Value {
			reviewers[i] = types.ReviewerOutput{
				DisplayName: r.DisplayName,
				ID:          r.ID,
				UniqueName:  r.UniqueName,
				Vote:        r.Vote,
				VoteLabel:   VoteLabel(r.Vote),
				IsRequired:  r.IsRequired,
			}
		}

		OutputJSON(types.ReviewersOutput{
			Success:       true,
			PullRequestID: prID,
			Reviewers:     reviewers,
			Count:         len(reviewers),
		}, gf.Pretty)
	},
}

func init() {
	prReviewersCmd.Flags().Int("id", 0, "Pull request ID (required)")
	rootCmd.AddCommand(prReviewersCmd)
}
