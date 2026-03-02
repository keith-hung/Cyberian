package main

import (
	"os"

	"github.com/keith-hung/timecard-cli/cmd"
)

func main() {
	if len(os.Args) < 2 {
		cmd.ExitError("usage: timecard <command> [flags]\ncommands: projects, activities, timesheet, summary, save, version", 1)
	}

	// Find the subcommand, then pass all other args through global flag parsing.
	// Global flags can appear before or after the subcommand.
	subCmd, otherArgs := extractSubcommand(os.Args[1:])

	gf, subArgs := cmd.ParseGlobalFlags(otherArgs)

	switch subCmd {
	case "version":
		cmd.RunVersion(gf)
	case "projects":
		cmd.RunProjects(gf)
	case "activities":
		cmd.RunActivities(gf, subArgs)
	case "timesheet":
		cmd.RunTimesheet(gf, subArgs)
	case "summary":
		cmd.RunSummary(gf, subArgs)
	case "save":
		cmd.RunSave(gf, subArgs)
	default:
		cmd.ExitError("unknown command: "+subCmd+"\ncommands: projects, activities, timesheet, summary, save, version", 1)
	}
}

// extractSubcommand finds and removes the subcommand from args.
// Returns (subcommand, remainingArgs).
func extractSubcommand(args []string) (string, []string) {
	commands := map[string]bool{
		"projects": true, "activities": true, "timesheet": true,
		"summary": true, "save": true, "version": true,
	}

	for i, arg := range args {
		if commands[arg] {
			remaining := make([]string, 0, len(args)-1)
			remaining = append(remaining, args[:i]...)
			remaining = append(remaining, args[i+1:]...)
			return arg, remaining
		}
	}

	// No known subcommand — treat first non-flag arg as the command
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
