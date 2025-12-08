package parser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/yourusername/tplan/internal/models"
)

// Parser handles parsing of Terraform plan output
type Parser struct {
	// Configuration options
	strictMode bool
}

// NewParser creates a new parser instance
func NewParser() *Parser {
	return &Parser{
		strictMode: false,
	}
}

// SetStrictMode enables or disables strict parsing mode
func (p *Parser) SetStrictMode(strict bool) {
	p.strictMode = strict
}

// Parse reads from the provided reader and parses the Terraform plan
// It automatically detects whether the input is JSON or human-readable format
func (p *Parser) Parse(reader io.Reader) (*models.PlanResult, error) {
	// Read all input
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("no input provided")
	}

	// Detect format
	format := p.detectFormat(data)

	var result *models.PlanResult
	switch format {
	case "json":
		result, err = p.parseJSON(data)
	case "text":
		result, err = p.parseText(data)
	default:
		return nil, fmt.Errorf("unable to detect input format")
	}

	if err != nil {
		return nil, err
	}

	result.ParsedAt = time.Now()
	result.InputFormat = format

	// Calculate summary
	p.calculateSummary(result)

	return result, nil
}

// detectFormat determines if the input is JSON or human-readable text
func (p *Parser) detectFormat(data []byte) string {
	// Trim whitespace
	trimmed := strings.TrimSpace(string(data))

	// Check if it starts with JSON object
	if strings.HasPrefix(trimmed, "{") && strings.Contains(trimmed, "\"format_version\"") {
		return "json"
	}

	// Check for common Terraform plan text patterns
	if strings.Contains(trimmed, "Terraform will perform") ||
		strings.Contains(trimmed, "No changes.") ||
		strings.Contains(trimmed, "terraform plan") {
		return "text"
	}

	// Try to parse as JSON
	var js map[string]interface{}
	if err := json.Unmarshal(data, &js); err == nil {
		return "json"
	}

	return "text" // Default to text if unsure
}

// parseJSON parses JSON format plan output (from terraform plan -json or terraform show -json)
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
				Name:      name,
				Sensitive: false, // Sensitivity info is in the change itself
				Change: models.Change{
					Actions:         convertActions(oc.Actions),
					Before:          convertToMap(oc.Before),
					After:           convertToMap(oc.After),
					AfterUnknown:    convertToMap(oc.AfterUnknown),
					BeforeSensitive: convertToMap(oc.BeforeSensitive),
					AfterSensitive:  convertToMap(oc.AfterSensitive),
				},
			})
		}
	}

	// Parse errors and warnings from plan metadata
	// Note: Error detection would require parsing plan diagnostics if available
	// in the JSON structure, or from the text output

	return result, nil
}

// parseText parses human-readable text format plan output
func (p *Parser) parseText(data []byte) (*models.PlanResult, error) {
	// Strip ANSI color codes that Terraform adds when outputting to terminal
	text := stripAnsiCodes(string(data))

	result := &models.PlanResult{
		Resources:        make([]models.ResourceChange, 0),
		OutputChanges:    make([]models.OutputChange, 0),
		Errors:           make([]models.PlanError, 0),
		Warnings:         make([]models.PlanWarning, 0),
		DriftedResources: make([]models.DriftedResource, 0),
	}

	scanner := bufio.NewScanner(strings.NewReader(text))

	var currentResource *models.ResourceChange
	var inResourceBlock bool
	var resourceLines []string
	var pendingAddress string
	var pendingAction models.ChangeAction
	var pendingReason string

	// Regular expressions for parsing
	resourceCommentRe := regexp.MustCompile(`^\s*#\s+([^\s]+(?:\[[^\]]+\])?)\s+(will be|must be)\s+(created|destroyed|updated|replaced|read)`)
	resourceLineRe := regexp.MustCompile(`^\s*([-+~])\s+resource\s+"([^"]+)"\s+"([^"]+)"`)
	errorRe := regexp.MustCompile(`^\s*Error:\s*(.+)$`)
	warningRe := regexp.MustCompile(`^\s*Warning:\s*(.+)$`)
	driftRe := regexp.MustCompile(`^\s*Note:.*drift.*detected|Objects have changed outside`)

	// Skip lines that are just progress/status messages
	skipLineRe := regexp.MustCompile(`^\s*(Refreshing|Acquiring|Releasing|Initializing|Preparing|Reading|Terraform will perform|Terraform used the selected providers)`)

	for scanner.Scan() {
		line := scanner.Text()

		// Skip progress/status lines
		if skipLineRe.MatchString(line) {
			continue
		}

		// Check for errors
		if matches := errorRe.FindStringSubmatch(line); matches != nil {
			result.Errors = append(result.Errors, models.PlanError{
				Message:  strings.TrimSpace(matches[1]),
				Severity: "error",
			})
			continue
		}

		// Check for warnings
		if matches := warningRe.FindStringSubmatch(line); matches != nil {
			result.Warnings = append(result.Warnings, models.PlanWarning{
				Message: strings.TrimSpace(matches[1]),
			})
			continue
		}

		// Check for drift
		if driftRe.MatchString(line) {
			result.DriftDetected = true
		}

		// Check for resource comment line (e.g., "# aws_instance.web will be created")
		if matches := resourceCommentRe.FindStringSubmatch(line); matches != nil {
			pendingAddress = matches[1]
			actionWord := matches[3]

			// Map action words to actions
			switch actionWord {
			case "created":
				pendingAction = models.ActionCreate
			case "destroyed":
				pendingAction = models.ActionDelete
			case "updated":
				pendingAction = models.ActionUpdate
			case "replaced":
				pendingAction = models.ActionReplace
			case "read":
				pendingAction = models.ActionRead
			}

			// Check for reason (e.g., "must be replaced")
			if matches[2] == "must be" {
				pendingReason = "must be " + actionWord
			}
			continue
		}

		// Check for resource line (e.g., "- resource "aws_instance" "web"")
		if matches := resourceLineRe.FindStringSubmatch(line); matches != nil {
			// Save previous resource if exists
			if currentResource != nil {
				p.parseResourceDetails(currentResource, resourceLines)
				result.Resources = append(result.Resources, *currentResource)
			}

			// Start new resource
			symbol := matches[1]
			resourceType := matches[2]
			resourceName := matches[3]

			// Use pending address if available, otherwise construct from resource line
			address := pendingAddress
			if address == "" {
				address = resourceType + "." + resourceName
			}

			currentResource = &models.ResourceChange{
				Address: address,
				Type:    resourceType,
				Name:    resourceName,
				Mode:    "managed",
			}

			// Use pending action if available, otherwise determine from symbol
			if pendingAction != "" {
				currentResource.Action = pendingAction
				currentResource.ActionReason = pendingReason
			} else {
				currentResource.Action = p.symbolToAction(symbol)
			}

			// Reset pending values
			pendingAddress = ""
			pendingAction = ""
			pendingReason = ""

			inResourceBlock = true
			resourceLines = []string{line}
			continue
		}

		// Collect resource details
		if inResourceBlock {
			resourceLines = append(resourceLines, line)

			// Check if block ended
			if strings.TrimSpace(line) == "}" {
				inResourceBlock = false
			}
		}
	}

	// Save last resource
	if currentResource != nil {
		p.parseResourceDetails(currentResource, resourceLines)
		result.Resources = append(result.Resources, *currentResource)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading input: %w", err)
	}

	// Try to extract Terraform version from text
	result.TerraformVersion = p.extractVersionFromText(string(data))

	// Calculate summary statistics
	p.calculateSummary(result)

	return result, nil
}

// convertResourceChange converts terraform-json ResourceChange to our model
func (p *Parser) convertResourceChange(rc *tfjson.ResourceChange) models.ResourceChange {
	change := models.ResourceChange{
		Address:      rc.Address,
		Type:         rc.Type,
		Name:         rc.Name,
		Module:       rc.ModuleAddress,
		Mode:         string(rc.Mode),
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

		// Note: ReplacePaths would be parsed from terraform-json if available
		// The field may not be present in all versions of the library

		// Determine primary action
		change.Action = determineAction(rc.Change.Actions)

		// Set action reason if replacing
		if change.Action == models.ActionReplace {
			change.ActionReason = "forces replacement"
		}
	}

	return change
}

// parseResourceAddress extracts type and name from resource address
func (p *Parser) parseResourceAddress(resource *models.ResourceChange, address string) {
	// Handle module prefix
	parts := strings.Split(address, ".")

	if len(parts) >= 2 {
		// Check if it starts with module
		if parts[0] == "module" {
			// Extract module name and continue parsing
			resource.Module = parts[1]
			parts = parts[2:] // Skip "module" and module name
		}

		if len(parts) >= 2 {
			resource.Type = parts[0]
			resource.Name = strings.Join(parts[1:], ".")
		}
	}

	// Determine mode
	if strings.HasPrefix(resource.Type, "data.") {
		resource.Mode = "data"
		resource.Type = strings.TrimPrefix(resource.Type, "data.")
	} else {
		resource.Mode = "managed"
	}
}

// parseResourceDetails extracts before/after values from text format
func (p *Parser) parseResourceDetails(resource *models.ResourceChange, lines []string) {
	resource.Change.Before = make(map[string]interface{})
	resource.Change.After = make(map[string]interface{})

	attributeRe := regexp.MustCompile(`^\s*([+~-])\s*(\w+)\s*=\s*(.+)$`)

	for _, line := range lines {
		if matches := attributeRe.FindStringSubmatch(line); matches != nil {
			symbol := matches[1]
			key := matches[2]
			value := strings.TrimSpace(matches[3])

			// Parse value (simplified)
			var parsedValue interface{}
			if value == "(known after apply)" {
				parsedValue = nil // Unknown value
			} else {
				parsedValue = value
			}

			switch symbol {
			case "+": // Added
				resource.Change.After[key] = parsedValue
			case "-": // Removed
				resource.Change.Before[key] = parsedValue
			case "~": // Changed
				resource.Change.Before[key] = parsedValue
				resource.Change.After[key] = parsedValue
			}
		}
	}
}

// symbolToAction converts the text format symbol to an action
func (p *Parser) symbolToAction(symbol string) models.ChangeAction {
	switch symbol {
	case "+":
		return models.ActionCreate
	case "-":
		return models.ActionDelete
	case "~":
		return models.ActionUpdate
	case "-/+", "+/-":
		return models.ActionReplace
	case "#":
		return models.ActionRead
	default:
		return models.ActionNoOp
	}
}

// stripAnsiCodes removes ANSI color codes from text
func stripAnsiCodes(text string) string {
	// ANSI escape sequence regex: \x1b\[[0-9;]*m
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansiRegex.ReplaceAllString(text, "")
}

// extractVersionFromText attempts to extract Terraform version from text output
func (p *Parser) extractVersionFromText(text string) string {
	versionRe := regexp.MustCompile(`Terraform\s+v?(\d+\.\d+\.\d+)`)
	if matches := versionRe.FindStringSubmatch(text); matches != nil {
		return matches[1]
	}
	return ""
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
func convertToMap(val interface{}) map[string]interface{} {
	if val == nil {
		return nil
	}

	switch v := val.(type) {
	case map[string]interface{}:
		return v
	case bool:
		// For boolean values like BeforeSensitive/AfterSensitive
		return nil
	default:
		// Try to marshal and unmarshal to convert
		data, err := json.Marshal(val)
		if err != nil {
			return nil
		}
		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			return nil
		}
		return result
	}
}

// determineAction determines the primary action from a list of actions
func determineAction(actions tfjson.Actions) models.ChangeAction {
	if len(actions) == 0 {
		return models.ActionNoOp
	}

	// Check for replace (delete + create)
	hasDelete := false
	hasCreate := false
	for _, action := range actions {
		switch action {
		case tfjson.ActionDelete:
			hasDelete = true
		case tfjson.ActionCreate:
			hasCreate = true
		}
	}

	if hasDelete && hasCreate {
		return models.ActionReplace
	}

	// Return the first action as primary
	switch actions[0] {
	case tfjson.ActionCreate:
		return models.ActionCreate
	case tfjson.ActionDelete:
		return models.ActionDelete
	case tfjson.ActionUpdate:
		return models.ActionUpdate
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
	// Drift is typically indicated by update actions on data sources
	// or when a managed resource has changes but no planned action
	if rc.Change == nil {
		return false
	}

	// Check if this is a drift-only change (no planned changes)
	// This is a simplified check - real drift detection is more complex
	for _, action := range rc.Change.Actions {
		if action == tfjson.ActionNoop && rc.Change.Before != nil {
			return true
		}
	}

	return false
}

// ParseFile is a convenience function to parse a plan from a file
func ParseFile(filename string) (*models.PlanResult, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	parser := NewParser()
	return parser.Parse(file)
}

// ParseString is a convenience function to parse a plan from a string
func ParseString(input string) (*models.PlanResult, error) {
	parser := NewParser()
	return parser.Parse(strings.NewReader(input))
}

// Legacy support functions

// ParsePlanFile reads and parses a Terraform JSON plan file (legacy compatibility)
func ParsePlanFile(filename string) (*models.Plan, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan file: %w", err)
	}

	var tfPlan tfjson.Plan
	if err := json.Unmarshal(data, &tfPlan); err != nil {
		return nil, fmt.Errorf("failed to parse plan JSON: %w", err)
	}

	return convertToModel(&tfPlan), nil
}

// convertToModel converts terraform-json Plan to our internal model (legacy compatibility)
func convertToModel(tfPlan *tfjson.Plan) *models.Plan {
	plan := &models.Plan{
		FormatVersion:    tfPlan.FormatVersion,
		TerraformVersion: tfPlan.TerraformVersion,
		Resources:        make([]models.Resource, 0),
		OutputChanges:    make([]models.OutputChange, 0),
	}

	// Convert resource changes
	for _, rc := range tfPlan.ResourceChanges {
		if rc.Change == nil {
			continue
		}

		resource := models.Resource{
			Address:      rc.Address,
			Type:         rc.Type,
			Name:         rc.Name,
			Mode:         string(rc.Mode),
			ProviderName: rc.ProviderName,
			Change: models.Change{
				Actions: make([]string, len(rc.Change.Actions)),
			},
		}

		for i, action := range rc.Change.Actions {
			resource.Change.Actions[i] = string(action)
		}

		plan.Resources = append(plan.Resources, resource)
	}

	return plan
}
