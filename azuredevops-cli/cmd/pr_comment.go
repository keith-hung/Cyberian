package cmd

import (
	"fmt"
	"strconv"

	"github.com/keith-hung/azuredevops-cli/internal/types"
)

// RunPRComment adds a comment to a pull request.
// If --thread-id is provided, replies to an existing thread.
// Otherwise, creates a new thread.
func RunPRComment(gf *GlobalFlags, args []string) {
	project := gf.Project
	repo := gf.Repo
	var prID int
	var threadID int
	var comment string

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
		case args[i] == "--thread-id" && i+1 < len(args):
			i++
			tid, err := strconv.Atoi(args[i])
			if err != nil {
				ExitError("--thread-id must be an integer", 3)
			}
			threadID = tid
		case args[i] == "--comment" && i+1 < len(args):
			i++
			comment = args[i]
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
	if comment == "" {
		ExitError("--comment is required", 3)
	}

	c := NewClient(gf)

	repoID, err := c.ResolveRepoID(project, repo)
	if err != nil {
		ExitErrorInfer(err.Error())
	}

	if threadID > 0 {
		if err := c.ReplyToThread(project, repoID, prID, threadID, comment); err != nil {
			ExitErrorInfer(err.Error())
		}
		OutputJSON(types.PRCommentOutput{
			Success:       true,
			PullRequestID: prID,
			ThreadID:      threadID,
			Message:       fmt.Sprintf("Reply added to thread %d on PR %d", threadID, prID),
		}, gf.Pretty)
	} else {
		newThreadID, err := c.CreateThread(project, repoID, prID, comment)
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
}
