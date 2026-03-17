package cmd

import (
	"github.com/keith-hung/azuredevops-cli/internal/types"
	"github.com/spf13/cobra"
)

// threadStatusLabel converts a thread status to a human-readable label.
func threadStatusLabel(status types.ThreadStatus) string {
	switch status {
	case 0:
		return "unknown"
	case 1:
		return "active"
	case 2:
		return "resolved"
	case 3:
		return "won't fix"
	case 4:
		return "closed"
	case 5:
		return "by design"
	case 6:
		return "pending"
	default:
		return "unknown"
	}
}

var prCommentsCmd = &cobra.Command{
	Use:   "pr-comments",
	Short: "List comment threads on a pull request",
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

		result, err := c.ListThreads(gf.Project, repoID, prID)
		if err != nil {
			ExitErrorInfer(err.Error())
		}

		threads := make([]types.ThreadOutput, 0, len(result.Value))
		for _, t := range result.Value {
			// Skip system-generated threads (commentType == "system")
			if len(t.Comments) == 0 || t.Comments[0].CommentType == "system" {
				continue
			}

			comments := make([]types.CommentOutput, 0, len(t.Comments))
			for _, co := range t.Comments {
				if co.IsDeleted {
					continue
				}
				comments = append(comments, types.CommentOutput{
					ID:            co.ID,
					Author:        co.Author.DisplayName,
					Content:       co.Content,
					PublishedDate: co.PublishedDate,
				})
			}

			if len(comments) == 0 {
				continue
			}

			threads = append(threads, types.ThreadOutput{
				ID:       t.ID,
				Status:   threadStatusLabel(t.Status),
				Comments: comments,
			})
		}

		OutputJSON(types.PRCommentsOutput{
			Success:       true,
			PullRequestID: prID,
			Threads:       threads,
			Count:         len(threads),
		}, gf.Pretty)
	},
}

func init() {
	prCommentsCmd.Flags().Int("id", 0, "Pull request ID (required)")
	rootCmd.AddCommand(prCommentsCmd)
}
