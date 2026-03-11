package cmd

import (
	"strings"

	"github.com/keith-hung/azuredevops-cli/internal/types"
	"github.com/spf13/cobra"
)

var prCreateCmd = &cobra.Command{
	Use:   "pr-create",
	Short: "Create a new pull request",
	Run: func(cmd *cobra.Command, args []string) {
		source, _ := cmd.Flags().GetString("source")
		target, _ := cmd.Flags().GetString("target")
		title, _ := cmd.Flags().GetString("title")
		description, _ := cmd.Flags().GetString("description")

		if gf.Project == "" {
			ExitError("--project or AZDO_PROJECT is required", 3)
		}
		if gf.Repo == "" {
			ExitError("--repo or AZDO_REPO is required", 3)
		}
		if source == "" {
			ExitError("--source is required", 3)
		}
		if target == "" {
			ExitError("--target is required", 3)
		}
		if title == "" {
			ExitError("--title is required", 3)
		}

		// Prepend refs/heads/ if not already present.
		if !strings.HasPrefix(source, "refs/") {
			source = "refs/heads/" + source
		}
		if !strings.HasPrefix(target, "refs/") {
			target = "refs/heads/" + target
		}

		c := NewClient(&gf)

		repoID, err := c.ResolveRepoID(gf.Project, gf.Repo)
		if err != nil {
			ExitErrorInfer(err.Error())
		}

		pr, err := c.CreatePullRequest(gf.Project, repoID, &types.PRCreateBody{
			SourceRefName: source,
			TargetRefName: target,
			Title:         title,
			Description:   description,
		})
		if err != nil {
			ExitErrorInfer(err.Error())
		}

		OutputJSON(types.PRCreateOutput{
			Success:       true,
			PullRequestID: pr.PullRequestID,
			URL:           pr.URL,
		}, gf.Pretty)
	},
}

func init() {
	prCreateCmd.Flags().String("source", "", "Source branch name (required)")
	prCreateCmd.Flags().String("target", "", "Target branch name (required)")
	prCreateCmd.Flags().String("title", "", "PR title (required)")
	prCreateCmd.Flags().String("description", "", "PR description")
	rootCmd.AddCommand(prCreateCmd)
}
