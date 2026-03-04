package cmd

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/keith-hung/azuredevops-cli/internal/types"
)

var guidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// RunPRAddReviewer adds a reviewer to a pull request.
func RunPRAddReviewer(gf *GlobalFlags, args []string) {
	project := gf.Project
	repo := gf.Repo
	var prID int
	var reviewer string

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
		case args[i] == "--reviewer" && i+1 < len(args):
			i++
			reviewer = args[i]
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
	if reviewer == "" {
		ExitError("--reviewer is required (GUID or domain\\username)", 3)
	}

	c := NewClient(gf)

	repoID, err := c.ResolveRepoID(project, repo)
	if err != nil {
		ExitErrorInfer(err.Error())
	}

	// Use reviewer value directly — if it's a GUID, pass as-is.
	// If it's a domain\username, Azure DevOps Server should accept it.
	reviewerID := reviewer
	if !guidPattern.MatchString(reviewer) {
		// Not a GUID — pass as unique name; the API endpoint path requires an ID,
		// so we use the unique name as the identifier which Azure DevOps may accept.
		reviewerID = reviewer
	}

	if err := c.AddReviewer(project, repoID, prID, reviewerID); err != nil {
		ExitErrorInfer(err.Error())
	}

	OutputJSON(types.AddReviewerOutput{
		Success:       true,
		PullRequestID: prID,
		Reviewer:      reviewer,
		Message:       fmt.Sprintf("Reviewer %q added to PR %d", reviewer, prID),
	}, gf.Pretty)
}
