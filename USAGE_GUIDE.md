# tplan Usage Guide

Complete guide for using tplan to analyze Terraform plans with drift detection.

## Quick Start

```bash
# Build tplan
go build -o tplan ./cmd/tplan

# Basic usage
terraform plan | ./tplan

# With drift detection
terraform plan | ./tplan -drift
```

## Detailed Usage

### 1. Basic Plan Analysis

The simplest way to use tplan is to pipe your Terraform plan directly:

```bash
cd your-terraform-project
terraform plan | tplan
```

**What you'll see:**
- Summary bar showing total creates, updates, deletes, and replaces
- Hierarchical tree view of all resource changes
- Color-coded actions:
  - 🟢 Green: Resources being created
  - 🟡 Yellow: Resources being updated
  - 🔴 Red: Resources being deleted
  - 🔵 Blue: Resources being replaced

**Navigation:**
- Use `↑`/`↓` or `j`/`k` to navigate
- Press `Enter` or `Space` to expand a resource and see details
- Press `Tab` to switch between Changes, Errors, and Warnings tabs

### 2. Drift Detection Mode

When you want to understand why resources have drifted and who made changes:

```bash
terraform plan | tplan -drift
```

**What drift mode does:**
1. Identifies drifted resources in your plan
2. Finds the corresponding `.tf` files
3. Queries git to get:
   - Last commit that modified the file
   - Commit author name and email
   - Branch name
   - Commit date
   - Uncommitted changes status

**Example output when you expand a resource:**
```
  Type: aws_instance
  Provider: registry.terraform.io/hashicorp/aws
  Mode: managed

  Git Information:
    File: modules/vpc/main.tf
    Commit: a1b2c3d4
    Branch: main
    Author: Jane Doe <jane@example.com>
    Date: 2024-12-07 14:30:22
```

### 3. Using JSON Format

For more detailed plans, use JSON format:

```bash
terraform plan -json | tplan -drift
```

**Advantages of JSON format:**
- More detailed resource information
- Better parsing of complex nested structures
- Includes module information
- More accurate error and warning detection

### 4. Saving Plans for Later Review

You can save plan output and review it later:

```bash
# Save plan output
terraform plan > plan.txt

# Review later
cat plan.txt | tplan

# Or with JSON
terraform plan -json > plan.json
cat plan.json | tplan -drift
```

## Keyboard Controls Reference

### Navigation
- `↑` / `k` - Move cursor up
- `↓` / `j` - Move cursor down
- `g` - Jump to top
- `G` - Jump to bottom

### Actions
- `Enter` - Expand/collapse selected resource
- `Space` - Expand/collapse selected resource
- `e` - Expand all resources
- `c` - Collapse all resources

### View Switching
- `Tab` - Next tab (Changes → Errors → Warnings → Changes)
- `Shift+Tab` - Previous tab

### Exit
- `q` - Quit
- `Esc` - Quit
- `Ctrl+C` - Quit

## Understanding the Display

### Summary Bar

```
✚ Create: 3  ~ Update: 2  ✖ Delete: 1  ⟳ Replace: 0  │  Version: 1.5.0
```

Shows the total count of each action type and Terraform version.

### Resource Tree View

```
▸ ✚ aws_instance.web
▸ ~ aws_security_group.allow_http
▾ ✖ aws_s3_bucket.old_bucket
    Type: aws_s3_bucket
    Provider: registry.terraform.io/hashicorp/aws
    Mode: managed
    bucket: my-old-bucket
    acl: private
```

- `▸` - Collapsed resource (press Enter to expand)
- `▾` - Expanded resource (press Enter to collapse)
- Icon and color indicate action type

### Expanded Resource Details

When you expand a resource, you see:

1. **Resource metadata**
   - Type (e.g., aws_instance)
   - Provider
   - Mode (managed or data)

2. **Git information** (if -drift flag is used)
   - File path
   - Commit ID
   - Branch
   - Author
   - Date

3. **Attribute changes**
   - For creates: All new attributes
   - For deletes: All removed attributes
   - For updates: Before → After diffs

### Errors Tab

Switch to the Errors tab (press Tab twice) to see:
- Parse errors
- Plan errors
- Resource-specific errors
- Each error shows the resource it's related to (if applicable)

### Warnings Tab

Switch to the Warnings tab (press Tab three times) to see:
- Deprecation warnings
- Best practice suggestions
- Resource-specific warnings

## Common Workflows

### Workflow 1: Pre-Apply Review

```bash
# 1. Review changes
terraform plan | tplan

# 2. Navigate through resources
#    - Press ↓ to move through list
#    - Press Enter to expand interesting resources
#    - Press Tab to check for errors/warnings

# 3. If everything looks good, apply
terraform apply
```

### Workflow 2: Investigating Drift

```bash
# 1. Detect drift with git info
terraform plan | tplan -drift

# 2. Find drifted resources (usually updates or replaces)
#    - Navigate to updated resources
#    - Press Enter to expand

# 3. Review git information
#    - See which file has the drift
#    - Identify who made the last change
#    - Check the commit date

# 4. Decide on action
#    - Update the code to match reality, or
#    - Apply the plan to restore desired state
```

### Workflow 3: Understanding Complex Plans

```bash
# 1. Generate JSON plan for maximum detail
terraform plan -json | tplan -drift

# 2. Use tree navigation to understand hierarchy
#    - Module-scoped resources are clearly marked
#    - Parent-child relationships are visible

# 3. Expand all resources for full view
#    - Press 'e' to expand everything
#    - Press 'c' to collapse when done

# 4. Export plan for documentation
terraform plan > plan-review-$(date +%Y%m%d).txt
```

### Workflow 4: Team Code Review

```bash
# 1. Create PR with Terraform changes
git checkout -b feature/new-infrastructure

# 2. Generate plan with full context
terraform plan -json > plan.json

# 3. Review with tplan
cat plan.json | tplan -drift

# 4. Take screenshot or record session for PR
#    - Expand key resources
#    - Show git attribution
#    - Document any warnings

# 5. Commit plan output to PR
git add plan.json
git commit -m "Add Terraform plan for review"
```

## Tips and Tricks

### Tip 1: Large Plans
For very large plans (100+ resources):
- Use `g` and `G` to jump to top/bottom
- Use `e` to expand all, then scroll to find patterns
- Focus on specific action types by visual scanning (colors)

### Tip 2: Module Changes
When working with modules:
- Module-prefixed resources group together
- Expand a few resources from each module to verify
- Check if module updates affect multiple resources

### Tip 3: Drift Detection Performance
For large repositories:
- Git queries can take time for first run
- Consider running without `-drift` for quick reviews
- Use `-drift` when you need to investigate specific changes

### Tip 4: CI/CD Integration
Integrate tplan in CI pipelines:
```bash
# In your CI script
terraform plan -json | tplan > plan-summary.txt || true
# Attach plan-summary.txt to build artifacts
```

## Troubleshooting

### Issue: "Error parsing Terraform plan"
**Solution:** Ensure you're piping valid Terraform output:
```bash
# Check plan output first
terraform plan | tee plan.txt
cat plan.txt | tplan
```

### Issue: "Not a git repository"
**Solution:** Only use `-drift` flag in git repositories:
```bash
cd your-terraform-project
git status  # Verify you're in a git repo
terraform plan | tplan -drift
```

### Issue: "Could not get git information"
**Possible causes:**
1. `.tf` file not tracked by git → Add and commit it
2. File has uncommitted changes → Commit or stash changes
3. Resource not found in any `.tf` file → Check resource address

### Issue: TUI not displaying correctly
**Solution:** Ensure terminal supports colors and UTF-8:
```bash
echo $TERM  # Should be xterm-256color or similar
locale | grep UTF-8  # Should show UTF-8
```

## Advanced Features

### Custom Module Paths
tplan automatically finds resources in modules:
- Searches `modules/` directory
- Follows Terraform module conventions
- Works with nested modules

### Error Recovery
tplan gracefully handles:
- Malformed plan output (shows what it could parse)
- Missing git information (shows resources without git data)
- Partial plans (shows available information)

### Performance
- Parses plans in < 1 second for typical projects
- Handles plans with 1000+ resources
- Git queries run in parallel for drift detection

## Getting Help

```bash
# Show help message
tplan -help

# Version info (from README or git)
cat README.md | grep "Version"

# Report issues
# https://github.com/yourusername/tplan/issues
```
