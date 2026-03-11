package cmd

import (
	"fmt"

	"github.com/keith-hung/azuredevops-cli/internal/types"
	"github.com/spf13/cobra"
)

var prUpdateCmd = &cobra.Command{
	Use:   "pr-update",
	Short: "Update an existing pull request",
	Run: func(cmd *cobra.Command, args []string) {
		prID, _ := cmd.Flags().GetInt("id")
		title, _ := cmd.Flags().GetString("title")
		description, _ := cmd.Flags().GetString("description")
		status, _ := cmd.Flags().GetString("status")

		if gf.Project == "" {
			ExitError("--project or AZDO_PROJECT is required", 3)
		}
		if gf.Repo == "" {
			ExitError("--repo or AZDO_REPO is required", 3)
		}
		if prID == 0 {
			ExitError("--id is required", 3)
		}
		if title == "" && description == "" && status == "" {
			ExitError("at least one of --title, --description, or --status is required", 3)
		}

		c := NewClient(&gf)

		repoID, err := c.ResolveRepoID(gf.Project, gf.Repo)
		if err != nil {
			ExitErrorInfer(err.Error())
		}

		update := &types.PRUpdateBody{
			Title:       title,
			Description: description,
			Status:      status,
		}

		_, err = c.UpdatePullRequest(gf.Project, repoID, prID, update)
		if err != nil {
			ExitErrorInfer(err.Error())
		}

		OutputJSON(types.PRUpdateOutput{
			Success:       true,
			PullRequestID: prID,
			Message:       fmt.Sprintf("PR %d updated successfully", prID),
		}, gf.Pretty)
	},
}

func init() {
	prUpdateCmd.Flags().Int("id", 0, "Pull request ID (required)")
	prUpdateCmd.Flags().String("title", "", "New PR title")
	prUpdateCmd.Flags().String("description", "", "New PR description")
	prUpdateCmd.Flags().String("status", "", "New status (active|completed|abandoned)")
	rootCmd.AddCommand(prUpdateCmd)
}
