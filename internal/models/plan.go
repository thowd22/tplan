package models

import "time"

// ChangeAction represents the type of action being performed on a resource
type ChangeAction string

const (
	ActionCreate  ChangeAction = "create"
	ActionUpdate  ChangeAction = "update"
	ActionDelete  ChangeAction = "delete"
	ActionReplace ChangeAction = "replace"
	ActionRead    ChangeAction = "read"
	ActionNoOp    ChangeAction = "no-op"
)

// PlanResult contains all information parsed from a Terraform plan
type PlanResult struct {
	// Core plan data
	Resources        []ResourceChange
	OutputChanges    []OutputChange
	FormatVersion    string
	TerraformVersion string

	// Summary statistics
	Summary PlanSummary

	// Additional metadata
	Errors           []PlanError
	Warnings         []PlanWarning
	DriftDetected    bool
	DriftedResources []DriftedResource

	// Parse metadata
	ParsedAt    time.Time
	InputFormat string // "json" or "text"
}

// PlanSummary provides aggregate statistics about the plan
type PlanSummary struct {
	ToCreate  int
	ToUpdate  int
	ToDelete  int
	ToReplace int
	NoOp      int
	Total     int
}

// ResourceChange represents a single resource change in the plan
type ResourceChange struct {
	// Resource identification
	Address      string
	Type         string
	Name         string
	Module       string
	Mode         string // "managed" or "data"
	ProviderName string

	// Change information
	Change       Change
	Action       ChangeAction
	ActionReason string // Why this action is being taken (e.g., "forces replacement")

	// Additional metadata
	Index   interface{} // For resources with count or for_each
	Deposed string      // Deposed object ID if applicable

	// Dependencies - addresses of resources this resource depends on
	Dependencies []string

	// Drift information (populated when -drift flag is used)
	DriftInfo *DriftInfo
}

// Change represents the before/after state of a resource
type Change struct {
	Actions         []string // Raw actions from Terraform
	Before          map[string]interface{}
	After           map[string]interface{}
	AfterUnknown    map[string]interface{} // Values that will be known after apply
	BeforeSensitive map[string]interface{} // Sensitive values in before state
	AfterSensitive  map[string]interface{} // Sensitive values in after state

	// Replacement information
	ReplacePaths [][]interface{} // Paths that are forcing replacement
}

// OutputChange represents a change to a Terraform output
type OutputChange struct {
	Name      string
	Change    Change
	Sensitive bool
	Type      string
}

// PlanError represents an error encountered during planning
type PlanError struct {
	Message  string
	Resource string // Optional: resource that caused the error
	Severity string // "error" or "fatal"
}

// PlanWarning represents a warning from the plan
type PlanWarning struct {
	Message  string
	Resource string // Optional: resource related to warning
}

// DriftedResource represents a resource that has drifted from its expected state
type DriftedResource struct {
	Address     string
	Type        string
	Name        string
	Module      string
	Change      Change
	DriftReason string
}

// Plan represents a parsed Terraform plan with enhanced metadata (legacy compatibility)
type Plan struct {
	Resources        []Resource
	OutputChanges    []OutputChange
	FormatVersion    string
	TerraformVersion string
}

// Resource represents a single resource change in the plan (legacy compatibility)
type Resource struct {
	Address      string
	Type         string
	Name         string
	Mode         string // "managed" or "data"
	ProviderName string
	Change       Change
}
