package cmd

import (
	"github.com/keith-hung/nouveau-timecard-cli/internal/types"
	"github.com/spf13/cobra"
)

var activitiesCmd = &cobra.Command{
	Use:   "activities",
	Short: "List the selectable activities for a project",
	Run: func(cmd *cobra.Command, args []string) {
		projectID, _ := cmd.Flags().GetString("project")
		if projectID == "" {
			ExitError("--project is required", 3)
		}
		year, month, err := monthFromFlags(cmd)
		if err != nil {
			ExitError(err.Error(), 3)
		}

		sess, err := NewSession(&gf)
		if err != nil {
			ExitError(err.Error(), 1)
		}

		activities, err := sess.GetActivities(year, month, projectID)
		if err != nil {
			ExitError("Failed to get activities: "+err.Error(), classifyError(err))
		}

		OutputJSON(types.ActivitiesOutput{
			ProjectID:  projectID,
			Activities: activities,
			Count:      len(activities),
		}, gf.Pretty)
	},
}

func init() {
	activitiesCmd.Flags().String("project", "", "Project ID (required)")
	addMonthFlags(activitiesCmd)
	rootCmd.AddCommand(activitiesCmd)
}
