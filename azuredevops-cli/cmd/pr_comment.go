package cmd

import (
	"fmt"

	"github.com/keith-hung/azuredevops-cli/internal/types"
	"github.com/spf13/cobra"
)

var prCommentCmd = &cobra.Command{
	Use:   "pr-comment",
	Short: "Add a comment to a pull request",
	Long:  "Creates a new comment thread or replies to an existing thread if --thread-id is provided.",
	Run: func(cmd *cobra.Command, args []string) {
		prID, _ := cmd.Flags().GetInt("id")
		threadID, _ := cmd.Flags().GetInt("thread-id")
		comment, _ := cmd.Flags().GetString("comment")

		if gf.Project == "" {
			ExitError("--project or AZDO_PROJECT is required", 3)
		}
		if gf.Repo == "" {
			ExitError("--repo or AZDO_REPO is required", 3)
		}
		if prID == 0 {
			ExitError("--id is required", 3)
		}
		if comment == "" {
			ExitError("--comment is required", 3)
		}

		c := NewClient(&gf)

		repoID, err := c.ResolveRepoID(gf.Project, gf.Repo)
		if err != nil {
			ExitErrorInfer(err.Error())
		}

		if threadID > 0 {
			if err := c.ReplyToThread(gf.Project, repoID, prID, threadID, comment); err != nil {
				ExitErrorInfer(err.Error())
			}
			OutputJSON(types.PRCommentOutput{
				Success:       true,
				PullRequestID: prID,
				ThreadID:      threadID,
				Message:       fmt.Sprintf("Reply added to thread %d on PR %d", threadID, prID),
			}, gf.Pretty)
		} else {
			newThreadID, err := c.CreateThread(gf.Project, repoID, prID, comment)
			if err != nil {
				ExitErrorInfer(err.Error())
			}
			OutputJSON(types.PRCommentOutput{
				Success:       true,
				PullRequestID: prID,
				ThreadID:      newThreadID,
				Message:       fmt.Sprintf("Comment added to PR %d (thread %d)", prID, newThreadID),
			}, gf.Pretty)
		}
	},
}

func init() {
	prCommentCmd.Flags().Int("id", 0, "Pull request ID (required)")
	prCommentCmd.Flags().Int("thread-id", 0, "Thread ID to reply to (optional)")
	prCommentCmd.Flags().String("comment", "", "Comment text (required)")
	rootCmd.AddCommand(prCommentCmd)
}
