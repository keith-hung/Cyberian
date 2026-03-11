package cmd

import (
	"fmt"

	"github.com/keith-hung/timecard-cli/internal/types"
	"github.com/spf13/cobra"
)

var activitiesCmd = &cobra.Command{
	Use:   "activities",
	Short: "List activities for a project",
	Run: func(cmd *cobra.Command, args []string) {
		projectID, _ := cmd.Flags().GetString("project")
		if projectID == "" {
			ExitError("--project is required", 3)
		}

		sess, err := NewSession(&gf)
		if err != nil {
			ExitError(err.Error(), 1)
		}
		if err := sess.EnsureAuth(); err != nil {
			ExitError("Authentication failed: "+err.Error(), 2)
		}

		allActivities, err := sess.GetActivities("")
		if err != nil {
			ExitError("Failed to get activities: "+err.Error(), 1)
		}

		// Filter by project ID and convert to output format
		var filtered []types.ActivityOutEntry
		for _, act := range allActivities {
			if act.ProjectID == projectID {
				filtered = append(filtered, types.ActivityOutEntry{
					ID:   act.UID,
					Name: act.Name,
					Value: fmt.Sprintf("%s$%s$%s$%s",
						act.IsBottom, act.UID, act.ProjectID, act.Progress),
				})
			}
		}

		OutputJSON(types.ActivitiesOutput{
			ProjectID:  projectID,
			Activities: filtered,
			Count:      len(filtered),
		}, gf.Pretty)
	},
}

func init() {
	activitiesCmd.Flags().String("project", "", "Project ID (required)")
	rootCmd.AddCommand(activitiesCmd)
}
