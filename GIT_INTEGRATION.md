# Git Integration Documentation

This document describes the git integration functionality for tracking Terraform resource drift.

## Overview

The git integration provides functionality to:
1. Detect if the current directory is a git repository
2. Find Terraform files containing specific resources
3. Extract git metadata (commit ID, author, date, branch) for those files
4. Handle edge cases like untracked files and uncommitted changes

## Components

### 1. DriftInfo Struct (`internal/models/drift.go`)

The `DriftInfo` struct contains all git-related information about a drifted resource:

```go
type DriftInfo struct {
    ResourceName          string    // Terraform resource address (e.g., "aws_instance.web")
    FilePath              string    // Path to the Terraform file
    CommitID              string    // SHA of the last commit that modified the file
    BranchName            string    // Current git branch
    AuthorName            string    // Name of the commit author
    AuthorEmail           string    // Email of the commit author
    CommitDate            time.Time // When the last modifying commit was made
    CommitMessage         string    // Commit message of the last modifying commit
    DiffExplanation       string    // Human-readable explanation of what changed
    IsTracked             bool      // Whether the file is tracked by git
    HasUncommittedChanges bool      // Whether the file has local modifications
    Error                 string    // Any error message encountered
}
```

#### Helper Methods

- `IsValid() bool` - Returns true if drift info was successfully populated
- `ShortCommitID() string` - Returns first 8 characters of commit ID
- `StatusSummary() string` - Returns human-readable status

### 2. Repository Type (`internal/git/git.go`)

The `Repository` type provides all git operations:

```go
type Repository struct {
    rootPath string
    isRepo   bool
}
```

#### Main Functions

**NewRepository(path string) (*Repository, error)**
- Creates a new Repository instance
- Detects if the directory is a git repository
- Returns an error if the path cannot be resolved

**GetDriftInfo(resourceAddress string) (*DriftInfo, error)**
- Main function to get git information for a resource
- Handles all edge cases gracefully
- Returns DriftInfo with error details in the Error field

**GetFileHistory(filePath string, limit int) ([]commitInfo, error)**
- Returns commit history for a file
- Limit parameter controls how many commits to retrieve (0 = all)

**GetFileDiff(filePath, fromCommit, toCommit string) (string, error)**
- Returns the diff between two commits for a file
- If toCommit is empty, diffs against working directory

## Usage Examples

### Basic Usage

```go
package main

import (
    "fmt"
    "log"
    "github.com/yourusername/tplan/internal/git"
)

func main() {
    // Create a repository instance
    repo, err := git.NewRepository(".")
    if err != nil {
        log.Fatalf("Failed to create repository: %v", err)
    }

    // Check if it's a git repository
    if !repo.IsGitRepository() {
        fmt.Println("Not a git repository")
        return
    }

    // Get drift information for a resource
    driftInfo, err := repo.GetDriftInfo("aws_instance.web")
    if err != nil {
        log.Fatalf("Failed to get drift info: %v", err)
    }

    // Check if the information was successfully retrieved
    if driftInfo.IsValid() {
        fmt.Printf("Resource: %s\n", driftInfo.ResourceName)
        fmt.Printf("File: %s\n", driftInfo.FilePath)
        fmt.Printf("Branch: %s\n", driftInfo.BranchName)
        fmt.Printf("Commit: %s\n", driftInfo.ShortCommitID())
        fmt.Printf("Author: %s <%s>\n", driftInfo.AuthorName, driftInfo.AuthorEmail)
        fmt.Printf("Date: %s\n", driftInfo.CommitDate)
        fmt.Printf("Message: %s\n", driftInfo.CommitMessage)
    } else {
        fmt.Printf("Error: %s\n", driftInfo.Error)
    }
}
```

### Getting File History

```go
// Get last 5 commits for a file
history, err := repo.GetFileHistory("/path/to/file.tf", 5)
if err != nil {
    log.Fatalf("Failed to get history: %v", err)
}

for i, commit := range history {
    fmt.Printf("%d. %s - %s by %s\n",
        i+1,
        commit.hash[:8],
        commit.message,
        commit.authorName,
    )
}
```

### Getting File Diff

```go
// Get diff between two commits
diff, err := repo.GetFileDiff("/path/to/file.tf", "abc123", "def456")
if err != nil {
    log.Fatalf("Failed to get diff: %v", err)
}
fmt.Println(diff)

// Get diff between commit and working directory
diff, err = repo.GetFileDiff("/path/to/file.tf", "abc123", "")
if err != nil {
    log.Fatalf("Failed to get diff: %v", err)
}
fmt.Println(diff)
```

## Resource Address Formats

The git integration handles various Terraform resource address formats:

1. **Simple resources**: `aws_instance.web`
2. **Module resources**: `module.vpc.aws_subnet.private`
3. **Nested modules**: `module.networking.module.vpc.aws_instance.app`

## Edge Cases Handled

### 1. Not a Git Repository
```go
driftInfo, _ := repo.GetDriftInfo("aws_instance.web")
// driftInfo.Error = "Not a git repository"
// driftInfo.IsTracked = false
```

### 2. File Not Tracked by Git
```go
driftInfo, _ := repo.GetDriftInfo("aws_instance.web")
// driftInfo.Error = "File not tracked by git"
// driftInfo.IsTracked = false
```

### 3. Uncommitted Changes
```go
driftInfo, _ := repo.GetDriftInfo("aws_instance.web")
// driftInfo.HasUncommittedChanges = true
// driftInfo.IsValid() returns true
// StatusSummary() returns "Has uncommitted changes"
```

### 4. Resource Not Found
```go
driftInfo, _ := repo.GetDriftInfo("nonexistent_resource.test")
// driftInfo.Error = "Failed to find Terraform file: resource nonexistent_resource.test not found in any .tf file"
```

### 5. No Commit History
```go
// For newly added files with no commits
driftInfo, _ := repo.GetDriftInfo("aws_instance.web")
// driftInfo.Error = "Failed to get commit info: no commit history found for file"
```

## Implementation Details

### Finding Terraform Files

The `findTerraformFile` function:
1. Parses the resource address to extract type and name
2. Handles module resources correctly
3. Searches all .tf files in the repository
4. Skips .terraform and hidden directories
5. Uses string matching to find resource definitions

### Git Commands Used

- `git rev-parse --git-dir` - Detect git repository
- `git ls-files --error-unmatch <file>` - Check if file is tracked
- `git status --porcelain <file>` - Check for uncommitted changes
- `git rev-parse --abbrev-ref HEAD` - Get current branch
- `git log -1 --format="%H|%an|%ae|%at|%s" -- <file>` - Get commit info
- `git log --format="%H|%an|%ae|%at|%s" -- <file>` - Get commit history
- `git diff <commit1> <commit2> -- <file>` - Get file diff

### Error Handling

All functions handle errors gracefully:
- Errors are returned in the DriftInfo.Error field
- No panics or fatal errors
- Allows calling code to decide how to handle issues
- IsValid() provides a simple way to check for success

## Testing

To test the git integration:

1. Create a test Terraform file:
```bash
cat > test.tf << 'EOF'
resource "aws_instance" "web" {
  ami           = "ami-12345678"
  instance_type = "t2.micro"
}
EOF
```

2. Commit the file:
```bash
git add test.tf
git commit -m "Add test instance"
```

3. Run the example:
```go
repo, _ := git.NewRepository(".")
driftInfo, _ := repo.GetDriftInfo("aws_instance.web")
fmt.Printf("%+v\n", driftInfo)
```

## Future Enhancements

Potential improvements for the git integration:

1. **HCL Parsing**: Use a proper HCL parser instead of string matching
2. **Performance**: Cache Terraform file locations
3. **Diff Analysis**: Parse git diffs to generate DiffExplanation
4. **Blame Information**: Add git blame support for line-level tracking
5. **Remote Tracking**: Track remote branches and upstream changes
6. **Multi-file Resources**: Handle resources split across multiple files
7. **Submodules**: Support for git submodules
8. **Monorepos**: Better handling of monorepo structures

## License

Same as the parent project.
