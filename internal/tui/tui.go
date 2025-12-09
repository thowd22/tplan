package tui

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yourusername/tplan/internal/models"
)

// ViewMode represents different view tabs
type ViewMode int

const (
	ViewChanges ViewMode = iota
	ViewErrors
	ViewWarnings
)

// TreeNode represents a node in the hierarchical tree view
type TreeNode struct {
	Resource     models.ResourceChange
	Expanded     bool
	Children     []*TreeNode
	Level        int
	RenderedLines int // Number of lines this node takes when rendered (including expanded details)
}

// Model is the Bubble Tea model for the TUI
type Model struct {
	plan         *models.PlanResult
	nodes        []*TreeNode
	cursor       int
	viewMode     ViewMode
	viewportTop  int
	viewportSize int
	width        int
	height       int
}

// Styles for the TUI
var (
	// Action colors - text colors based on terraform action
	createStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true) // Green for creates
	updateStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true) // Yellow for updates
	deleteStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)  // Red for deletes
	replaceStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true) // Blue for replaces
	noopStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))            // White for no changes

	// UI element styles
	selectedBgStyle = lipgloss.NewStyle().Background(lipgloss.Color("62")) // Just background, no foreground override
	summaryStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62")).Padding(0, 1).MarginBottom(1)
	tabActiveStyle  = lipgloss.NewStyle().Background(lipgloss.Color("62")).Foreground(lipgloss.Color("15")).Padding(0, 2).Bold(true)
	tabStyle        = lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("250")).Padding(0, 2)
	helpStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true)
	treeLineStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	attributeStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	valueAddStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	valueRemStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
)

// NewModel creates a new TUI model
func NewModel(plan *models.PlanResult) Model {
	nodes := buildTreeNodes(plan.Resources)
	return Model{
		plan:         plan,
		nodes:        nodes,
		cursor:       0,
		viewMode:     ViewChanges,
		viewportTop:  0,
		viewportSize: 20, // Will be updated on window size
		width:        80,
		height:       24,
	}
}

// buildTreeNodes converts resources into a hierarchical tree structure with grouping
func buildTreeNodes(resources []models.ResourceChange) []*TreeNode {
	// Filter out resources with no changes (no-op)
	// Only show resources that are actually changing
	changingResources := make([]models.ResourceChange, 0)
	for _, res := range resources {
		if res.Action != models.ActionNoOp {
			changingResources = append(changingResources, res)
		}
	}

	// Group resources by module
	moduleGroups := make(map[string][]models.ResourceChange)

	for _, res := range changingResources {
		module := res.Module
		if module == "" {
			module = "root" // Root module resources
		}
		moduleGroups[module] = append(moduleGroups[module], res)
	}

	// Sort module names for consistent ordering
	moduleNames := make([]string, 0, len(moduleGroups))
	for moduleName := range moduleGroups {
		moduleNames = append(moduleNames, moduleName)
	}
	sort.Strings(moduleNames)

	// Build tree nodes
	nodes := make([]*TreeNode, 0)

	for _, moduleName := range moduleNames {
		moduleResources := moduleGroups[moduleName]

		// Sort resources within module by address
		sort.Slice(moduleResources, func(i, j int) bool {
			return moduleResources[i].Address < moduleResources[j].Address
		})

		// Special handling for root module - group by file
		if moduleName == "root" {
			// Group root resources by file
			fileGroups := make(map[string][]models.ResourceChange)
			ungroupedResources := make([]models.ResourceChange, 0)

			// First pass: group resources by file
			for _, res := range moduleResources {
				fileName := getResourceFileName(res)
				if fileName == "unknown.tf" {
					// Don't group resources we can't find files for yet
					ungroupedResources = append(ungroupedResources, res)
				} else {
					fileGroups[fileName] = append(fileGroups[fileName], res)
				}
			}

			// Second pass: try to group ungrouped deleted resources with their replacements
			remainingUngrouped := make([]models.ResourceChange, 0)
			for _, res := range ungroupedResources {
				// Only try to relocate deleted resources
				if res.Action == models.ActionDelete {
					// Look for a create operation with the same type and index
					targetFile := findReplacementFile(res, moduleResources)
					if targetFile != "" {
						// Group this deleted resource with its replacement
						fileGroups[targetFile] = append(fileGroups[targetFile], res)
					} else {
						remainingUngrouped = append(remainingUngrouped, res)
					}
				} else {
					remainingUngrouped = append(remainingUngrouped, res)
				}
			}
			ungroupedResources = remainingUngrouped

			// Sort file names
			fileNames := make([]string, 0, len(fileGroups))
			for fileName := range fileGroups {
				fileNames = append(fileNames, fileName)
			}
			sort.Strings(fileNames)

			// Create file group nodes
			for _, fileName := range fileNames {
				fileResources := fileGroups[fileName]

				// If only one file in root and no ungrouped resources, don't create a grouping node
				if len(fileGroups) == 1 && len(ungroupedResources) == 0 {
					for _, res := range fileResources {
						node := &TreeNode{
							Resource: res,
							Expanded: false,
							Children: []*TreeNode{},
							Level:    0,
						}
						nodes = append(nodes, node)
					}
				} else {
					// Create a file group node
					if len(fileResources) > 0 {
						firstRes := fileResources[0]
						fileNode := &TreeNode{
							Resource: models.ResourceChange{
								Address:      fileName,
								Type:         "file",
								Name:         fileName,
								Module:       "root",
								Mode:         "file",
								ProviderName: firstRes.ProviderName,
								Action:       models.ActionNoOp, // File nodes are just grouping, not actions
								Change: models.Change{
									Actions: []string{"no-op"},
								},
							},
							Expanded: false,
							Children: make([]*TreeNode, 0),
							Level:    0,
						}

						// Add all resources in this file as children
						for _, res := range fileResources {
							childNode := &TreeNode{
								Resource: res,
								Expanded: false,
								Children: []*TreeNode{},
								Level:    1,
							}
							fileNode.Children = append(fileNode.Children, childNode)
						}

						nodes = append(nodes, fileNode)
					}
				}
			}

			// Add ungrouped resources at the end (no file grouping)
			for _, res := range ungroupedResources {
				node := &TreeNode{
					Resource: res,
					Expanded: false,
					Children: []*TreeNode{},
					Level:    0,
				}
				nodes = append(nodes, node)
			}
		} else {
			// Create a module group node for non-root modules
			if len(moduleResources) > 0 {
				firstRes := moduleResources[0]
				moduleNode := &TreeNode{
					Resource: models.ResourceChange{
						Address:      moduleName,
						Type:         "module",
						Name:         moduleName,
						Module:       moduleName,
						Mode:         "module",
						ProviderName: firstRes.ProviderName,
						Action:       models.ActionNoOp, // Module nodes are just grouping, not actions
						Change: models.Change{
							Actions: []string{"no-op"},
						},
					},
					Expanded: false,
					Children: make([]*TreeNode, 0),
					Level:    0,
				}

				// Add all resources in this module as children
				for _, res := range moduleResources {
					childNode := &TreeNode{
						Resource: res,
						Expanded: false,
						Children: []*TreeNode{},
						Level:    1,
					}
					moduleNode.Children = append(moduleNode.Children, childNode)
				}

				nodes = append(nodes, moduleNode)
			}
		}
	}

	return nodes
}

// getResourceFileName extracts the file name from a resource
func getResourceFileName(res models.ResourceChange) string {
	// If drift info is available, use the file path
	if res.DriftInfo != nil && res.DriftInfo.FilePath != "" {
		// Extract just the filename from the path
		parts := strings.Split(res.DriftInfo.FilePath, "/")
		return parts[len(parts)-1]
	}

	// Fallback: return "unknown.tf" if no file info available
	return "unknown.tf"
}

// findReplacementFile finds the file for a deleted resource by looking for a create operation
// with the same resource type and index (likely a renamed resource)
func findReplacementFile(deletedRes models.ResourceChange, allResources []models.ResourceChange) string {
	// Extract the index from the deleted resource
	deletedIndex := deletedRes.Index

	// Look for a create operation with the same type and index
	for _, res := range allResources {
		if res.Action == models.ActionCreate && res.Type == deletedRes.Type {
			// Check if the index matches
			if indexMatches(res.Index, deletedIndex) {
				// Found a potential replacement - get its file
				fileName := getResourceFileName(res)
				if fileName != "unknown.tf" {
					return fileName
				}
			}
		}
	}

	return ""
}

// indexMatches checks if two resource indices match
func indexMatches(idx1, idx2 interface{}) bool {
	// Handle nil cases
	if idx1 == nil && idx2 == nil {
		return true
	}
	if idx1 == nil || idx2 == nil {
		return false
	}

	// Compare as strings to handle both int and string indices
	return fmt.Sprintf("%v", idx1) == fmt.Sprintf("%v", idx2)
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewportSize = msg.Height - 10 // Account for header, summary, tabs, and help

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m = m.adjustViewport()
			}

		case "down", "j":
			visibleNodes := m.getVisibleNodes()
			if m.cursor < len(visibleNodes)-1 {
				m.cursor++
				m = m.adjustViewport()
			}

		case "enter", " ":
			visibleNodes := m.getVisibleNodes()
			if m.cursor < len(visibleNodes) {
				visibleNodes[m.cursor].Expanded = !visibleNodes[m.cursor].Expanded
				// Adjust viewport after expanding/collapsing to ensure content is visible
				m = m.adjustViewport()
			}

		case "tab":
			m.viewMode = (m.viewMode + 1) % 3
			m.cursor = 0
			m.viewportTop = 0

		case "shift+tab":
			if m.viewMode == 0 {
				m.viewMode = 2
			} else {
				m.viewMode--
			}
			m.cursor = 0
			m.viewportTop = 0

		case "g":
			// Go to top
			m.cursor = 0
			m.viewportTop = 0

		case "G":
			// Go to bottom
			visibleNodes := m.getVisibleNodes()
			m.cursor = len(visibleNodes) - 1
			m = m.adjustViewport()

		case "e":
			// Expand all
			for _, node := range m.nodes {
				node.Expanded = true
			}

		case "c":
			// Collapse all
			for _, node := range m.nodes {
				node.Expanded = false
			}
		}
	}

	return m, nil
}

// View renders the UI
func (m Model) View() string {
	var b strings.Builder

	// Render tabs
	b.WriteString(m.renderTabs())
	b.WriteString("\n\n")

	// Render summary
	b.WriteString(m.renderSummary())
	b.WriteString("\n")

	// Render content based on view mode
	switch m.viewMode {
	case ViewChanges:
		b.WriteString(m.renderChangesView())
	case ViewErrors:
		b.WriteString(m.renderErrorsView())
	case ViewWarnings:
		b.WriteString(m.renderWarningsView())
	}

	// Render help
	b.WriteString("\n")
	b.WriteString(m.renderHelp())

	return b.String()
}

// renderTabs renders the tab bar
func (m Model) renderTabs() string {
	tabs := []string{}

	changeCount := len(m.plan.Resources)
	errorCount := len(m.plan.Errors)
	warningCount := len(m.plan.Warnings)

	changesTab := fmt.Sprintf("Changes (%d)", changeCount)
	errorsTab := fmt.Sprintf("Errors (%d)", errorCount)
	warningsTab := fmt.Sprintf("Warnings (%d)", warningCount)

	if m.viewMode == ViewChanges {
		tabs = append(tabs, tabActiveStyle.Render(changesTab))
	} else {
		tabs = append(tabs, tabStyle.Render(changesTab))
	}

	if m.viewMode == ViewErrors {
		tabs = append(tabs, tabActiveStyle.Render(errorsTab))
	} else {
		tabs = append(tabs, tabStyle.Render(errorsTab))
	}

	if m.viewMode == ViewWarnings {
		tabs = append(tabs, tabActiveStyle.Render(warningsTab))
	} else {
		tabs = append(tabs, tabStyle.Render(warningsTab))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
}

// renderSummary renders the summary section
func (m Model) renderSummary() string {
	summary := fmt.Sprintf(
		"%s %d  %s %d  %s %d  %s %d  │  Version: %s",
		createStyle.Render("✚ Create:"),
		m.plan.Summary.ToCreate,
		updateStyle.Render("~ Update:"),
		m.plan.Summary.ToUpdate,
		deleteStyle.Render("✖ Delete:"),
		m.plan.Summary.ToDelete,
		replaceStyle.Render("⟳ Replace:"),
		m.plan.Summary.ToReplace,
		m.plan.TerraformVersion,
	)

	return summaryStyle.Render(summary)
}

// renderChangesView renders the changes tree view
func (m Model) renderChangesView() string {
	var b strings.Builder

	visibleNodes := m.getVisibleNodes()
	if len(visibleNodes) == 0 {
		return helpStyle.Render("No changes to display")
	}

	// Build all lines first, then apply viewport
	var allLines []string
	for i, node := range visibleNodes {
		isSelected := (i == m.cursor)

		// Render the node line
		line := m.renderTreeNode(node, isSelected)
		allLines = append(allLines, line)

		// Render expanded details if applicable
		if node.Expanded && (node.Level == 0 || node.Resource.Type != "module") {
			detailsContent := m.renderResourceDetails(node)
			if detailsContent != "" {
				// Split details into individual lines
				detailLines := strings.Split(strings.TrimSuffix(detailsContent, "\n"), "\n")
				allLines = append(allLines, detailLines...)
			}
		}
	}

	// Apply viewport - only show lines within the viewport range
	totalLines := len(allLines)
	viewportEnd := m.viewportTop + m.viewportSize
	if viewportEnd > totalLines {
		viewportEnd = totalLines
	}

	displayedNodes := 0
	for i := m.viewportTop; i < viewportEnd; i++ {
		b.WriteString(allLines[i])
		b.WriteString("\n")
		displayedNodes++
	}

	// Scroll indicator
	if totalLines > m.viewportSize {
		scrollInfo := fmt.Sprintf("\n%s [lines %d-%d of %d]",
			helpStyle.Render("Scroll:"),
			m.viewportTop+1,
			viewportEnd,
			totalLines,
		)
		b.WriteString(scrollInfo)
	}

	return b.String()
}

// getTotalRenderedLines calculates the total number of lines that would be rendered
func (m Model) getTotalRenderedLines() int {
	visibleNodes := m.getVisibleNodes()
	totalLines := 0

	for _, node := range visibleNodes {
		totalLines++ // The node line itself
		if node.Expanded && (node.Level == 0 || node.Resource.Type != "module") {
			details := m.renderResourceDetails(node)
			if details != "" {
				totalLines += strings.Count(details, "\n")
			}
		}
	}

	return totalLines
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// renderTreeNode renders a single tree node
func (m Model) renderTreeNode(node *TreeNode, selected bool) string {
	// Tree structure
	prefix := strings.Repeat("  ", node.Level)

	// Expand icon - only show for nodes with children or expandable content
	expandIcon := " "
	hasExpandableContent := len(node.Children) > 0 || (node.Resource.Type != "module" && node.Resource.Type != "file" && node.Level == 0)
	if hasExpandableContent {
		if node.Expanded {
			expandIcon = "▾"
		} else {
			expandIcon = "▸"
		}
	}

	// Special handling for module nodes
	if node.Resource.Type == "module" {
		moduleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true) // Cyan
		childInfo := fmt.Sprintf(" [%d resources]", len(node.Children))

		if selected {
			// Apply background only, preserve text colors
			selector := selectedBgStyle.Render("❯ ")
			prefixText := selectedBgStyle.Render(prefix)
			expandText := selectedBgStyle.Render(expandIcon + " ")
			iconAndName := selectedBgStyle.Copy().Inherit(moduleStyle).Render("📦 " + node.Resource.Address)
			childInfoStyled := selectedBgStyle.Render(childInfo)
			return selector + prefixText + expandText + iconAndName + childInfoStyled
		} else {
			selector := treeLineStyle.Render("  ")
			prefixText := treeLineStyle.Render(prefix)
			expandText := treeLineStyle.Render(expandIcon + " ")
			iconAndName := moduleStyle.Render("📦 " + node.Resource.Address)
			childInfoStyled := treeLineStyle.Render(childInfo)
			return selector + prefixText + expandText + iconAndName + childInfoStyled
		}
	}

	// Special handling for file nodes
	if node.Resource.Type == "file" {
		fileStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15")) // White
		childInfo := fmt.Sprintf(" [%d resources]", len(node.Children))

		if selected {
			// Apply background only, preserve text colors
			selector := selectedBgStyle.Render("❯ ")
			prefixText := selectedBgStyle.Render(prefix)
			expandText := selectedBgStyle.Render(expandIcon + " ")
			iconAndName := selectedBgStyle.Copy().Inherit(fileStyle).Render("📄 " + node.Resource.Address)
			childInfoStyled := selectedBgStyle.Render(childInfo)
			return selector + prefixText + expandText + iconAndName + childInfoStyled
		} else {
			selector := treeLineStyle.Render("  ")
			prefixText := treeLineStyle.Render(prefix)
			expandText := treeLineStyle.Render(expandIcon + " ")
			iconAndName := fileStyle.Render("📄 " + node.Resource.Address)
			childInfoStyled := treeLineStyle.Render(childInfo)
			return selector + prefixText + expandText + iconAndName + childInfoStyled
		}
	}

	// Action icon and style for regular resources
	// Use the Action field from the resource, not Change.Actions
	action := string(node.Resource.Action)
	actionIcon, actionStyle := getActionIconAndStyle(action)

	// Build the line with selection indicator
	address := node.Resource.Address

	// Add child count for parent nodes (dependency-based grouping, if any)
	childInfo := ""
	if node.Level == 0 && len(node.Children) > 0 && node.Resource.Type != "module" && node.Resource.Type != "file" {
		childInfo = fmt.Sprintf(" (%d related)", len(node.Children))
	}

	if selected {
		// Apply background only, preserve action text colors
		selector := selectedBgStyle.Render("❯ ")
		prefixText := selectedBgStyle.Render(prefix)
		expandText := selectedBgStyle.Render(expandIcon + " ")
		iconAndName := selectedBgStyle.Copy().Inherit(actionStyle).Render(actionIcon + " " + address)
		childInfoStyled := selectedBgStyle.Render(childInfo)
		return selector + prefixText + expandText + iconAndName + childInfoStyled
	} else {
		// Normal rendering with colored resource text based on action
		selector := treeLineStyle.Render("  ")
		prefixText := treeLineStyle.Render(prefix)
		expandText := treeLineStyle.Render(expandIcon + " ")

		// Use action style for both icon AND address text
		iconAndName := actionStyle.Render(actionIcon + " " + address)
		childInfoStyled := treeLineStyle.Render(childInfo)

		return selector + prefixText + expandText + iconAndName + childInfoStyled
	}
}

// renderResourceDetails renders expanded resource details
func (m Model) renderResourceDetails(node *TreeNode) string {
	var b strings.Builder
	// Indent details to align with resource name (2 spaces for selection indicator + 2 for content)
	indent := "    "

	res := node.Resource

	// Get the action style for this resource
	action := string(res.Action)
	_, actionStyle := getActionIconAndStyle(action)

	// Resource metadata - use action color with aligned labels
	b.WriteString(fmt.Sprintf("%s", indent))
	b.WriteString(actionStyle.Render(fmt.Sprintf("%-5s %s\n", "Type:", res.Type)))
	b.WriteString(fmt.Sprintf("%s", indent))
	b.WriteString(actionStyle.Render(fmt.Sprintf("%-5s %s\n", "Provider:", res.ProviderName)))
	b.WriteString(fmt.Sprintf("%s", indent))
	b.WriteString(actionStyle.Render(fmt.Sprintf("%-5s %s\n", "Mode:", res.Mode)))

	// Show file and git information if available
	if res.DriftInfo != nil && res.DriftInfo.FilePath != "" {
		b.WriteString("\n")

		// Always show file path - use action color for header
		b.WriteString(fmt.Sprintf("%s", indent))
		b.WriteString(actionStyle.Render("File Information:\n"))

		// Show git info if available (IsValid checks for full git info)
		if res.DriftInfo.IsValid() {
			// File and Commit on the same line
			b.WriteString(fmt.Sprintf("%s  ", indent))
			b.WriteString(actionStyle.Render(fmt.Sprintf("%-5s %-80s %-7s %s\n",
				"File:", res.DriftInfo.FilePath,
				"Commit:", res.DriftInfo.ShortCommitID())))
			b.WriteString(fmt.Sprintf("%s  ", indent))
			b.WriteString(actionStyle.Render(fmt.Sprintf("%-5s %s\n", "Branch:", res.DriftInfo.BranchName)))
			b.WriteString(fmt.Sprintf("%s  ", indent))
			b.WriteString(actionStyle.Render(fmt.Sprintf("%-5s %s <%s>\n", "Author:", res.DriftInfo.AuthorName, res.DriftInfo.AuthorEmail)))
			b.WriteString(fmt.Sprintf("%s  ", indent))
			b.WriteString(actionStyle.Render(fmt.Sprintf("%-5s %s\n", "Date:", res.DriftInfo.CommitDate.Format("2006-01-02 15:04:05"))))
		} else {
			// If no git info, just show the file path
			b.WriteString(fmt.Sprintf("%s  ", indent))
			b.WriteString(actionStyle.Render(fmt.Sprintf("%-5s %s\n", "File:", res.DriftInfo.FilePath)))
		}

		if res.DriftInfo.HasUncommittedChanges {
			warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11")) // Yellow
			b.WriteString(fmt.Sprintf("%s  ", indent))
			b.WriteString(warnStyle.Render(fmt.Sprintf("%-5s Has uncommitted changes\n", "Status:")))
		}
		b.WriteString("\n")
	}

	// Show attribute changes
	if action == "create" {
		b.WriteString(m.renderAttributes(indent, res.Change.After, "  ", actionStyle))
	} else if action == "delete" {
		b.WriteString(m.renderAttributes(indent, res.Change.Before, "  ", actionStyle))
	} else if action == "update" || action == "replace" {
		b.WriteString(m.renderAttributeDiff(indent, res.Change.Before, res.Change.After))
	}

	// Add a blank line after expanded details to separate from next resource
	b.WriteString("\n")

	return b.String()
}

// renderAttributes renders attribute map with indentation
func (m Model) renderAttributes(baseIndent string, attrs map[string]interface{}, subIndent string, actionStyle lipgloss.Style) string {
	var b strings.Builder

	// Sort keys to ensure consistent ordering
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Show all attributes with proper nesting
	for _, k := range keys {
		v := attrs[k]
		m.renderValue(&b, baseIndent, k, v, actionStyle, 0)
	}

	return b.String()
}

// renderValue renders a single value with proper handling of nested structures
func (m Model) renderValue(b *strings.Builder, indent string, key string, value interface{}, style lipgloss.Style, depth int) {
	// Limit nesting depth to prevent excessive output
	if depth > 5 {
		b.WriteString(style.Render(fmt.Sprintf("%s%s = <deeply nested>\n", indent, key)))
		return
	}

	switch v := value.(type) {
	case map[string]interface{}:
		// Nested object
		if len(v) == 0 {
			b.WriteString(style.Render(fmt.Sprintf("%s%s = {}\n", indent, key)))
		} else {
			b.WriteString(style.Render(fmt.Sprintf("%s%s = {\n", indent, key)))
			// Sort nested keys
			nestedKeys := make([]string, 0, len(v))
			for k := range v {
				nestedKeys = append(nestedKeys, k)
			}
			sort.Strings(nestedKeys)
			for _, nk := range nestedKeys {
				m.renderValue(b, indent+"  ", nk, v[nk], style, depth+1)
			}
			b.WriteString(style.Render(fmt.Sprintf("%s}\n", indent)))
		}
	case []interface{}:
		// Array
		if len(v) == 0 {
			b.WriteString(style.Render(fmt.Sprintf("%s%s = []\n", indent, key)))
		} else {
			b.WriteString(style.Render(fmt.Sprintf("%s%s = [\n", indent, key)))
			for i, item := range v {
				m.renderValue(b, indent+"  ", fmt.Sprintf("[%d]", i), item, style, depth+1)
			}
			b.WriteString(style.Render(fmt.Sprintf("%s]\n", indent)))
		}
	case string:
		// String value - show with quotes
		b.WriteString(style.Render(fmt.Sprintf("%s%s = %q\n", indent, key, v)))
	case nil:
		// Null value
		b.WriteString(style.Render(fmt.Sprintf("%s%s = null\n", indent, key)))
	case bool:
		// Boolean value
		b.WriteString(style.Render(fmt.Sprintf("%s%s = %t\n", indent, key, v)))
	case float64:
		// Number - check if it's an integer
		if v == float64(int64(v)) {
			b.WriteString(style.Render(fmt.Sprintf("%s%s = %d\n", indent, key, int64(v))))
		} else {
			b.WriteString(style.Render(fmt.Sprintf("%s%s = %g\n", indent, key, v)))
		}
	default:
		// Fallback for other types
		valueStr := fmt.Sprintf("%v", v)
		if len(valueStr) > 100 {
			valueStr = valueStr[:97] + "..."
		}
		b.WriteString(style.Render(fmt.Sprintf("%s%s = %s\n", indent, key, valueStr)))
	}
}

// renderAttributeDiff renders before/after attribute differences
func (m Model) renderAttributeDiff(baseIndent string, before, after map[string]interface{}) string {
	var b strings.Builder

	// Collect all keys from both maps and sort them
	keySet := make(map[string]bool)
	for k := range before {
		keySet[k] = true
	}
	for k := range after {
		keySet[k] = true
	}

	keys := make([]string, 0, len(keySet))
	for k := range keySet {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Process all attributes with proper nesting
	for _, k := range keys {
		afterVal, existsAfter := after[k]
		beforeVal, existsBefore := before[k]

		if !existsBefore && existsAfter {
			// New attribute - show with + prefix
			m.renderDiffValue(&b, baseIndent, "+", k, afterVal, valueAddStyle, 0)
		} else if existsBefore && !existsAfter {
			// Removed attribute - show with - prefix
			m.renderDiffValue(&b, baseIndent, "-", k, beforeVal, valueRemStyle, 0)
		} else {
			// Check if changed
			m.renderDiffComparison(&b, baseIndent, k, beforeVal, afterVal, 0)
		}
	}

	return b.String()
}

// renderDiffValue renders a value in a diff context (added or removed)
func (m Model) renderDiffValue(b *strings.Builder, indent string, prefix string, key string, value interface{}, style lipgloss.Style, depth int) {
	if depth > 5 {
		b.WriteString(attributeStyle.Render(fmt.Sprintf("%s  %s %s = <deeply nested>\n", indent, prefix, key)))
		return
	}

	switch v := value.(type) {
	case map[string]interface{}:
		if len(v) == 0 {
			b.WriteString(attributeStyle.Render(fmt.Sprintf("%s  %s %s = ", indent, prefix, key)))
			b.WriteString(style.Render("{}"))
			b.WriteString("\n")
		} else {
			b.WriteString(attributeStyle.Render(fmt.Sprintf("%s  %s %s = {\n", indent, prefix, key)))
			nestedKeys := make([]string, 0, len(v))
			for k := range v {
				nestedKeys = append(nestedKeys, k)
			}
			sort.Strings(nestedKeys)
			for _, nk := range nestedKeys {
				m.renderDiffValue(b, indent+"  ", prefix, nk, v[nk], style, depth+1)
			}
			b.WriteString(attributeStyle.Render(fmt.Sprintf("%s  %s }\n", indent, prefix)))
		}
	case []interface{}:
		if len(v) == 0 {
			b.WriteString(attributeStyle.Render(fmt.Sprintf("%s  %s %s = ", indent, prefix, key)))
			b.WriteString(style.Render("[]"))
			b.WriteString("\n")
		} else {
			b.WriteString(attributeStyle.Render(fmt.Sprintf("%s  %s %s = [\n", indent, prefix, key)))
			for i, item := range v {
				m.renderDiffValue(b, indent+"  ", prefix, fmt.Sprintf("[%d]", i), item, style, depth+1)
			}
			b.WriteString(attributeStyle.Render(fmt.Sprintf("%s  %s ]\n", indent, prefix)))
		}
	case string:
		b.WriteString(attributeStyle.Render(fmt.Sprintf("%s  %s %s = ", indent, prefix, key)))
		b.WriteString(style.Render(fmt.Sprintf("%q", v)))
		b.WriteString("\n")
	case nil:
		b.WriteString(attributeStyle.Render(fmt.Sprintf("%s  %s %s = ", indent, prefix, key)))
		b.WriteString(style.Render("null"))
		b.WriteString("\n")
	case bool:
		b.WriteString(attributeStyle.Render(fmt.Sprintf("%s  %s %s = ", indent, prefix, key)))
		b.WriteString(style.Render(fmt.Sprintf("%t", v)))
		b.WriteString("\n")
	case float64:
		b.WriteString(attributeStyle.Render(fmt.Sprintf("%s  %s %s = ", indent, prefix, key)))
		if v == float64(int64(v)) {
			b.WriteString(style.Render(fmt.Sprintf("%d", int64(v))))
		} else {
			b.WriteString(style.Render(fmt.Sprintf("%g", v)))
		}
		b.WriteString("\n")
	default:
		valueStr := fmt.Sprintf("%v", v)
		if len(valueStr) > 100 {
			valueStr = valueStr[:97] + "..."
		}
		b.WriteString(attributeStyle.Render(fmt.Sprintf("%s  %s %s = ", indent, prefix, key)))
		b.WriteString(style.Render(valueStr))
		b.WriteString("\n")
	}
}

// renderDiffComparison compares before and after values and renders the diff
func (m Model) renderDiffComparison(b *strings.Builder, indent string, key string, before, after interface{}, depth int) {
	if depth > 5 {
		b.WriteString(attributeStyle.Render(fmt.Sprintf("%s  ~ %s = <deeply nested>\n", indent, key)))
		return
	}

	// Deep comparison using JSON-like comparison
	beforeStr := fmt.Sprintf("%v", before)
	afterStr := fmt.Sprintf("%v", after)

	if beforeStr == afterStr {
		// No change
		return
	}

	// Check if both values are strings - might be JSON or long text
	beforeString, beforeIsString := before.(string)
	afterString, afterIsString := after.(string)

	if beforeIsString && afterIsString {
		// For strings longer than 60 chars, show them on separate lines (like terraform plan)
		if len(beforeString) > 60 || len(afterString) > 60 {
			// Render the attribute label without styling the indent
			b.WriteString(indent)
			b.WriteString(attributeStyle.Render(fmt.Sprintf("  ~ %s:\n", key)))

			// Try to pretty-print if it's JSON
			beforeFormatted := m.tryPrettyJSON(beforeString)
			afterFormatted := m.tryPrettyJSON(afterString)

			// Split into lines
			beforeLines := strings.Split(beforeFormatted, "\n")
			afterLines := strings.Split(afterFormatted, "\n")

			// Show side-by-side diff
			m.renderSideBySideDiff(b, indent, beforeLines, afterLines)

			return
		}
	}

	// For short values or non-strings, show inline
	b.WriteString(attributeStyle.Render(fmt.Sprintf("%s  ~ %s: ", indent, key)))

	// Format before value
	switch v := before.(type) {
	case string:
		b.WriteString(valueRemStyle.Render(fmt.Sprintf("%q", v)))
	case nil:
		b.WriteString(valueRemStyle.Render("null"))
	case bool:
		b.WriteString(valueRemStyle.Render(fmt.Sprintf("%t", v)))
	case float64:
		if v == float64(int64(v)) {
			b.WriteString(valueRemStyle.Render(fmt.Sprintf("%d", int64(v))))
		} else {
			b.WriteString(valueRemStyle.Render(fmt.Sprintf("%g", v)))
		}
	default:
		if len(beforeStr) > 60 {
			beforeStr = beforeStr[:57] + "..."
		}
		b.WriteString(valueRemStyle.Render(beforeStr))
	}

	b.WriteString(attributeStyle.Render(" → "))

	// Format after value
	switch v := after.(type) {
	case string:
		b.WriteString(valueAddStyle.Render(fmt.Sprintf("%q", v)))
	case nil:
		b.WriteString(valueAddStyle.Render("null"))
	case bool:
		b.WriteString(valueAddStyle.Render(fmt.Sprintf("%t", v)))
	case float64:
		if v == float64(int64(v)) {
			b.WriteString(valueAddStyle.Render(fmt.Sprintf("%d", int64(v))))
		} else {
			b.WriteString(valueAddStyle.Render(fmt.Sprintf("%g", v)))
		}
	default:
		if len(afterStr) > 60 {
			afterStr = afterStr[:57] + "..."
		}
		b.WriteString(valueAddStyle.Render(afterStr))
	}

	b.WriteString("\n")
}

// wrapString wraps a long string into multiple lines at word boundaries
func (m Model) wrapString(s string, maxLen int) []string {
	if len(s) <= maxLen {
		return []string{s}
	}

	var lines []string
	remaining := s

	for len(remaining) > 0 {
		if len(remaining) <= maxLen {
			lines = append(lines, remaining)
			break
		}

		// Try to break at a reasonable point (space, comma, etc.)
		breakPoint := maxLen
		for i := maxLen; i > maxLen-20 && i > 0; i-- {
			if remaining[i] == ' ' || remaining[i] == ',' || remaining[i] == ';' {
				breakPoint = i + 1
				break
			}
		}

		lines = append(lines, remaining[:breakPoint])
		remaining = remaining[breakPoint:]
	}

	return lines
}

// renderErrorsView renders the errors view
func (m Model) renderErrorsView() string {
	if len(m.plan.Errors) == 0 {
		return helpStyle.Render("No errors to display")
	}

	var b strings.Builder
	for i, err := range m.plan.Errors {
		resource := ""
		if err.Resource != "" {
			resource = fmt.Sprintf("[%s] ", err.Resource)
		}

		line := fmt.Sprintf("✖ %s%s", resource, err.Message)

		if i == m.cursor {
			// Full line highlight with selection indicator, preserve red color
			selector := selectedBgStyle.Render("❯ ")
			content := selectedBgStyle.Copy().Inherit(deleteStyle).Render(line)
			b.WriteString(selector + content)
		} else {
			// Normal rendering with spacing for alignment
			b.WriteString(fmt.Sprintf("  %s", deleteStyle.Render(line)))
		}
		b.WriteString("\n")
	}
	return b.String()
}

// renderWarningsView renders the warnings view
func (m Model) renderWarningsView() string {
	if len(m.plan.Warnings) == 0 {
		return helpStyle.Render("No warnings to display")
	}

	var b strings.Builder
	for i, warn := range m.plan.Warnings {
		resource := ""
		if warn.Resource != "" {
			resource = fmt.Sprintf("[%s] ", warn.Resource)
		}

		line := fmt.Sprintf("⚠ %s%s", resource, warn.Message)

		if i == m.cursor {
			// Full line highlight with selection indicator, preserve yellow color
			selector := selectedBgStyle.Render("❯ ")
			content := selectedBgStyle.Copy().Inherit(updateStyle).Render(line)
			b.WriteString(selector + content)
		} else {
			// Normal rendering with spacing for alignment
			b.WriteString(fmt.Sprintf("  %s", updateStyle.Render(line)))
		}
		b.WriteString("\n")
	}
	return b.String()
}

// renderHelp renders the help text
func (m Model) renderHelp() string {
	help := "↑/↓: Navigate  Enter/Space: Expand/Collapse  Tab: Switch View  e: Expand All  c: Collapse All  g/G: Top/Bottom  q: Quit"
	return helpStyle.Render(help)
}

// getVisibleNodes returns all currently visible nodes (considering expand/collapse state)
func (m Model) getVisibleNodes() []*TreeNode {
	visible := make([]*TreeNode, 0)
	for _, node := range m.nodes {
		visible = append(visible, node)
		// If node is expanded, add its children
		if node.Expanded && len(node.Children) > 0 {
			visible = append(visible, node.Children...)
		}
	}
	return visible
}

// adjustViewport adjusts the viewport to keep the cursor visible
func (m Model) adjustViewport() Model {
	visibleNodes := m.getVisibleNodes()
	if len(visibleNodes) == 0 {
		m.viewportTop = 0
		return m
	}

	// Ensure cursor is within bounds
	if m.cursor >= len(visibleNodes) {
		m.cursor = len(visibleNodes) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}

	// Calculate the line position of the cursor (start of the cursor node)
	cursorLineStart := 0
	for i := 0; i < m.cursor && i < len(visibleNodes); i++ {
		node := visibleNodes[i]
		cursorLineStart++ // The node line itself
		if node.Expanded && (node.Level == 0 || node.Resource.Type != "module") {
			details := m.renderResourceDetails(node)
			if details != "" {
				cursorLineStart += strings.Count(details, "\n")
			}
		}
	}

	// Calculate the total lines for the current cursor node (including expanded content)
	currentNode := visibleNodes[m.cursor]
	currentNodeLines := 1 // The node line itself
	if currentNode.Expanded && (currentNode.Level == 0 || currentNode.Resource.Type != "module") {
		details := m.renderResourceDetails(currentNode)
		if details != "" {
			currentNodeLines += strings.Count(details, "\n")
		}
	}
	cursorLineEnd := cursorLineStart + currentNodeLines - 1

	// Adjust viewport to keep cursor visible
	if cursorLineStart < m.viewportTop {
		// Cursor start is above viewport, scroll up to show the start
		m.viewportTop = cursorLineStart
	} else if cursorLineEnd >= m.viewportTop+m.viewportSize {
		// Cursor end is below viewport, scroll down to show as much as possible
		// Try to show the entire node if it fits, otherwise show from the start
		if currentNodeLines <= m.viewportSize {
			// Node fits in viewport, position it at the bottom
			m.viewportTop = cursorLineEnd - m.viewportSize + 1
		} else {
			// Node is larger than viewport, show from the start
			m.viewportTop = cursorLineStart
		}
	}

	// Ensure viewport doesn't go negative
	if m.viewportTop < 0 {
		m.viewportTop = 0
	}

	return m
}

// countActions counts different action types in the plan
func (m Model) countActions() (creates, updates, deletes, replaces int) {
	for _, res := range m.plan.Resources {
		action := string(res.Action)
		switch action {
		case "create":
			creates++
		case "update":
			updates++
		case "delete":
			deletes++
		case "replace":
			replaces++
		}
	}
	return
}

// getActionIconAndStyle returns the icon and style for an action
func getActionIconAndStyle(action string) (string, lipgloss.Style) {
	switch action {
	case "create":
		return "✚", createStyle
	case "update":
		return "~", updateStyle
	case "delete":
		return "✖", deleteStyle
	case "replace":
		return "⟳", replaceStyle
	default:
		return "•", noopStyle
	}
}

// renderSideBySideDiff renders two sets of lines side-by-side
func (m Model) renderSideBySideDiff(b *strings.Builder, indent string, beforeLines, afterLines []string) {
	// First pass: find the longest line content (just the line itself, not including our prefix)
	maxLineWidth := 0
	for _, line := range beforeLines {
		if len(line) > maxLineWidth {
			maxLineWidth = len(line)
		}
	}

	maxLines := len(beforeLines)
	if len(afterLines) > maxLines {
		maxLines = len(afterLines)
	}

	// Second pass: render each line with exact padding
	for i := 0; i < maxLines; i++ {
		var beforeLine, afterLine string

		if i < len(beforeLines) {
			beforeLine = beforeLines[i]
		}
		if i < len(afterLines) {
			afterLine = afterLines[i]
		}

		// Calculate how much padding we need after this line's content to reach maxLineWidth
		paddingNeeded := maxLineWidth - len(beforeLine)
		if paddingNeeded < 0 {
			paddingNeeded = 0
		}

		// Render the line: indent + content + padding + separator + after content
		// Note: We don't add extra spacing because the JSON already has its own indentation
		b.WriteString(indent)
		b.WriteString(valueRemStyle.Render(beforeLine))
		b.WriteString(strings.Repeat(" ", paddingNeeded))
		b.WriteString(" │ ")
		b.WriteString(valueAddStyle.Render(afterLine))
		b.WriteString("\n")
	}
}

// tryPrettyJSON attempts to parse and pretty-print JSON, returns original string if not JSON
func (m Model) tryPrettyJSON(s string) string {
	// Try to parse as JSON
	var jsonData interface{}
	if err := json.Unmarshal([]byte(s), &jsonData); err != nil {
		// Not valid JSON, return original with wrapping
		return m.wrapStringSimple(s, 100)
	}

	// Pretty-print the JSON with 2-space indentation
	prettyJSON, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		// Failed to marshal, return original
		return m.wrapStringSimple(s, 100)
	}

	return string(prettyJSON)
}

// wrapStringSimple wraps a string at a maximum length without JSON parsing
func (m Model) wrapStringSimple(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	var result strings.Builder
	remaining := s

	for len(remaining) > 0 {
		if len(remaining) <= maxLen {
			result.WriteString(remaining)
			break
		}

		// Try to break at a reasonable point (space, comma, etc.)
		breakPoint := maxLen
		for i := maxLen; i > maxLen-20 && i > 0; i-- {
			if remaining[i] == ' ' || remaining[i] == ',' || remaining[i] == ';' {
				breakPoint = i + 1
				break
			}
		}

		result.WriteString(remaining[:breakPoint])
		result.WriteString("\n")
		remaining = remaining[breakPoint:]
	}

	return result.String()
}

// Run starts the TUI application
func Run(plan *models.PlanResult) error {
	p := tea.NewProgram(NewModel(plan), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
