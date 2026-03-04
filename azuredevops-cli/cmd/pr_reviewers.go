package cmd

import (
	"strconv"

	"github.com/keith-hung/azuredevops-cli/internal/types"
)

// RunPRReviewers lists reviewers of a pull request.
func RunPRReviewers(gf *GlobalFlags, args []string) {
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

	result, err := c.ListReviewers(project, repoID, prID)
	if err != nil {
		ExitErrorInfer(err.Error())
	}

	reviewers := make([]types.ReviewerOutput, len(result.Value))
	for i, r := range result.Value {
		reviewers[i] = types.ReviewerOutput{
			DisplayName: r.DisplayName,
			ID:          r.ID,
			UniqueName:  r.UniqueName,
			Vote:        r.Vote,
			VoteLabel:   VoteLabel(r.Vote),
			IsRequired:  r.IsRequired,
		}
	}

	OutputJSON(types.ReviewersOutput{
		Success:       true,
		PullRequestID: prID,
		Reviewers:     reviewers,
		Count:         len(reviewers),
	}, gf.Pretty)
}
