package cmd

import (
	"strings"

	"github.com/keith-hung/azuredevops-cli/internal/types"
)

// RunRepos lists all repositories in a project.
func RunRepos(gf *GlobalFlags, args []string) {
	project := gf.Project
	for i := 0; i < len(args); i++ {
		if args[i] == "--project" && i+1 < len(args) {
			i++
			project = args[i]
		}
	}

	if project == "" {
		ExitError("--project or AZDO_PROJECT is required", 3)
	}

	c := NewClient(gf)

	result, err := c.ListRepos(project)
	if err != nil {
		ExitErrorInfer(err.Error())
	}

	repos := make([]types.RepoOutput, len(result.Value))
	for i, r := range result.Value {
		defaultBranch := r.DefaultBranch
		defaultBranch = strings.TrimPrefix(defaultBranch, "refs/heads/")
		repos[i] = types.RepoOutput{
			ID:            r.ID,
			Name:          r.Name,
			DefaultBranch: defaultBranch,
			RemoteURL:     r.RemoteURL,
			Size:          r.Size,
		}
	}

	OutputJSON(types.ReposOutput{
		Success: true,
		Project: project,
		Repos:   repos,
		Count:   len(repos),
	}, gf.Pretty)
}
