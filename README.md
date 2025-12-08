# tplan

A Terminal User Interface (TUI) for viewing and analyzing Terraform plans with git integration and drift detection.

## Overview

`tplan` is a command-line tool that provides an interactive interface for reviewing Terraform plans. It parses Terraform plan output (both human-readable and JSON formats) from stdin and presents it in an easy-to-navigate hierarchical TUI, similar to git log. With built-in drift detection and git integration, you can see exactly which commits and authors modified the resources that have drifted.

## Features

- **Pipe-friendly**: Direct integration with `terraform plan` via stdin
- **Dual Format Support**: Works with both human-readable and JSON plan output
- **Interactive Hierarchical TUI**: Navigate through Terraform plan changes in a git log-style tree view
- **Expand/Collapse**: Toggle resource details with keyboard controls
- **Git Integration**: Full drift detection showing commit ID, branch, author, and file information
- **Error & Warning Display**: Dedicated tabs for errors and warnings
- **Color-Coded Actions**: Visual distinction between creates (green), updates (yellow), deletes (red), and replaces (blue)
- **Resource Details**: View before/after states and attribute changes

## Installation

Build from source:

```bash
cd tplan
go build -o tplan ./cmd/tplan
```

Or install directly:

```bash
go install ./cmd/tplan
```

## Usage

### Basic Usage

Pipe your Terraform plan directly into tplan:

```bash
# Human-readable format
terraform plan | tplan

# JSON format
terraform plan -json | tplan
```

### Drift Detection

Enable drift detection and git integration with the `-drift` flag:

```bash
terraform plan | tplan -drift
```

When drift is detected, tplan will show:
- The Terraform file containing the drifted resource
- Git commit ID (last commit that modified the file)
- Git branch name
- Commit author name and email
- Commit date
- Uncommitted changes status

### Help

```bash
tplan -help
```

### Keyboard Shortcuts

- `↑/↓` or `j/k`: Navigate through resources
- `Enter` or `Space`: Expand/collapse resource details
- `e`: Expand all resources
- `c`: Collapse all resources
- `Tab`: Switch between Changes/Errors/Warnings tabs
- `g`: Jump to top
- `G`: Jump to bottom
- `q` or `Esc`: Quit

## Examples

### Example 1: Basic Plan Review

```bash
cd your-terraform-project
terraform plan | tplan
```

Navigate through the changes, expand resources to see details, and review errors/warnings in separate tabs.

### Example 2: Drift Detection

```bash
cd your-terraform-project
terraform plan | tplan -drift
```

When you expand a resource, you'll see git information like:

```
  Git Information:
    File: modules/vpc/main.tf
    Commit: a1b2c3d4
    Branch: main
    Author: John Doe <john@example.com>
    Date: 2024-12-08 10:30:45
```

### Example 3: Using JSON Format

```bash
terraform plan -json | tplan -drift
```

JSON format provides more detailed information and is recommended for complex plans.

## Project Structure

```
tplan/
├── cmd/tplan/              # Main application entry point
│   └── main.go            # CLI with stdin pipe and -drift flag support
├── internal/
│   ├── parser/            # Terraform plan parsing (JSON + text)
│   │   └── parser.go      # Dual-format parser with auto-detection
│   ├── tui/               # Terminal UI components
│   │   └── tui.go         # Bubble Tea hierarchical tree view
│   ├── git/               # Git integration utilities
│   │   └── git.go         # Commit, branch, author detection
│   └── models/            # Data structures and models
│       ├── plan.go        # Plan and resource models
│       └── drift.go       # Drift information models
├── examples/              # Example usage and demos
├── go.mod                 # Go module definition
└── README.md              # This file
```

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) v0.25.0 - TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) v0.9.1 - Terminal styling
- [terraform-json](https://github.com/hashicorp/terraform-json) v0.18.0 - Terraform JSON parsing

## How It Works

1. **Input**: tplan reads Terraform plan output from stdin
2. **Parsing**: Automatically detects format (JSON or text) and parses it
3. **Drift Detection**: If `-drift` flag is used, searches for corresponding .tf files and queries git
4. **Display**: Renders an interactive TUI with hierarchical tree view
5. **Navigation**: Use keyboard to explore changes, errors, and warnings

## Development

### Prerequisites

- Go 1.21 or later
- Git (for drift detection features)

### Building

```bash
go build -o tplan ./cmd/tplan
```

### Testing the Parser

```bash
# Test with example data
go run examples/test_parser.go test_plan.json

# Test with live Terraform plan
terraform plan -json | go run examples/test_parser.go -
```

### Testing the TUI

```bash
# Run TUI example with sample data
go run example_tui_usage.go
```

### Testing Git Integration

```bash
# Test git drift detection
go run test_git.go
```

## License

MIT

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
