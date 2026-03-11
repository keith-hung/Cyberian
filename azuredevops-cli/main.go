package main

import (
	"os"

	"github.com/keith-hung/azuredevops-cli/cmd"
)

func main() {
	if len(os.Args) < 2 {
		cmd.ExitError("usage: azuredevops <command> [flags]\ncommands: projects, repos, prs, my-prs, pr, pr-create, pr-update, pr-approve, pr-reject, pr-comment, pr-attachment, pr-reviewers, pr-add-reviewer, version", 1)
	}

	subCmd, otherArgs := extractSubcommand(os.Args[1:])

	gf, subArgs := cmd.ParseGlobalFlags(otherArgs)

	switch subCmd {
	case "version":
		cmd.RunVersion(gf)
	case "projects":
		cmd.RunProjects(gf)
	case "repos":
		cmd.RunRepos(gf, subArgs)
	case "prs":
		cmd.RunPRs(gf, subArgs)
	case "my-prs":
		cmd.RunMyPRs(gf, subArgs)
	case "pr":
		cmd.RunPR(gf, subArgs)
	case "pr-create":
		cmd.RunPRCreate(gf, subArgs)
	case "pr-update":
		cmd.RunPRUpdate(gf, subArgs)
	case "pr-approve":
		cmd.RunPRApprove(gf, subArgs)
	case "pr-reject":
		cmd.RunPRReject(gf, subArgs)
	case "pr-comment":
		cmd.RunPRComment(gf, subArgs)
	case "pr-attachment":
		cmd.RunPRAttachment(gf, subArgs)
	case "pr-reviewers":
		cmd.RunPRReviewers(gf, subArgs)
	case "pr-add-reviewer":
		cmd.RunPRAddReviewer(gf, subArgs)
	default:
		cmd.ExitError("unknown command: "+subCmd+"\ncommands: projects, repos, prs, my-prs, pr, pr-create, pr-update, pr-approve, pr-reject, pr-comment, pr-attachment, pr-reviewers, pr-add-reviewer, version", 1)
	}
}

// extractSubcommand finds and removes the subcommand from args.
func extractSubcommand(args []string) (string, []string) {
	commands := map[string]bool{
		"projects": true, "repos": true, "prs": true, "my-prs": true, "pr": true,
		"pr-create": true, "pr-update": true, "pr-approve": true,
		"pr-reject": true, "pr-comment": true, "pr-attachment": true,
		"pr-reviewers": true, "pr-add-reviewer": true, "version": true,
	}

	for i, arg := range args {
		if commands[arg] {
			remaining := make([]string, 0, len(args)-1)
			remaining = append(remaining, args[:i]...)
			remaining = append(remaining, args[i+1:]...)
			return arg, remaining
		}
	}

	// No known subcommand — treat first non-flag arg as the command.
	for i, arg := range args {
		if len(arg) > 0 && arg[0] != '-' {
			remaining := make([]string, 0, len(args)-1)
			remaining = append(remaining, args[:i]...)
			remaining = append(remaining, args[i+1:]...)
			return arg, remaining
		}
	}

	return "", args
}
