# tplan

A Terminal User Interface (TUI) for viewing and analyzing Terraform plans with git integration and drift detection.

## Overview

`tplan` is a command-line tool that provides an interactive interface for reviewing Terraform plans. It parses Terraform JSON plan output from stdin and presents it in an easy-to-navigate hierarchical TUI, similar to git log. With built-in drift detection and git integration, you can see exactly which commits and authors modified the resources.

## Features

- **JSON Plan Format**: Works with Terraform's complete JSON plan format for accurate, detailed information
- **Interactive Hierarchical TUI**: Navigate through Terraform plan changes in a git log-style tree view
- **Expand/Collapse**: Toggle resource details with keyboard controls
- **Complete Attribute Display**: View all resource attributes, including nested structures
- **Git Integration**: Drift detection showing commit ID, branch, author, and file information
- **Error & Warning Display**: Dedicated tabs for errors and warnings
- **Color-Coded Actions**: Visual distinction between creates (green), updates (yellow), deletes (red), and replaces (blue)
- **Resource Details**: View before/after states and attribute changes
- **Report Generation**: Export plan analysis to Markdown format

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

tplan requires Terraform's JSON plan format for complete and accurate information:

```bash
# Create a plan file
terraform plan -out=tfplan

# View it with tplan
terraform show -json tfplan | tplan
```

### Drift Detection

Enable drift detection and git integration with the `-drift` flag:

```bash
terraform plan -out=tfplan
terraform show -json tfplan | tplan -drift
```

When drift is detected, tplan will show:
- The Terraform file containing the resource
- Git commit ID (last commit that modified the file)
- Git branch name
- Commit author name and email
- Commit date
- Uncommitted changes status

### Report Generation

Generate a Markdown report of the plan:

```bash
terraform plan -out=tfplan
terraform show -json tfplan | tplan -report
```

Combine with drift detection:

```bash
terraform plan -out=tfplan
terraform show -json tfplan | tplan -report -drift
```

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
- `q`: Quit

## Examples

### Example 1: Basic Plan Review

```bash
cd your-terraform-project
terraform plan -out=tfplan
terraform show -json tfplan | tplan
```

Navigate through the changes, expand resources to see all attributes and details, and review errors/warnings in separate tabs.

### Example 2: Drift Detection

```bash
cd your-terraform-project
terraform plan -out=tfplan
terraform show -json tfplan | tplan -drift
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

### Example 3: Generate Report

```bash
terraform plan -out=tfplan
terraform show -json tfplan | tplan -report -drift
```

Creates a `report.md` file with complete plan analysis including git information.

## Why JSON Format Only?

tplan requires the full JSON plan format because:

- **Complete Data**: JSON format includes all resource attributes and nested structures
- **Accurate Parsing**: Structured data ensures reliable parsing
- **No Ambiguity**: Unlike text format, JSON is designed for programmatic consumption
- **Better Experience**: You get to see all the details, not just a subset

The text format from `terraform plan` is designed for human reading and doesn't include complete structured data. The streaming JSON format from `terraform plan -json` doesn't include detailed before/after attribute values.

## Project Structure

```
tplan/
├── cmd/tplan/              # Main application entry point
│   └── main.go            # CLI with stdin pipe and flags
├── internal/
│   ├── parser/            # Terraform JSON plan parsing
│   │   └── parser.go      # JSON parser using terraform-json library
│   ├── tui/               # Terminal UI components
│   │   └── tui.go         # Bubble Tea hierarchical tree view
│   ├── git/               # Git integration utilities
│   │   └── git.go         # Commit, branch, author detection
│   ├── report/            # Report generation
│   │   └── report.go      # Markdown report generator
│   └── models/            # Data structures and models
│       ├── plan.go        # Plan and resource models
│       └── drift.go       # Drift information models
├── examples/              # Example usage and demos
├── .github/workflows/     # GitHub Actions for releases
├── go.mod                 # Go module definition
└── README.md              # This file
```

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) v0.25.0 - TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) v0.9.1 - Terminal styling
- [terraform-json](https://github.com/hashicorp/terraform-json) v0.18.0 - Terraform JSON parsing

## How It Works

1. **Input**: tplan reads Terraform JSON plan from stdin
2. **Validation**: Ensures input is valid JSON plan format with format_version
3. **Parsing**: Uses terraform-json library for reliable parsing
4. **Drift Detection**: If `-drift` flag is used, searches for corresponding .tf files and queries git
5. **Display**: Renders an interactive TUI with hierarchical tree view showing all resource details
6. **Navigation**: Use keyboard to explore changes, errors, and warnings

## Development

### Prerequisites

- Go 1.21 or later
- Git (for drift detection features)
- Terraform (for generating test plans)

### Building

```bash
go build -o tplan ./cmd/tplan
```

### Testing

```bash
# Create a test plan
cd examples
terraform init
terraform plan -out=tfplan
terraform show -json tfplan | ../tplan
```

## Versioning

The version is automatically set during GitHub Actions releases via git tags:

```bash
git tag v1.0.0
git push origin v1.0.0
```

The GitHub Actions workflow will build binaries for multiple platforms with the version embedded.

## License

MIT

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
