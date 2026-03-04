package cmd

import (
	"fmt"
	"strconv"

	"github.com/keith-hung/azuredevops-cli/internal/types"
)

// RunPRUpdate updates an existing pull request.
func RunPRUpdate(gf *GlobalFlags, args []string) {
	project := gf.Project
	repo := gf.Repo
	var prID int
	var title, description, status string

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
		case args[i] == "--title" && i+1 < len(args):
			i++
			title = args[i]
		case args[i] == "--description" && i+1 < len(args):
			i++
			description = args[i]
		case args[i] == "--status" && i+1 < len(args):
			i++
			status = args[i]
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
	if title == "" && description == "" && status == "" {
		ExitError("at least one of --title, --description, or --status is required", 3)
	}

	c := NewClient(gf)

	repoID, err := c.ResolveRepoID(project, repo)
	if err != nil {
		ExitErrorInfer(err.Error())
	}

	update := &types.PRUpdateBody{
		Title:       title,
		Description: description,
		Status:      status,
	}

	_, err = c.UpdatePullRequest(project, repoID, prID, update)
	if err != nil {
		ExitErrorInfer(err.Error())
	}

	OutputJSON(types.PRUpdateOutput{
		Success:       true,
		PullRequestID: prID,
		Message:       fmt.Sprintf("PR %d updated successfully", prID),
	}, gf.Pretty)
}
