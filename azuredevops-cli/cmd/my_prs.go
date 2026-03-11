package cmd

import (
	"sort"
	"strings"

	"github.com/keith-hung/azuredevops-cli/internal/types"
	"github.com/spf13/cobra"
)

var myPrsCmd = &cobra.Command{
	Use:   "my-prs",
	Short: "List pull requests where you are creator or reviewer",
	Run: func(cmd *cobra.Command, args []string) {
		status, _ := cmd.Flags().GetString("status")
		role, _ := cmd.Flags().GetString("role")

		switch status {
		case "active", "completed", "abandoned", "all":
		default:
			ExitError("--status must be one of: active, completed, abandoned, all", 3)
		}

		switch role {
		case "all", "creator", "reviewer":
		default:
			ExitError("--role must be one of: all, creator, reviewer", 3)
		}

		c := NewClient(&gf)

		userID, err := c.GetCurrentUserID()
		if err != nil {
			ExitErrorInfer(err.Error())
		}

		// Collect PRs based on role, deduplicating by PR ID.
		type entry struct {
			pr    types.APIPR
			roles []string
		}
		prMap := make(map[int]*entry)

		if role == "creator" || role == "all" {
			result, err := c.ListMyPullRequests(status, "searchCriteria.creatorId", userID)
			if err != nil {
				ExitErrorInfer(err.Error())
			}
			for _, pr := range result.Value {
				e := prMap[pr.PullRequestID]
				if e == nil {
					e = &entry{pr: pr}
					prMap[pr.PullRequestID] = e
				}
				e.roles = append(e.roles, "creator")
			}
		}

		if role == "reviewer" || role == "all" {
			result, err := c.ListMyPullRequests(status, "searchCriteria.reviewerId", userID)
			if err != nil {
				ExitErrorInfer(err.Error())
			}
			for _, pr := range result.Value {
				e := prMap[pr.PullRequestID]
				if e == nil {
					e = &entry{pr: pr}
					prMap[pr.PullRequestID] = e
				}
				e.roles = append(e.roles, "reviewer")
			}
		}

		// Convert to output, applying optional project filter.
		prs := make([]types.MyPROutput, 0, len(prMap))
		for _, e := range prMap {
			pr := e.pr

			if gf.Project != "" && !strings.EqualFold(pr.Repository.Project.Name, gf.Project) {
				continue
			}

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

			prs = append(prs, types.MyPROutput{
				ID:           pr.PullRequestID,
				Title:        pr.Title,
				Status:       pr.Status,
				CreatedBy:    pr.CreatedBy.DisplayName,
				CreationDate: pr.CreationDate,
				SourceBranch: strings.TrimPrefix(pr.SourceRefName, "refs/heads/"),
				TargetBranch: strings.TrimPrefix(pr.TargetRefName, "refs/heads/"),
				MergeStatus:  pr.MergeStatus,
				IsDraft:      pr.IsDraft,
				Project:      pr.Repository.Project.Name,
				Repo:         pr.Repository.Name,
				Roles:        e.roles,
				Reviewers:    reviewers,
			})
		}

		// Sort by creation date descending (newest first).
		sort.Slice(prs, func(i, j int) bool {
			return prs[i].CreationDate > prs[j].CreationDate
		})

		OutputJSON(types.MyPRsOutput{
			Success:      true,
			Role:         role,
			Status:       status,
			Project:      gf.Project,
			PullRequests: prs,
			Count:        len(prs),
		}, gf.Pretty)
	},
}

func init() {
	myPrsCmd.Flags().String("status", "active", "Filter by status (active|completed|abandoned|all)")
	myPrsCmd.Flags().String("role", "all", "Filter by role (all|creator|reviewer)")
	rootCmd.AddCommand(myPrsCmd)
}
