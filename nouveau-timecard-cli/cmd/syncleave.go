package cmd

import (
	"github.com/spf13/cobra"
)

var syncLeaveCmd = &cobra.Command{
	Use:   "sync-leave",
	Short: "Sync approved leave from BPM into the 休假 activity (draft only)",
	Long: `Fetch the month's approved leave days from the BPM system and fill them
into the project's "休假" activity as a draft. Days already submitted or
approved are skipped. This command NEVER submits for approval.`,
	Run: func(cmd *cobra.Command, args []string) {
		year, month, err := monthFromFlags(cmd)
		if err != nil {
			ExitError(err.Error(), 3)
		}

		sess, err := NewSession(&gf)
		if err != nil {
			ExitError(err.Error(), 1)
		}

		out, err := sess.SyncLeave(year, month)
		if err != nil {
			ExitError(err.Error(), classifyError(err))
		}

		OutputJSON(out, gf.Pretty)
	},
}

func init() {
	addMonthFlags(syncLeaveCmd)
	rootCmd.AddCommand(syncLeaveCmd)
}
