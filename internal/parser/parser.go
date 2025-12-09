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

	// Build a map of resource configurations for dependency extraction
	configMap := make(map[string]*tfjson.ConfigResource)
	if plan.Config != nil && plan.Config.RootModule != nil {
		p.buildConfigMap(plan.Config.RootModule, "", configMap)
	}

	// Parse resource changes
	if plan.ResourceChanges != nil {
		for _, rc := range plan.ResourceChanges {
			if rc.Change == nil {
				continue
			}

			resourceChange := p.convertResourceChange(rc)

			// Extract dependencies from configuration
			if config, exists := configMap[rc.Address]; exists {
				resourceChange.Dependencies = p.extractDependenciesFromConfig(config, configMap)
			}

			// Also check After state for additional dependencies
			afterDeps := extractDependencies(rc.Change.After)
			for _, dep := range afterDeps {
				// Add if not already in dependencies
				found := false
				for _, existing := range resourceChange.Dependencies {
					if existing == dep {
						found = true
						break
					}
				}
				if !found {
					resourceChange.Dependencies = append(resourceChange.Dependencies, dep)
				}
			}

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

// buildConfigMap recursively builds a map of resource addresses to their configurations
func (p *Parser) buildConfigMap(module *tfjson.ConfigModule, modulePrefix string, configMap map[string]*tfjson.ConfigResource) {
	if module == nil {
		return
	}

	// Process resources in this module
	for _, res := range module.Resources {
		addr := res.Address
		if modulePrefix != "" {
			addr = modulePrefix + "." + addr
		}
		configMap[addr] = res
	}

	// Recursively process child modules
	for name, call := range module.ModuleCalls {
		childPrefix := name
		if modulePrefix != "" {
			childPrefix = modulePrefix + ".module." + name
		} else {
			childPrefix = "module." + name
		}
		if call.Module != nil {
			p.buildConfigMap(call.Module, childPrefix, configMap)
		}
	}
}

// extractDependenciesFromConfig extracts dependencies from a resource's configuration
func (p *Parser) extractDependenciesFromConfig(config *tfjson.ConfigResource, configMap map[string]*tfjson.ConfigResource) []string {
	deps := make([]string, 0)
	seen := make(map[string]bool)

	// First, add explicit depends_on
	for _, dep := range config.DependsOn {
		if !seen[dep] {
			seen[dep] = true
			deps = append(deps, dep)
		}
	}

	// Then extract references from expressions
	if config.Expressions != nil {
		for _, expr := range config.Expressions {
			p.extractDepsFromExpression(expr, &deps, seen, configMap)
		}
	}

	return deps
}

// extractDepsFromExpression recursively extracts resource references from expressions
func (p *Parser) extractDepsFromExpression(expr *tfjson.Expression, deps *[]string, seen map[string]bool, configMap map[string]*tfjson.ConfigResource) {
	if expr == nil {
		return
	}

	// Check for direct references
	if expr.References != nil {
		for _, ref := range expr.References {
			addr := extractResourceAddress(ref)
			if addr != "" && !seen[addr] {
				// Verify this resource exists in the plan
				if _, exists := configMap[addr]; exists {
					seen[addr] = true
					*deps = append(*deps, addr)
				}
			}
		}
	}

	// Recursively process nested expressions if any
	// Note: The Expression type may contain nested values, but the tfjson library
	// primarily exposes References which is what we need
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
		Dependencies: make([]string, 0),
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

		// Extract dependencies from After values
		change.Dependencies = extractDependencies(rc.Change.After)
	}

	return change
}

// Helper functions

// extractDependencies recursively searches for resource references in the After state
func extractDependencies(v interface{}) []string {
	deps := make([]string, 0)
	seen := make(map[string]bool)

	extractDepsRecursive(v, &deps, seen)
	return deps
}

// extractDepsRecursive recursively extracts resource addresses from nested structures
func extractDepsRecursive(v interface{}, deps *[]string, seen map[string]bool) {
	switch val := v.(type) {
	case map[string]interface{}:
		for _, v := range val {
			extractDepsRecursive(v, deps, seen)
		}
	case []interface{}:
		for _, item := range val {
			extractDepsRecursive(item, deps, seen)
		}
	case string:
		// Look for resource references - they typically contain resource type patterns
		// Examples: "aws_s3_bucket.example", "${aws_iam_role.example.arn}"
		if isResourceReference(val) {
			addr := extractResourceAddress(val)
			if addr != "" && !seen[addr] {
				seen[addr] = true
				*deps = append(*deps, addr)
			}
		}
	}
}

// isResourceReference checks if a string looks like a resource reference
func isResourceReference(s string) bool {
	// Check if string contains a resource type pattern (provider_service_resource.name)
	// Common patterns: aws_, google_, azurerm_, etc.
	return strings.Contains(s, "_") && strings.Contains(s, ".")
}

// extractResourceAddress extracts the resource address from various reference formats
func extractResourceAddress(s string) string {
	// Remove common wrapper patterns like ${...} or data.
	s = strings.TrimPrefix(s, "${")
	s = strings.TrimSuffix(s, "}")

	// Remove "data." prefix for data sources
	s = strings.TrimPrefix(s, "data.")

	// Split by "." and take the first two parts (type.name)
	parts := strings.Split(s, ".")
	if len(parts) >= 2 {
		// Check if the first part looks like a resource type
		if strings.Contains(parts[0], "_") {
			return parts[0] + "." + parts[1]
		}
	}

	return ""
}

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
