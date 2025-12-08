package models

import "time"

// DriftInfo contains git information about a drifted resource
type DriftInfo struct {
	// ResourceName is the Terraform resource address (e.g., "aws_instance.web")
	ResourceName string

	// FilePath is the path to the Terraform file containing the resource
	FilePath string

	// CommitID is the SHA of the last commit that modified the file
	CommitID string

	// BranchName is the current git branch
	BranchName string

	// AuthorName is the name of the commit author
	AuthorName string

	// AuthorEmail is the email of the commit author
	AuthorEmail string

	// CommitDate is when the last modifying commit was made
	CommitDate time.Time

	// CommitMessage is the commit message of the last modifying commit
	CommitMessage string

	// DiffExplanation is a human-readable explanation of what changed
	DiffExplanation string

	// IsTracked indicates if the file is tracked by git
	IsTracked bool

	// HasUncommittedChanges indicates if the file has local modifications
	HasUncommittedChanges bool

	// Error contains any error message encountered during git operations
	Error string
}

// IsValid returns true if the drift info was successfully populated
func (d *DriftInfo) IsValid() bool {
	return d.Error == "" && d.IsTracked
}

// ShortCommitID returns the first 8 characters of the commit ID
func (d *DriftInfo) ShortCommitID() string {
	if len(d.CommitID) >= 8 {
		return d.CommitID[:8]
	}
	return d.CommitID
}

// StatusSummary returns a human-readable status of the file
func (d *DriftInfo) StatusSummary() string {
	if d.Error != "" {
		return "Error: " + d.Error
	}
	if !d.IsTracked {
		return "Not tracked by git"
	}
	if d.HasUncommittedChanges {
		return "Has uncommitted changes"
	}
	return "Up to date"
}
