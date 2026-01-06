package models

import "time"

// GHRun represents a GitHub workflow run
type GHRun struct {
	DatabaseID   int       `json:"databaseId"`
	DisplayTitle string    `json:"displayTitle"`
	WorkflowName string    `json:"workflowName"`
	Status       string    `json:"status"`
	Conclusion   string    `json:"conclusion"`
	CreatedAt    time.Time `json:"createdAt"`
	HeadBranch   string    `json:"headBranch"`
}

// GHRunDetail contains the jobs for a workflow run
type GHRunDetail struct {
	Jobs []GHJob `json:"jobs"`
}

// GHJob represents a single job in a workflow run
type GHJob struct {
	Name         string `json:"name"`
	Status       string `json:"status"`
	Conclusion   string `json:"conclusion"`
	WorkflowName string
	RunID        int
}
