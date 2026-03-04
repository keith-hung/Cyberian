package cmd

import (
	"strconv"

	"github.com/keith-hung/azuredevops-cli/internal/types"
)

// RunPR shows details of a single pull request.
func RunPR(gf *GlobalFlags, args []string) {
	project := gf.Project
	repo := gf.Repo
	var prID int

	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--project" && i+1 < len(args):
			i++
			project = args[i]
		case args[i] == "--repo" && i+1 < len(args):
			i++
			repo = args[i]
		case args[i] == "--id" && i+1 < len(args):
			i++
			id, err := strconv.Atoi(args[i])
			if err != nil {
				ExitError("--id must be an integer", 3)
			}
			prID = id
		}
	}

	if project == "" {
		ExitError("--project or AZDO_PROJECT is required", 3)
	}
	if repo == "" {
		ExitError("--repo or AZDO_REPO is required", 3)
	}
	if prID == 0 {
		ExitError("--id is required", 3)
	}

	c := NewClient(gf)

	repoID, err := c.ResolveRepoID(project, repo)
	if err != nil {
		ExitErrorInfer(err.Error())
	}

	pr, err := c.GetPullRequest(project, repoID, prID)
	if err != nil {
		ExitErrorInfer(err.Error())
	}

	OutputJSON(types.PRDetailOutput{
		Success:     true,
		PullRequest: apiPRToOutput(*pr),
	}, gf.Pretty)
}
