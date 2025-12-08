# Report Generation Feature

## Overview

tplan can generate comprehensive Markdown reports from Terraform plans using the `-report` flag. These reports provide a detailed, organized view of all infrastructure changes, making them perfect for documentation, code reviews, and audit trails.

## Usage

### Basic Report Generation

```bash
# Generate a basic report
terraform plan | tplan -report

# This creates: report.md
```

### Report with Drift Detection

```bash
# Include git information in the report
terraform plan | tplan -report -drift

# This creates: report.md with git attribution
```

### Using JSON Format

```bash
# More detailed reports from JSON plans
terraform plan -json | tplan -report -drift
```

## Report Structure

### 1. Executive Summary

Provides a quick overview with counts:
- 🟢 Resources to create
- 🟡 Resources to update
- 🔴 Resources to delete
- 🔵 Resources to replace
- ❌ Errors (if any)
- ⚠️ Warnings (if any)

### 2. Table of Contents

Auto-generated navigation links to all sections

### 3. Resource Changes by Action

Resources are organized by action type with detailed information:

#### For Each Resource:
- **Resource Address** (e.g., `aws_instance.web`)
- **Details:**
  - Type
  - Provider
  - Module (if applicable)
  - Action
  - Reason (for replacements)
- **Git Information** (with `-drift` flag):
  - File path
  - Commit ID
  - Branch name
  - Author name and email
  - Commit date
  - Commit message
  - Uncommitted changes warning
- **Attribute Changes:**
  - For **creates**: All new attributes
  - For **deletes**: All removed attributes
  - For **updates/replaces**: Before/After comparison table

### 4. Errors Section

Details of any errors encountered during planning

### 5. Warnings Section

List of warnings with context

## Example Report

### Without Drift Detection

```markdown
# Terraform Plan Report

**Generated:** 2024-12-08 14:10:14 MST
**Terraform Version:** 1.5.0

## Executive Summary

| Action | Count |
|--------|-------|
| 🟢 Create | 3 |
| 🟡 Update | 2 |
| 🔴 Delete | 1 |
| **Total Changes** | **6** |

## Resources to Create

### 1. aws_instance.web

**Details:**
- **Type:** `aws_instance`
- **Provider:** `registry.terraform.io/hashicorp/aws`
- **Action:** `create`

**Attributes:**
```hcl
ami = ami-abc123
instance_type = t3.micro
tags = map[Name:web-server]
```
```

### With Drift Detection

```markdown
### 1. aws_s3_bucket.data

**Details:**
- **Type:** `aws_s3_bucket`
- **Provider:** `registry.terraform.io/hashicorp/aws`
- **Action:** `update`

**Git Information:**
- **File:** `modules/storage/main.tf`
- **Commit:** `a1b2c3d4`
- **Branch:** `main`
- **Author:** Jane Doe <jane@example.com>
- **Date:** 2024-12-07 10:30:00
- **Commit Message:** Update S3 bucket configuration

**Changes:**

| Attribute | Before | After |
|-----------|--------|-------|
| `tags` | `map[Environment:dev]` | `map[Environment:prod]` |
```

## Use Cases

### 1. Code Review Documentation

Generate reports for pull requests:

```bash
# In your PR workflow
terraform plan -json > plan.json
cat plan.json | tplan -report -drift
git add report.md
git commit -m "Add Terraform plan report"
```

**Benefits:**
- Reviewers can see exactly what will change
- Git attribution shows who last modified each resource
- Clear, formatted documentation in PR

### 2. Audit Trail

Create permanent records of infrastructure changes:

```bash
# Before applying changes
terraform plan | tplan -report -drift
mv report.md "reports/plan-$(date +%Y%m%d-%H%M%S).md"
git add reports/
git commit -m "Document infrastructure changes"
```

### 3. Change Approval Process

Generate reports for approval workflows:

```bash
# Generate report
terraform plan -json | tplan -report -drift

# Email or attach report.md for approval
# After approval, apply changes
```

### 4. Documentation

Keep infrastructure change history:

```bash
# Generate report with timestamp
terraform plan | tplan -report
mv report.md "docs/changes/$(date +%Y-%m-%d)-infrastructure-update.md"
```

## Report Features

### Organized by Action Type

Resources are grouped logically:
1. **Creates** - New infrastructure
2. **Updates** - Modified existing resources
3. **Deletes** - Resources to remove
4. **Replaces** - Resources to recreate

### Detailed Attribute Changes

For updates and replacements, see exactly what changed:
- Before/After comparison tables
- Highlights new, modified, and removed attributes
- Truncates long values for readability

### Git Attribution (with `-drift`)

Understand the context of changes:
- Which file defines each resource
- Who last modified it
- When it was changed
- Why (commit message)

### Professional Formatting

- Markdown format for easy viewing on GitHub/GitLab
- Code blocks for attributes
- Tables for comparisons
- Emoji indicators for quick scanning
- Auto-generated table of contents

## Integration Examples

### GitHub Actions

```yaml
name: Terraform Plan Report

on: [pull_request]

jobs:
  plan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Terraform
        uses: hashicorp/setup-terraform@v2
      
      - name: Setup tplan
        run: |
          wget https://github.com/yourusername/tplan/releases/latest/download/tplan-linux-amd64.tar.gz
          tar -xzf tplan-linux-amd64.tar.gz
          chmod +x tplan-linux-amd64
          sudo mv tplan-linux-amd64 /usr/local/bin/tplan
      
      - name: Terraform Plan and Report
        run: |
          terraform init
          terraform plan -json | tplan -report -drift
      
      - name: Upload Report
        uses: actions/upload-artifact@v3
        with:
          name: terraform-plan-report
          path: report.md
      
      - name: Comment PR
        uses: actions/github-script@v6
        with:
          script: |
            const fs = require('fs');
            const report = fs.readFileSync('report.md', 'utf8');
            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: '## Terraform Plan Report\n\n' + report
            });
```

### GitLab CI

```yaml
terraform_plan_report:
  stage: plan
  script:
    - terraform init
    - terraform plan -json | tplan -report -drift
  artifacts:
    paths:
      - report.md
    expire_in: 1 week
```

### Jenkins Pipeline

```groovy
pipeline {
    agent any
    stages {
        stage('Plan') {
            steps {
                sh 'terraform init'
                sh 'terraform plan -json | tplan -report -drift'
                archiveArtifacts artifacts: 'report.md'
            }
        }
    }
}
```

## Output File

The report is always saved as `report.md` in the current directory.

**Note:** If `report.md` already exists, it will be overwritten.

### Custom Output Location

To save to a different location:

```bash
# Generate in current directory, then move
terraform plan | tplan -report
mv report.md /path/to/custom/location/

# Or use shell redirection (future feature)
```

## Comparison: TUI vs Report

| Feature | TUI Mode | Report Mode |
|---------|----------|-------------|
| **Interactive** | ✓ | ✗ |
| **Navigation** | ✓ | ✗ |
| **File Output** | ✗ | ✓ |
| **CI/CD Friendly** | ✗ | ✓ |
| **Documentation** | ✗ | ✓ |
| **Quick Review** | ✓ | ✗ |
| **Git Attribution** | ✓ (with -drift) | ✓ (with -drift) |
| **Sharing** | ✗ | ✓ |

**Use TUI when:**
- You want to quickly review changes interactively
- You're working locally on your terminal
- You want to navigate and explore resources

**Use Report when:**
- You need documentation for reviews/audits
- Running in CI/CD pipelines
- Sharing plans with non-technical stakeholders
- Creating permanent records

## Tips

### Combining Both Modes

You can review interactively first, then generate a report:

```bash
# 1. Review in TUI
terraform plan | tee plan.txt | tplan -drift

# 2. Generate report from saved plan
cat plan.txt | tplan -report -drift
```

### Report Customization

While reports use a standard format, you can post-process them:

```bash
# Add custom header
echo "# Custom Project Name" > final-report.md
echo "" >> final-report.md
cat report.md >> final-report.md

# Convert to PDF (requires pandoc)
pandoc report.md -o report.pdf
```

### Version Control

Include reports in version control for history:

```bash
mkdir -p docs/terraform-plans
terraform plan | tplan -report -drift
mv report.md "docs/terraform-plans/$(date +%Y-%m-%d)-plan.md"
git add docs/terraform-plans/
git commit -m "Add Terraform plan for infrastructure update"
```

## Troubleshooting

### Empty Report

**Issue:** Report is generated but has no resources

**Solution:** Ensure your Terraform plan has changes:
```bash
# Check plan output first
terraform plan | tee plan.txt
# Verify it shows changes
# Then generate report
cat plan.txt | tplan -report
```

### Missing Git Information

**Issue:** Report generated with `-drift` but no git info shown

**Solution:** 
- Ensure you're in a git repository
- Ensure `.tf` files are committed
- Check that resources can be found in `.tf` files

### Truncated Attributes

**Issue:** Long attribute values are truncated with `...`

**Solution:** This is by design for readability. Check the original plan or TUI mode for full values.

## Future Enhancements

Potential improvements:
- Custom output filename
- HTML output format
- PDF generation
- Custom templates
- Filtering (only show certain resources)
- Diff highlighting
- Summary email format

