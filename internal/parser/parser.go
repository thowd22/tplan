package parser

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/yourusername/tplan/internal/models"
)

// Parser handles parsing Terraform plans
type Parser struct{}

// NewParser creates a new parser instance
func NewParser() *Parser {
	return &Parser{}
}

// ParseBytes parses Terraform plan from byte slice
func (p *Parser) ParseBytes(data []byte) (*models.PlanResult, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("no input provided")
	}

	// Only support full JSON plan format
	if !p.isValidJSON(data) {
		return nil, fmt.Errorf(`invalid input format

tplan only supports Terraform JSON plan format.

Usage:
  tplan

The tool will automatically run terraform/tofu plan and show results.`)
	}

	result, err := p.parseJSON(data)
	if err != nil {
		return nil, err
	}

	result.ParsedAt = time.Now()
	result.InputFormat = "json"

	// Calculate summary
	p.calculateSummary(result)

	return result, nil
}

// isValidJSON checks if the input is valid full JSON plan format
func (p *Parser) isValidJSON(data []byte) bool {
	trimmed := strings.TrimSpace(string(data))

	// Must start with {
	if !strings.HasPrefix(trimmed, "{") {
		return false
	}

	// Try to parse as JSON
	var js map[string]interface{}
	if err := json.Unmarshal(data, &js); err != nil {
		return false
	}

	// Must have format_version field (indicates full plan JSON, not streaming)
	if _, ok := js["format_version"]; !ok {
		return false
	}

	return true
}

// parseJSON parses the full Terraform JSON plan format
func (p *Parser) parseJSON(data []byte) (*models.PlanResult, error) {
	var plan tfjson.Plan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("failed to parse JSON plan: %w", err)
	}

	result := &models.PlanResult{
		FormatVersion:    plan.FormatVersion,
		TerraformVersion: plan.TerraformVersion,
		Resources:        make([]models.ResourceChange, 0),
		OutputChanges:    make([]models.OutputChange, 0),
		Errors:           make([]models.PlanError, 0),
		Warnings:         make([]models.PlanWarning, 0),
		DriftedResources: make([]models.DriftedResource, 0),
	}

	// Parse resource changes
	if plan.ResourceChanges != nil {
		for _, rc := range plan.ResourceChanges {
			if rc.Change == nil {
				continue
			}

			resourceChange := p.convertResourceChange(rc)
			result.Resources = append(result.Resources, resourceChange)

			// Check for drift
			if isDrift(rc) {
				result.DriftDetected = true
				result.DriftedResources = append(result.DriftedResources, models.DriftedResource{
					Address:     rc.Address,
					Type:        rc.Type,
					Name:        rc.Name,
					Module:      rc.ModuleAddress,
					Change:      resourceChange.Change,
					DriftReason: "Resource has drifted from expected state",
				})
			}
		}
	}

	// Parse output changes
	if plan.OutputChanges != nil {
		for name, oc := range plan.OutputChanges {
			if oc == nil {
				continue
			}

			result.OutputChanges = append(result.OutputChanges, models.OutputChange{
				Name:   name,
				Change: convertOutputChange(oc),
			})
		}
	}

	return result, nil
}

// convertResourceChange converts tfjson.ResourceChange to our internal model
func (p *Parser) convertResourceChange(rc *tfjson.ResourceChange) models.ResourceChange {
	change := models.ResourceChange{
		Address:      rc.Address,
		Type:         rc.Type,
		Name:         rc.Name,
		Mode:         string(rc.Mode),
		Module:       rc.ModuleAddress,
		ProviderName: rc.ProviderName,
		Index:        rc.Index,
		Deposed:      rc.DeposedKey,
	}

	if rc.Change != nil {
		change.Change = models.Change{
			Actions:         convertActions(rc.Change.Actions),
			Before:          convertToMap(rc.Change.Before),
			After:           convertToMap(rc.Change.After),
			AfterUnknown:    convertToMap(rc.Change.AfterUnknown),
			BeforeSensitive: convertToMap(rc.Change.BeforeSensitive),
			AfterSensitive:  convertToMap(rc.Change.AfterSensitive),
		}

		// Determine primary action
		change.Action = determineAction(rc.Change.Actions)

		// Set action reason if replacing
		if change.Action == models.ActionReplace {
			change.ActionReason = "forces replacement"
		}
	}

	return change
}

// Helper functions

// convertActions converts tfjson.Actions to string slice
func convertActions(actions tfjson.Actions) []string {
	result := make([]string, len(actions))
	for i, action := range actions {
		result[i] = string(action)
	}
	return result
}

// convertToMap converts interface{} to map[string]interface{}
func convertToMap(v interface{}) map[string]interface{} {
	if v == nil {
		return make(map[string]interface{})
	}

	if m, ok := v.(map[string]interface{}); ok {
		return m
	}

	return make(map[string]interface{})
}

// convertOutputChange converts tfjson.Change to our internal Change model
func convertOutputChange(oc *tfjson.Change) models.Change {
	return models.Change{
		Actions:      convertActions(oc.Actions),
		Before:       convertToMap(oc.Before),
		After:        convertToMap(oc.After),
		AfterUnknown: convertToMap(oc.AfterUnknown),
	}
}

// determineAction determines the primary action from a list of actions
func determineAction(actions tfjson.Actions) models.ChangeAction {
	if len(actions) == 0 {
		return models.ActionNoOp
	}

	// Handle replace (delete + create)
	hasDelete := false
	hasCreate := false
	for _, a := range actions {
		if a == tfjson.ActionDelete {
			hasDelete = true
		}
		if a == tfjson.ActionCreate {
			hasCreate = true
		}
	}
	if hasDelete && hasCreate {
		return models.ActionReplace
	}

	// Convert first action
	switch actions[0] {
	case tfjson.ActionCreate:
		return models.ActionCreate
	case tfjson.ActionUpdate:
		return models.ActionUpdate
	case tfjson.ActionDelete:
		return models.ActionDelete
	case tfjson.ActionRead:
		return models.ActionRead
	case tfjson.ActionNoop:
		return models.ActionNoOp
	default:
		return models.ActionNoOp
	}
}

// isDrift checks if a resource change represents drift
func isDrift(rc *tfjson.ResourceChange) bool {
	// Drift is detected when there are changes but the mode is "data"
	// or when the change is not part of the plan (out-of-band changes)
	// This is a simplified check - actual drift detection may be more complex
	return false // For now, we rely on explicit drift detection in the plan
}

// calculateSummary calculates aggregate statistics
func (p *Parser) calculateSummary(result *models.PlanResult) {
	summary := models.PlanSummary{}

	for _, rc := range result.Resources {
		summary.Total++
		switch rc.Action {
		case models.ActionCreate:
			summary.ToCreate++
		case models.ActionUpdate:
			summary.ToUpdate++
		case models.ActionDelete:
			summary.ToDelete++
		case models.ActionReplace:
			summary.ToReplace++
		case models.ActionNoOp:
			summary.NoOp++
		}
	}

	result.Summary = summary
}
