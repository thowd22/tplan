# tplan

A Terminal User Interface (TUI) wrapper for Terraform/OpenTofu that makes reviewing plans interactive and easy.

## Overview

`tplan` is a drop-in replacement for `terraform plan` that automatically runs the plan, captures the output, and displays it in an interactive terminal UI. Simply use `tplan` instead of `terraform plan` and enjoy a better planning experience with git integration and drift detection.

## Features

- **Drop-in Replacement**: Just use `tplan` instead of `terraform plan`
- **Auto-detection**: Automatically detects and uses Terraform or OpenTofu
- **Interactive TUI**: Navigate through plan changes in a git log-style tree view
- **Expand/Collapse**: Toggle resource details with keyboard controls
- **Complete Attribute Display**: View all resource attributes, including nested structures
- **Git Integration**: Drift detection showing commit ID, branch, author, and file information
- **Error & Warning Display**: Dedicated tabs for errors and warnings
- **Color-Coded Actions**: Visual distinction between creates (green), updates (yellow), deletes (red), and replaces (blue)
- **Report Generation**: Export plan analysis to Markdown format
- **Pass-through Arguments**: All terraform/tofu arguments work seamlessly

## Installation

Build from source:

```bash
cd tplan
go build -o tplan ./cmd/tplan
```

Install to your PATH:

```bash
go install ./cmd/tplan
# Or copy the binary
sudo cp tplan /usr/local/bin/
```

## Requirements

Either **Terraform** or **OpenTofu** must be installed and available in your PATH.

- Install Terraform: https://developer.hashicorp.com/terraform/install
- Install OpenTofu: https://opentofu.org/docs/intro/install/

tplan will automatically detect which one is available (preferring Terraform if both are present).

## Usage

### Basic Usage

Simply replace `terraform plan` with `tplan`:

```bash
# Instead of this:
terraform plan

# Use this:
tplan
```

tplan will:
1. Run `terraform plan -out=.tplan-temp.tfplan`
2. Convert it to JSON with `terraform show -json`
3. Display the results in an interactive TUI
4. Clean up the temporary plan file when you exit

### Drift Detection

Enable drift detection and git integration:

```bash
tplan -drift
```

When you expand a resource, you'll see git information:
- The Terraform file containing the resource
- Git commit ID (last commit that modified the file)
- Git branch name
- Commit author name and email
- Commit date
- Uncommitted changes status

### Report Generation

Generate a Markdown report instead of the TUI:

```bash
tplan -report
```

Combine with drift detection:

```bash
tplan -report -drift
```

### Passing Terraform Arguments

All additional arguments are passed directly to terraform/tofu:

```bash
# Target specific resource
tplan -target=aws_instance.web

# Use variable file
tplan -var-file=production.tfvars

# Plan destroy
tplan -destroy

# Combine multiple arguments
tplan -var-file=prod.tfvars -target=module.vpc
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

### Example 1: Quick Plan Review

```bash
cd your-terraform-project
tplan
```

Navigate through the changes, expand resources to see all attributes, and review errors/warnings in separate tabs.

### Example 2: Drift Detection

```bash
cd your-terraform-project
tplan -drift
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
tplan -report -drift
```

Creates a `report.md` file with complete plan analysis including git information.

### Example 4: Target Specific Resources

```bash
# Plan changes for specific resource
tplan -target=aws_instance.web

# Plan changes for module
tplan -target=module.database
```

### Example 5: Using Variable Files

```bash
# Use production variables
tplan -var-file=environments/production.tfvars

# Multiple variable files
tplan -var-file=common.tfvars -var-file=prod.tfvars
```

## How It Works

1. **Detection**: tplan checks if terraform or tofu is available
2. **Planning**: Runs `terraform/tofu plan -out=.tplan-temp.tfplan [args]`
3. **Conversion**: Converts plan to JSON with `terraform/tofu show -json`
4. **Parsing**: Parses the JSON using the terraform-json library
5. **Git Integration**: If `-drift` is enabled, queries git for resource file history
6. **Display**: Shows results in interactive TUI or generates report
7. **Cleanup**: Removes temporary `.tplan-temp.tfplan` file on exit

## Project Structure

```
tplan/
├── cmd/tplan/              # Main application
│   └── main.go            # CLI wrapper that executes terraform/tofu
├── internal/
│   ├── parser/            # JSON plan parsing
│   │   └── parser.go      # Parser using terraform-json library
│   ├── tui/               # Terminal UI
│   │   └── tui.go         # Interactive tree view (Bubble Tea)
│   ├── git/               # Git integration
│   │   └── git.go         # Commit and file history detection
│   ├── report/            # Report generation
│   │   └── report.go      # Markdown report generator
│   └── models/            # Data structures
│       ├── plan.go        # Plan and resource models
│       └── drift.go       # Drift information models
├── examples/              # Example code and demos
├── .github/workflows/     # GitHub Actions for releases
├── go.mod                 # Go module definition
└── README.md              # This file
```

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) v0.25.0 - TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) v0.9.1 - Terminal styling
- [terraform-json](https://github.com/hashicorp/terraform-json) v0.18.0 - Terraform JSON parsing

## Comparison with terraform plan

| Feature | `terraform plan` | `tplan` |
|---------|-----------------|---------|
| View plan output | ✓ | ✓ |
| Navigate resources | ✗ | ✓ |
| Expand/collapse details | ✗ | ✓ |
| See all attributes | ✗ | ✓ |
| Git integration | ✗ | ✓ |
| Color-coded actions | ✗ | ✓ |
| Error/warning tabs | ✗ | ✓ |
| Generate reports | ✗ | ✓ |
| Pass-through args | ✓ | ✓ |

## Development

### Prerequisites

- Go 1.21 or later
- Terraform or OpenTofu
- Git (for drift detection)

### Building

```bash
go build -o tplan ./cmd/tplan
```

### Testing

```bash
# Initialize a test Terraform project
cd examples
terraform init

# Run tplan
../tplan
```

## Versioning

The version is automatically set during GitHub Actions releases via git tags:

```bash
git tag v1.0.0
git push origin v1.0.0
```

The workflow builds binaries for multiple platforms with the version embedded.

## Troubleshooting

### "Neither Terraform nor OpenTofu is installed"

Install either Terraform or OpenTofu and ensure it's in your PATH:

```bash
# Check if terraform is available
which terraform

# Check if tofu is available
which tofu
```

### Temporary plan file not cleaned up

If tplan crashes or is force-killed, the `.tplan-temp.tfplan` file might remain. You can safely delete it:

```bash
rm .tplan-temp.tfplan
```

### Git information not showing

Make sure:
1. You're running tplan from within a git repository
2. The Terraform files are tracked by git
3. You're using the `-drift` flag

## License

MIT

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
