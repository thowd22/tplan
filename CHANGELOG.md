# Changelog

All notable changes to tplan will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2024-12-08

### Added
- Initial release of tplan
- Terminal User Interface (TUI) for analyzing Terraform plans
- Hierarchical tree view similar to git log
- Expand/collapse functionality for resource details
- Support for both JSON and text Terraform plan formats
- Stdin pipe support (`terraform plan | tplan`)
- Drift detection mode with `-drift` flag
- Git integration showing:
  - Commit ID of last modification
  - Branch name
  - Author name and email
  - Commit date
  - File path
  - Uncommitted changes detection
- Three-tab interface:
  - Changes tab: All resource modifications
  - Errors tab: Plan errors and diagnostics
  - Warnings tab: Plan warnings
- Color-coded resource actions:
  - Green for creates
  - Yellow for updates
  - Red for deletes
  - Blue for replaces
- Keyboard navigation:
  - Arrow keys and vim-style (j/k) navigation
  - Enter/Space to expand/collapse
  - Tab to switch between views
  - g/G to jump to top/bottom
  - e/c to expand/collapse all
  - q/Esc to quit
- Summary bar showing totals for each action type
- Resource detail view with before/after states
- Attribute change visualization
- Multi-platform binaries (Linux, macOS, Windows)
- Support for both amd64 and arm64 architectures

### Technical Details
- Built with Go 1.21
- Uses Bubble Tea TUI framework
- Lipgloss for terminal styling
- terraform-json for JSON plan parsing
- Automatic format detection (JSON vs text)
- Comprehensive error handling
- Module support with nested paths

### Documentation
- Complete README with usage examples
- Comprehensive usage guide (USAGE_GUIDE.md)
- Git integration documentation (GIT_INTEGRATION.md)
- Parser documentation (PARSER_README.md)
- TUI features guide (TUI_FEATURES.md)
- Implementation summary
- Multiple example files

[Unreleased]: https://github.com/yourusername/tplan/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/yourusername/tplan/releases/tag/v1.0.0
