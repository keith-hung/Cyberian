package cmd

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/keith-hung/azuredevops-cli/internal/types"
)

// RunPRAttachment uploads a file attachment to a pull request.
func RunPRAttachment(gf *GlobalFlags, args []string) {
	project := gf.Project
	repo := gf.Repo
	var prID int
	var filePath string
	var name string

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
		case args[i] == "--file" && i+1 < len(args):
			i++
			filePath = args[i]
		case args[i] == "--name" && i+1 < len(args):
			i++
			name = args[i]
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
	if filePath == "" {
		ExitError("--file is required", 3)
	}

	f, err := os.Open(filePath)
	if err != nil {
		ExitErrorInfer("cannot open file: " + err.Error())
	}
	defer f.Close()

	if name == "" {
		name = filepath.Base(filePath)
	}

	c := NewClient(gf)

	repoID, err := c.ResolveRepoID(project, repo)
	if err != nil {
		ExitErrorInfer(err.Error())
	}

	attachment, err := c.UploadAttachment(project, repoID, prID, name, f)
	if err != nil {
		ExitErrorInfer(err.Error())
	}

	OutputJSON(types.PRAttachmentOutput{
		Success:       true,
		PullRequestID: prID,
		Filename:      name,
		URL:           attachment.URL,
	}, gf.Pretty)
}
