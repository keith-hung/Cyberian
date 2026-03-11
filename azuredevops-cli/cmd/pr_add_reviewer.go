package cmd

import (
	"fmt"

	"github.com/keith-hung/azuredevops-cli/internal/types"
	"github.com/spf13/cobra"
)

var prAddReviewerCmd = &cobra.Command{
	Use:   "pr-add-reviewer",
	Short: "Add a reviewer to a pull request",
	Run: func(cmd *cobra.Command, args []string) {
		prID, _ := cmd.Flags().GetInt("id")
		reviewer, _ := cmd.Flags().GetString("reviewer")

		if gf.Project == "" {
			ExitError("--project or AZDO_PROJECT is required", 3)
		}
		if gf.Repo == "" {
			ExitError("--repo or AZDO_REPO is required", 3)
		}
		if prID == 0 {
			ExitError("--id is required", 3)
		}
		if reviewer == "" {
			ExitError("--reviewer is required (GUID or domain\\username)", 3)
		}

		c := NewClient(&gf)

		repoID, err := c.ResolveRepoID(gf.Project, gf.Repo)
		if err != nil {
			ExitErrorInfer(err.Error())
		}

		// Resolve reviewer: if not a GUID, look up via Identity API.
		reviewerID := reviewer
		if !isGUID(reviewer) {
			resolved, err := c.ResolveIdentityID(reviewer)
			if err != nil {
				ExitErrorInfer(err.Error())
			}
			reviewerID = resolved
		}

		if err := c.AddReviewer(gf.Project, repoID, prID, reviewerID); err != nil {
			ExitErrorInfer(err.Error())
		}

		OutputJSON(types.AddReviewerOutput{
			Success:       true,
			PullRequestID: prID,
			Reviewer:      reviewer,
			Message:       fmt.Sprintf("Reviewer %q added to PR %d", reviewer, prID),
		}, gf.Pretty)
	},
}

func init() {
	prAddReviewerCmd.Flags().Int("id", 0, "Pull request ID (required)")
	prAddReviewerCmd.Flags().String("reviewer", "", "Reviewer GUID or domain\\username (required)")
	rootCmd.AddCommand(prAddReviewerCmd)
}
