package cmd

import (
	"strconv"

	"github.com/keith-hung/azuredevops-cli/internal/types"
)

// threadStatusLabel converts a thread status integer to a human-readable label.
func threadStatusLabel(status int) string {
	switch status {
	case 0:
		return "unknown"
	case 1:
		return "active"
	case 2:
		return "fixed"
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

// RunPRComments lists comment threads on a pull request.
func RunPRComments(gf *GlobalFlags, args []string) {
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

	result, err := c.ListThreads(project, repoID, prID)
	if err != nil {
		ExitErrorInfer(err.Error())
	}

	threads := make([]types.ThreadOutput, 0, len(result.Value))
	for _, t := range result.Value {
		// Skip system-generated threads (commentType == "system")
		if len(t.Comments) > 0 && t.Comments[0].CommentType == "system" {
			continue
		}

		comments := make([]types.CommentOutput, 0, len(t.Comments))
		for _, c := range t.Comments {
			comments = append(comments, types.CommentOutput{
				ID:            c.ID,
				Author:        c.Author.DisplayName,
				Content:       c.Content,
				PublishedDate: c.PublishedDate,
			})
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
}
