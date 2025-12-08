package tui

import (
	"fmt"
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
	Resource models.ResourceChange
	Expanded bool
	Children []*TreeNode
	Level    int
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
	// Action colors
	createStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true) // Green
	updateStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true) // Yellow
	deleteStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)  // Red
	replaceStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true) // Blue
	noopStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))             // Gray

	// UI element styles
	selectedStyle  = lipgloss.NewStyle().Background(lipgloss.Color("62")).Foreground(lipgloss.Color("15")).Bold(true) // Bright highlight
	summaryStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62")).Padding(0, 1).MarginBottom(1)
	tabActiveStyle = lipgloss.NewStyle().Background(lipgloss.Color("62")).Foreground(lipgloss.Color("15")).Padding(0, 2).Bold(true)
	tabStyle       = lipgloss.NewStyle().Background(lipgloss.Color("236")).Foreground(lipgloss.Color("250")).Padding(0, 2)
	helpStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true)
	treeLineStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	attributeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	valueAddStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	valueRemStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
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

// buildTreeNodes converts resources into a hierarchical tree structure
func buildTreeNodes(resources []models.ResourceChange) []*TreeNode {
	nodes := make([]*TreeNode, 0, len(resources))
	for _, res := range resources {
		node := &TreeNode{
			Resource: res,
			Expanded: false,
			Children: []*TreeNode{},
			Level:    0,
		}
		nodes = append(nodes, node)
	}
	return nodes
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
				m.adjustViewport()
			}

		case "down", "j":
			visibleNodes := m.getVisibleNodes()
			if m.cursor < len(visibleNodes)-1 {
				m.cursor++
				m.adjustViewport()
			}

		case "enter", " ":
			visibleNodes := m.getVisibleNodes()
			if m.cursor < len(visibleNodes) {
				visibleNodes[m.cursor].Expanded = !visibleNodes[m.cursor].Expanded
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
			m.adjustViewport()

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

	// Calculate viewport bounds
	start := m.viewportTop
	end := m.viewportTop + m.viewportSize
	if end > len(visibleNodes) {
		end = len(visibleNodes)
	}

	for i := start; i < end; i++ {
		node := visibleNodes[i]
		line := m.renderTreeNode(node, i == m.cursor)
		b.WriteString(line)
		b.WriteString("\n")

		// Render expanded details
		if node.Expanded {
			details := m.renderResourceDetails(node)
			b.WriteString(details)
		}
	}

	// Scroll indicator
	if len(visibleNodes) > m.viewportSize {
		scrollInfo := fmt.Sprintf("\n%s [%d-%d of %d]",
			helpStyle.Render("▲▼ Scroll:"),
			start+1,
			end,
			len(visibleNodes),
		)
		b.WriteString(scrollInfo)
	}

	return b.String()
}

// renderTreeNode renders a single tree node
func (m Model) renderTreeNode(node *TreeNode, selected bool) string {
	// Tree structure
	prefix := strings.Repeat("  ", node.Level)
	expandIcon := "▸"
	if node.Expanded {
		expandIcon = "▾"
	}

	// Action icon and style
	action := getAction(node.Resource.Change.Actions)
	actionIcon, actionStyle := getActionIconAndStyle(action)

	// Build the line with selection indicator
	address := node.Resource.Address

	if selected {
		// Build full text line first, then apply selection style to entire line
		fullLine := fmt.Sprintf("❯ %s%s %s %s", prefix, expandIcon, actionIcon, address)
		// Apply selection background to full line, but keep width constraint
		return selectedStyle.Width(m.width - 2).Render(fullLine)
	} else {
		// Normal rendering with colored resource text
		line := fmt.Sprintf("  %s%s %s %s",
			prefix,
			expandIcon,
			actionIcon,
			address,
		)
		// Apply action color to the entire line for consistency
		return actionStyle.Render(line)
	}
}

// renderResourceDetails renders expanded resource details
func (m Model) renderResourceDetails(node *TreeNode) string {
	var b strings.Builder
	indent := strings.Repeat("  ", node.Level+2)

	res := node.Resource

	// Resource metadata
	b.WriteString(attributeStyle.Render(fmt.Sprintf("%sType: %s\n", indent, res.Type)))
	b.WriteString(attributeStyle.Render(fmt.Sprintf("%sProvider: %s\n", indent, res.ProviderName)))
	b.WriteString(attributeStyle.Render(fmt.Sprintf("%sMode: %s\n", indent, res.Mode)))

	// Show drift information if available
	if res.DriftInfo != nil && res.DriftInfo.IsValid() {
		driftStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true) // Cyan
		b.WriteString("\n")
		b.WriteString(driftStyle.Render(fmt.Sprintf("%sGit Information:\n", indent)))
		b.WriteString(attributeStyle.Render(fmt.Sprintf("%s  File: %s\n", indent, res.DriftInfo.FilePath)))
		b.WriteString(attributeStyle.Render(fmt.Sprintf("%s  Commit: %s\n", indent, res.DriftInfo.ShortCommitID())))
		b.WriteString(attributeStyle.Render(fmt.Sprintf("%s  Branch: %s\n", indent, res.DriftInfo.BranchName)))
		b.WriteString(attributeStyle.Render(fmt.Sprintf("%s  Author: %s <%s>\n", indent, res.DriftInfo.AuthorName, res.DriftInfo.AuthorEmail)))
		b.WriteString(attributeStyle.Render(fmt.Sprintf("%s  Date: %s\n", indent, res.DriftInfo.CommitDate.Format("2006-01-02 15:04:05"))))
		if res.DriftInfo.HasUncommittedChanges {
			warnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11")) // Yellow
			b.WriteString(warnStyle.Render(fmt.Sprintf("%s  Status: Has uncommitted changes\n", indent)))
		}
		b.WriteString("\n")
	}

	// Show attribute changes
	action := getAction(res.Change.Actions)
	if action == "create" {
		b.WriteString(m.renderAttributes(indent, res.Change.After, "  "))
	} else if action == "delete" {
		b.WriteString(m.renderAttributes(indent, res.Change.Before, "  "))
	} else if action == "update" || action == "replace" {
		b.WriteString(m.renderAttributeDiff(indent, res.Change.Before, res.Change.After))
	}

	return b.String()
}

// renderAttributes renders attribute map with indentation
func (m Model) renderAttributes(baseIndent string, attrs map[string]interface{}, subIndent string) string {
	var b strings.Builder
	count := 0
	maxDisplay := 5

	for k, v := range attrs {
		if count >= maxDisplay {
			remaining := len(attrs) - maxDisplay
			b.WriteString(attributeStyle.Render(fmt.Sprintf("%s%s... (%d more attributes)\n", baseIndent, subIndent, remaining)))
			break
		}
		valueStr := fmt.Sprintf("%v", v)
		if len(valueStr) > 50 {
			valueStr = valueStr[:47] + "..."
		}
		b.WriteString(attributeStyle.Render(fmt.Sprintf("%s%s%s = ", baseIndent, subIndent, k)))
		b.WriteString(valueAddStyle.Render(valueStr))
		b.WriteString("\n")
		count++
	}

	return b.String()
}

// renderAttributeDiff renders before/after attribute differences
func (m Model) renderAttributeDiff(baseIndent string, before, after map[string]interface{}) string {
	var b strings.Builder

	b.WriteString(attributeStyle.Render(fmt.Sprintf("%s  Changes:\n", baseIndent)))

	count := 0
	maxDisplay := 5

	// Check for changed/added attributes
	for k, afterVal := range after {
		if count >= maxDisplay {
			break
		}
		beforeVal, existedBefore := before[k]

		if !existedBefore {
			// New attribute
			valueStr := fmt.Sprintf("%v", afterVal)
			if len(valueStr) > 40 {
				valueStr = valueStr[:37] + "..."
			}
			b.WriteString(attributeStyle.Render(fmt.Sprintf("%s    + %s = ", baseIndent, k)))
			b.WriteString(valueAddStyle.Render(valueStr))
			b.WriteString("\n")
			count++
		} else if fmt.Sprintf("%v", beforeVal) != fmt.Sprintf("%v", afterVal) {
			// Changed attribute
			beforeStr := fmt.Sprintf("%v", beforeVal)
			afterStr := fmt.Sprintf("%v", afterVal)
			if len(beforeStr) > 30 {
				beforeStr = beforeStr[:27] + "..."
			}
			if len(afterStr) > 30 {
				afterStr = afterStr[:27] + "..."
			}
			b.WriteString(attributeStyle.Render(fmt.Sprintf("%s    ~ %s: ", baseIndent, k)))
			b.WriteString(valueRemStyle.Render(beforeStr))
			b.WriteString(attributeStyle.Render(" → "))
			b.WriteString(valueAddStyle.Render(afterStr))
			b.WriteString("\n")
			count++
		}
	}

	// Check for removed attributes
	for k := range before {
		if count >= maxDisplay {
			break
		}
		if _, exists := after[k]; !exists {
			valueStr := fmt.Sprintf("%v", before[k])
			if len(valueStr) > 40 {
				valueStr = valueStr[:37] + "..."
			}
			b.WriteString(attributeStyle.Render(fmt.Sprintf("%s    - %s = ", baseIndent, k)))
			b.WriteString(valueRemStyle.Render(valueStr))
			b.WriteString("\n")
			count++
		}
	}

	if count == 0 {
		b.WriteString(attributeStyle.Render(fmt.Sprintf("%s    (no attribute changes shown)\n", baseIndent)))
	}

	return b.String()
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
			// Full line highlight with selection indicator
			b.WriteString(selectedStyle.Render("❯ " + line))
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
			// Full line highlight with selection indicator
			b.WriteString(selectedStyle.Render("❯ " + line))
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
	return m.nodes // For now, all nodes are visible at the top level
}

// adjustViewport adjusts the viewport to keep the cursor visible
func (m Model) adjustViewport() {
	if m.cursor < m.viewportTop {
		m.viewportTop = m.cursor
	} else if m.cursor >= m.viewportTop+m.viewportSize {
		m.viewportTop = m.cursor - m.viewportSize + 1
	}
}

// countActions counts different action types in the plan
func (m Model) countActions() (creates, updates, deletes, replaces int) {
	for _, res := range m.plan.Resources {
		action := getAction(res.Change.Actions)
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

// getAction determines the primary action from a list of actions
func getAction(actions []string) string {
	if len(actions) == 0 {
		return "no-op"
	}

	// Handle replace (delete + create)
	hasDelete := false
	hasCreate := false
	for _, a := range actions {
		if a == "delete" {
			hasDelete = true
		}
		if a == "create" {
			hasCreate = true
		}
	}
	if hasDelete && hasCreate {
		return "replace"
	}

	// Return first action
	return actions[0]
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

// Run starts the TUI application
func Run(plan *models.PlanResult) error {
	p := tea.NewProgram(NewModel(plan), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
