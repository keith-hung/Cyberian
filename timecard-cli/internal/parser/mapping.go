package parser

import (
	"fmt"

	"github.com/keith-hung/timecard-cli/internal/types"
)

// BuildIndexMapping constructs the timearray index → project/activity mapping.
// Activities are ordered by project (in insertion order), then by activity within project.
// timearray[i][0] is a "select activity" placeholder, so activity indices start at 1.
func BuildIndexMapping(activities []types.ActivityEntry) types.IndexMapping {
	// Maintain insertion order of project IDs
	var projectIDs []string
	projectActivityMap := make(map[string][]types.ActivityEntry)

	for _, act := range activities {
		if _, exists := projectActivityMap[act.ProjectID]; !exists {
			projectIDs = append(projectIDs, act.ProjectID)
		}
		projectActivityMap[act.ProjectID] = append(projectActivityMap[act.ProjectID], act)
	}

	mapping := types.IndexMapping{
		ProjectIndexToID: make(map[int]string),
		ActivityByIndex:  make(map[string]types.ActivityEntry),
	}

	for i, pid := range projectIDs {
		mapping.ProjectIndexToID[i] = pid
		acts := projectActivityMap[pid]
		for j, act := range acts {
			// +1 because timearray[i][0] is the "select activity" placeholder
			key := fmt.Sprintf("%d_%d", i, j+1)
			mapping.ActivityByIndex[key] = act
		}
	}

	return mapping
}
