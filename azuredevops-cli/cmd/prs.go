package cmd

import (
	"strings"

	"github.com/keith-hung/azuredevops-cli/internal/types"
)

// RunPRs lists pull requests in a repository.
func RunPRs(gf *GlobalFlags, args []string) {
	project := gf.Project
	repo := gf.Repo
	status := "active"

	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--project" && i+1 < len(args):
			i++
			project = args[i]
		case args[i] == "--repo" && i+1 < len(args):
			i++
			repo = args[i]
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

	c := NewClient(gf)

	repoID, err := c.ResolveRepoID(project, repo)
	if err != nil {
		ExitErrorInfer(err.Error())
	}

	result, err := c.ListPullRequests(project, repoID, status)
	if err != nil {
		ExitErrorInfer(err.Error())
	}

	prs := make([]types.PROutput, len(result.Value))
	for i, pr := range result.Value {
		prs[i] = apiPRToOutput(pr)
	}

	OutputJSON(types.PRsOutput{
		Success:      true,
		PullRequests: prs,
		Count:        len(prs),
	}, gf.Pretty)
}

// apiPRToOutput converts an API PR to a CLI output PR.
func apiPRToOutput(pr types.APIPR) types.PROutput {
	reviewers := make([]types.ReviewerOutput, len(pr.Reviewers))
	for j, r := range pr.Reviewers {
		reviewers[j] = types.ReviewerOutput{
			DisplayName: r.DisplayName,
			ID:          r.ID,
			UniqueName:  r.UniqueName,
			Vote:        r.Vote,
			VoteLabel:   VoteLabel(r.Vote),
			IsRequired:  r.IsRequired,
		}
	}

	return types.PROutput{
		ID:           pr.PullRequestID,
		Title:        pr.Title,
		Description:  pr.Description,
		Status:       pr.Status,
		CreatedBy:    pr.CreatedBy.DisplayName,
		CreationDate: pr.CreationDate,
		SourceBranch: strings.TrimPrefix(pr.SourceRefName, "refs/heads/"),
		TargetBranch: strings.TrimPrefix(pr.TargetRefName, "refs/heads/"),
		MergeStatus:  pr.MergeStatus,
		IsDraft:      pr.IsDraft,
		Reviewers:    reviewers,
	}
}
