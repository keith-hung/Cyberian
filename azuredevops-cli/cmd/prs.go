package cmd

import (
	"strings"

	"github.com/keith-hung/azuredevops-cli/internal/types"
	"github.com/spf13/cobra"
)

var prsCmd = &cobra.Command{
	Use:   "prs",
	Short: "List pull requests in a repository",
	Run: func(cmd *cobra.Command, args []string) {
		status, _ := cmd.Flags().GetString("status")

		if gf.Project == "" {
			ExitError("--project or AZDO_PROJECT is required", 3)
		}
		if gf.Repo == "" {
			ExitError("--repo or AZDO_REPO is required", 3)
		}

		c := NewClient(&gf)

		repoID, err := c.ResolveRepoID(gf.Project, gf.Repo)
		if err != nil {
			ExitErrorInfer(err.Error())
		}

		result, err := c.ListPullRequests(gf.Project, repoID, status)
		if err != nil {
			ExitErrorInfer(err.Error())
		}

		prs := make([]types.PROutput, len(result.Value))
		for i, pr := range result.Value {
			prs[i] = apiPRToOutput(pr)
		}

		OutputJSON(types.PRsOutput{
			Success:      true,
			PullRequests: prs,
			Count:        len(prs),
		}, gf.Pretty)
	},
}

func init() {
	prsCmd.Flags().String("status", "active", "Filter by status (active|completed|abandoned|all)")
	rootCmd.AddCommand(prsCmd)
}

// apiPRToOutput converts an API PR to a CLI output PR.
func apiPRToOutput(pr types.APIPR) types.PROutput {
	reviewers := make([]types.ReviewerOutput, len(pr.Reviewers))
	for j, r := range pr.Reviewers {
		reviewers[j] = types.ReviewerOutput{
			DisplayName: r.DisplayName,
			ID:          r.ID,
			UniqueName:  r.UniqueName,
			Vote:        r.Vote,
			VoteLabel:   VoteLabel(r.Vote),
			IsRequired:  r.IsRequired,
		}
	}

	return types.PROutput{
		ID:           pr.PullRequestID,
		Title:        pr.Title,
		Description:  pr.Description,
		Status:       pr.Status,
		CreatedBy:    pr.CreatedBy.DisplayName,
		CreationDate: pr.CreationDate,
		SourceBranch: strings.TrimPrefix(pr.SourceRefName, "refs/heads/"),
		TargetBranch: strings.TrimPrefix(pr.TargetRefName, "refs/heads/"),
		MergeStatus:  pr.MergeStatus,
		IsDraft:      pr.IsDraft,
		Reviewers:    reviewers,
	}
}
