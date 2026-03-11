package cmd

import (
	"os"
	"path/filepath"

	"github.com/keith-hung/azuredevops-cli/internal/types"
	"github.com/spf13/cobra"
)

var prAttachmentCmd = &cobra.Command{
	Use:   "pr-attachment",
	Short: "Upload a file attachment to a pull request",
	Run: func(cmd *cobra.Command, args []string) {
		prID, _ := cmd.Flags().GetInt("id")
		filePath, _ := cmd.Flags().GetString("file")
		name, _ := cmd.Flags().GetString("name")

		if gf.Project == "" {
			ExitError("--project or AZDO_PROJECT is required", 3)
		}
		if gf.Repo == "" {
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

		c := NewClient(&gf)

		repoID, err := c.ResolveRepoID(gf.Project, gf.Repo)
		if err != nil {
			ExitErrorInfer(err.Error())
		}

		attachment, err := c.UploadAttachment(gf.Project, repoID, prID, name, f)
		if err != nil {
			ExitErrorInfer(err.Error())
		}

		OutputJSON(types.PRAttachmentOutput{
			Success:       true,
			PullRequestID: prID,
			Filename:      name,
			URL:           attachment.URL,
		}, gf.Pretty)
	},
}

func init() {
	prAttachmentCmd.Flags().Int("id", 0, "Pull request ID (required)")
	prAttachmentCmd.Flags().String("file", "", "File path to upload (required)")
	prAttachmentCmd.Flags().String("name", "", "Filename for the attachment (default: basename of --file)")
	rootCmd.AddCommand(prAttachmentCmd)
}
