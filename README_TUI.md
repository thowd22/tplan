# Terraform Plan TUI - Quick Start

## Installation

```bash
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/lipgloss
```

## Run Example

```bash
go run example_tui_usage.go
```

## Features at a Glance

### Visual Design
- 🌳 **Hierarchical Tree View** - Expand/collapse resources
- 🎨 **Color Coded** - Green (create), Yellow (update), Red (delete), Blue (replace)
- 📊 **Summary Bar** - Quick stats at the top
- 🗂️ **Multiple Tabs** - Changes, Errors, Warnings

### Keyboard Controls
```
↑/↓ or j/k  - Navigate up/down
Enter/Space - Expand/collapse resource
Tab         - Switch between tabs
e           - Expand all
c           - Collapse all
g           - Jump to top
G           - Jump to bottom
q           - Quit
```

### What You See

```
╭─────────────────────────────────────────────────────╮
│ ✚ Create: 2  ~ Update: 1  ✖ Delete: 1  ⟳ Replace: 1│
╰─────────────────────────────────────────────────────╯

▾ ✚ aws_instance.web_server        [Selected & Expanded]
    Type: aws_instance
    Provider: aws
    ami = ami-12345678
    instance_type = t2.micro

▸ ~ aws_s3_bucket.data
▸ ✖ aws_security_group.old_sg
```

## Integration Example

```go
package main

import (
    "github.com/yourusername/tplan/internal/models"
    "github.com/yourusername/tplan/internal/tui"
)

func main() {
    plan := models.Plan{
        TerraformVersion: "1.5.0",
        Resources: []models.Resource{
            {
                Address: "aws_instance.web",
                Type:    "aws_instance",
                Name:    "web",
                Mode:    "managed",
                ProviderName: "aws",
                Change: models.Change{
                    Actions: []string{"create"},
                    After: map[string]interface{}{
                        "ami": "ami-12345",
                    },
                },
            },
        },
    }
    
    tui.Run(plan)
}
```

## Documentation

- `TUI_FEATURES.md` - Complete feature documentation
- `TUI_SCREENSHOT.txt` - Visual demo of the interface
- `IMPLEMENTATION_SUMMARY.md` - Technical implementation details
- `internal/tui/tui.go` - Source code (566 lines)

## Architecture

```
Model (Bubble Tea)
  ├── plan: Terraform plan data
  ├── nodes: Tree structure
  ├── cursor: Current selection
  ├── viewMode: Active tab
  └── viewport: Scroll management

View Components
  ├── Tab Bar (Changes/Errors/Warnings)
  ├── Summary (Stats with icons)
  ├── Tree View (Hierarchical resources)
  └── Help Bar (Keyboard shortcuts)
```

## Color Scheme

| Action  | Color  | Icon |
|---------|--------|------|
| Create  | Green  | ✚    |
| Update  | Yellow | ~    |
| Delete  | Red    | ✖    |
| Replace | Blue   | ⟳    |
| No-op   | Gray   | •    |

## Key Files

- `internal/tui/tui.go` - Main TUI implementation
- `example_tui_usage.go` - Working example with sample data
- `internal/models/plan.go` - Data structures

## Dependencies

- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/lipgloss` - Terminal styling
- Go 1.21+

## Tips

1. **Terminal Size**: Works best in 80x24 or larger
2. **Colors**: Requires terminal with ANSI color support
3. **Scrolling**: Automatic for long lists
4. **Performance**: Handles hundreds of resources efficiently

## Next Steps

1. Customize colors in `tui.go` (lines 44-62)
2. Add custom views by extending `ViewMode`
3. Implement resource grouping in `buildTreeNodes()`
4. Add search/filter functionality

---

Made with ❤️ using Bubble Tea
