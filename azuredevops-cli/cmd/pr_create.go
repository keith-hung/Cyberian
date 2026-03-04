package cmd

import (
	"strings"

	"github.com/keith-hung/azuredevops-cli/internal/types"
)

// RunPRCreate creates a new pull request.
func RunPRCreate(gf *GlobalFlags, args []string) {
	project := gf.Project
	repo := gf.Repo
	var source, target, title, description string

	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--project" && i+1 < len(args):
			i++
			project = args[i]
		case args[i] == "--repo" && i+1 < len(args):
			i++
			repo = args[i]
		case args[i] == "--source" && i+1 < len(args):
			i++
			source = args[i]
		case args[i] == "--target" && i+1 < len(args):
			i++
			target = args[i]
		case args[i] == "--title" && i+1 < len(args):
			i++
			title = args[i]
		case args[i] == "--description" && i+1 < len(args):
			i++
			description = args[i]
		}
	}

	if project == "" {
		ExitError("--project or AZDO_PROJECT is required", 3)
	}
	if repo == "" {
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

	c := NewClient(gf)

	repoID, err := c.ResolveRepoID(project, repo)
	if err != nil {
		ExitErrorInfer(err.Error())
	}

	pr, err := c.CreatePullRequest(project, repoID, &types.PRCreateBody{
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
}
