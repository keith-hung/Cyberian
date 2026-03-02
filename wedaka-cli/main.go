package main

import (
	"os"

	"github.com/keith-hung/wedaka-cli/cmd"
)

func main() {
	if len(os.Args) < 2 {
		cmd.ExitError("usage: wedaka <command> [flags]\ncommands: clock-in, clock-out, timelog, check-workday, version", 1)
	}

	subCmd, otherArgs := extractSubcommand(os.Args[1:])

	gf, subArgs := cmd.ParseGlobalFlags(otherArgs)

	switch subCmd {
	case "version":
		cmd.RunVersion(gf)
	case "clock-in":
		cmd.RunClockIn(gf, subArgs)
	case "clock-out":
		cmd.RunClockOut(gf, subArgs)
	case "timelog":
		cmd.RunTimelog(gf, subArgs)
	case "check-workday":
		cmd.RunCheckWorkday(gf, subArgs)
	default:
		cmd.ExitError("unknown command: "+subCmd+"\ncommands: clock-in, clock-out, timelog, check-workday, version", 1)
	}
}

// extractSubcommand finds and removes the subcommand from args.
func extractSubcommand(args []string) (string, []string) {
	commands := map[string]bool{
		"clock-in": true, "clock-out": true, "timelog": true,
		"check-workday": true, "version": true,
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
