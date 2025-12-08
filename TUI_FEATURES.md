# Terraform Plan TUI - Feature Documentation

## Overview
A comprehensive Terminal User Interface (TUI) for visualizing Terraform plan results using Bubble Tea and Lipgloss.

## Features

### 1. Hierarchical Tree View
- Resources displayed in a tree structure similar to git log
- Expand/collapse functionality for detailed view
- Proper indentation showing resource hierarchy
- Tree-like visual indicators (▸ collapsed, ▾ expanded)

### 2. Keyboard Navigation
| Key | Action |
|-----|--------|
| `↑` / `k` | Move cursor up |
| `↓` / `j` | Move cursor down |
| `Enter` / `Space` | Expand/collapse selected node |
| `Tab` | Switch to next view (Changes → Errors → Warnings) |
| `Shift+Tab` | Switch to previous view |
| `e` | Expand all nodes |
| `c` | Collapse all nodes |
| `g` | Jump to top |
| `G` | Jump to bottom |
| `q` / `Ctrl+C` | Quit application |

### 3. Color-Coded Actions
Resources are color-coded based on their Terraform action:
- **Green (✚)**: Create - New resources being created
- **Yellow (~)**: Update - Existing resources being modified
- **Red (✖)**: Delete - Resources being destroyed
- **Blue (⟳)**: Replace - Resources being deleted and recreated
- **Gray (•)**: No-op - No changes

### 4. Summary Section
At the top of the interface, a bordered summary box displays:
- Total number of creates
- Total number of updates
- Total number of deletes
- Total number of replaces
- Terraform version used

Example:
```
╭─────────────────────────────────────────────────────────────╮
│ ✚ Create: 2  ~ Update: 1  ✖ Delete: 1  ⟳ Replace: 1  │  ... │
╰─────────────────────────────────────────────────────────────╯
```

### 5. Multiple View Modes (Tab Navigation)
Three different views accessible via Tab key:

#### Changes View (Default)
- Shows all resource changes
- Expandable tree nodes
- Displays resource details when expanded:
  - Resource type
  - Provider name
  - Mode (managed/data)
  - Attribute changes

#### Errors View
- Lists all Terraform errors
- Error icon (✖) with description
- Empty state message if no errors

#### Warnings View
- Lists all Terraform warnings
- Warning icon (⚠) with description
- Empty state message if no warnings

### 6. Scrolling Support
- Automatic viewport management for long lists
- Cursor stays visible as you navigate
- Scroll indicator shows current position: `[1-20 of 50]`
- Smooth scrolling with cursor adjustment

### 7. Expanded Resource Details
When a node is expanded (Enter/Space), it shows:
- **Metadata**: Type, Provider, Mode
- **For Create actions**: All new attributes
- **For Delete actions**: Attributes being removed
- **For Update/Replace actions**: Before/after comparison
  - New attributes marked with `+`
  - Removed attributes marked with `-`
  - Changed attributes marked with `~` showing: `old → new`
- Attribute truncation for long values (max 50 chars)
- Limited display (max 5 attributes) with "... (N more)" indicator

### 8. Styled UI Components

#### Tabs
- Active tab: Bright background with bold text
- Inactive tabs: Dimmed background
- Tab counts: Shows number of items in each view

#### Selection Highlight
- Currently selected item has highlighted background
- Bold text for selected items
- Clear visual indicator of current position

#### Tree Structure
- Dimmed color for tree lines and expand/collapse icons
- Proper indentation for hierarchy levels
- Consistent spacing throughout

#### Attribute Display
- Dimmed text for attribute names
- Green for added/new values
- Red for removed values
- Arrows (→) for changed values

### 9. Responsive Design
- Adapts to terminal window size
- Viewport automatically adjusts on resize
- Minimum size recommendations: 80x24

### 10. Help Text
Always visible at the bottom:
```
↑/↓: Navigate  Enter/Space: Expand/Collapse  Tab: Switch View  
e: Expand All  c: Collapse All  g/G: Top/Bottom  q: Quit
```

## Usage Example

```go
package main

import (
    "log"
    "github.com/yourusername/tplan/internal/models"
    "github.com/yourusername/tplan/internal/tui"
)

func main() {
    // Create your Terraform plan
    plan := models.Plan{
        TerraformVersion: "1.5.0",
        Resources: []models.Resource{
            // ... your resources
        },
    }
    
    // Run the TUI
    if err := tui.Run(plan); err != nil {
        log.Fatal(err)
    }
}
```

## Architecture

### Key Types

#### Model
The main Bubble Tea model containing:
- `plan`: The Terraform plan data
- `nodes`: Hierarchical tree nodes
- `cursor`: Current selection position
- `viewMode`: Active view (Changes/Errors/Warnings)
- `viewportTop/Size`: Scroll position management
- `errors/warnings`: Error and warning messages
- `width/height`: Terminal dimensions

#### TreeNode
Represents a node in the tree:
- `Resource`: Associated Terraform resource
- `Expanded`: Whether node is expanded
- `Children`: Child nodes (for future hierarchical support)
- `Level`: Indentation level

#### ViewMode
Enum for different views:
- `ViewChanges` (0)
- `ViewErrors` (1)
- `ViewWarnings` (2)

### Core Functions

- `NewModel()`: Initialize the TUI model
- `Init()`: Bubble Tea initialization
- `Update()`: Handle keyboard input and window resize
- `View()`: Render the complete UI
- `Run()`: Start the TUI application

## Future Enhancements

Potential improvements:
1. Search/filter functionality
2. Resource grouping by provider or module
3. Export to file
4. Diff view with syntax highlighting
5. Resource dependency graph
6. Mouse support for clicking to expand/collapse
7. Copy resource details to clipboard
8. Custom color themes

## Dependencies

- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/lipgloss` - Styling and layout
- `github.com/yourusername/tplan/internal/models` - Data models
