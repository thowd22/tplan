package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/yourusername/tplan/internal/models"
)

// Repository represents a git repository context
type Repository struct {
	rootPath string
	isRepo   bool
}

// NewRepository creates a new Repository instance and detects if the current directory is a git repo
func NewRepository(path string) (*Repository, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	repo := &Repository{
		rootPath: absPath,
	}

	// Check if this is a git repository
	repo.isRepo = repo.detectGitRepository()

	return repo, nil
}

// IsGitRepository returns true if the current directory is within a git repository
func (r *Repository) IsGitRepository() bool {
	return r.isRepo
}

// detectGitRepository checks if the current directory is a git repository
func (r *Repository) detectGitRepository() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = r.rootPath
	err := cmd.Run()
	return err == nil
}

// GetDriftInfo retrieves git information for a given resource address
// It attempts to find the Terraform file containing the resource and extract git metadata
func (r *Repository) GetDriftInfo(resourceAddress string) (*models.DriftInfo, error) {
	info := &models.DriftInfo{
		ResourceName: resourceAddress,
	}

	// If not a git repository, return early
	if !r.isRepo {
		info.Error = "Not a git repository"
		info.IsTracked = false
		return info, nil
	}

	// Find the Terraform file containing this resource
	filePath, err := r.findTerraformFile(resourceAddress)
	if err != nil {
		info.Error = fmt.Sprintf("Failed to find Terraform file: %v", err)
		return info, nil
	}

	info.FilePath = filePath

	// Check if file is tracked by git
	tracked, err := r.isFileTracked(filePath)
	if err != nil {
		info.Error = fmt.Sprintf("Failed to check git tracking: %v", err)
		return info, nil
	}

	info.IsTracked = tracked
	if !tracked {
		info.Error = "File not tracked by git"
		return info, nil
	}

	// Check for uncommitted changes
	hasChanges, err := r.hasUncommittedChanges(filePath)
	if err != nil {
		info.Error = fmt.Sprintf("Failed to check for uncommitted changes: %v", err)
		return info, nil
	}
	info.HasUncommittedChanges = hasChanges

	// Get the current branch
	branch, err := r.getCurrentBranch()
	if err != nil {
		info.Error = fmt.Sprintf("Failed to get branch: %v", err)
		return info, nil
	}
	info.BranchName = branch

	// Get the last commit that modified this file
	commitInfo, err := r.getLastCommitInfo(filePath)
	if err != nil {
		info.Error = fmt.Sprintf("Failed to get commit info: %v", err)
		return info, nil
	}

	// Populate commit information
	info.CommitID = commitInfo.hash
	info.AuthorName = commitInfo.authorName
	info.AuthorEmail = commitInfo.authorEmail
	info.CommitDate = commitInfo.date
	info.CommitMessage = commitInfo.message

	return info, nil
}

// findTerraformFile searches for the Terraform file containing the given resource address
func (r *Repository) findTerraformFile(resourceAddress string) (string, error) {
	// Parse resource address (e.g., "aws_instance.web" or "module.vpc.aws_subnet.private")
	parts := strings.Split(resourceAddress, ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid resource address format: %s", resourceAddress)
	}

	// Extract resource type and name
	// Handle module resources (e.g., module.vpc.aws_subnet.private)
	var resourceType, resourceName string
	if parts[0] == "module" {
		// For module resources, we need at least 4 parts: module.name.type.resource
		if len(parts) < 4 {
			return "", fmt.Errorf("invalid module resource address: %s", resourceAddress)
		}
		resourceType = parts[len(parts)-2]
		resourceName = parts[len(parts)-1]
	} else {
		resourceType = parts[0]
		resourceName = parts[1]
	}

	// Search for .tf files containing the resource definition
	tfFiles, err := r.findTerraformFiles()
	if err != nil {
		return "", err
	}

	// Search for the resource in each file
	for _, tfFile := range tfFiles {
		content, err := os.ReadFile(tfFile)
		if err != nil {
			continue
		}

		// Simple pattern matching - in production, you might want to use a proper HCL parser
		if strings.Contains(string(content), fmt.Sprintf(`resource "%s" "%s"`, resourceType, resourceName)) {
			return tfFile, nil
		}

		// Also check for single-quoted resources (less common but possible)
		if strings.Contains(string(content), fmt.Sprintf(`resource '%s' '%s'`, resourceType, resourceName)) {
			return tfFile, nil
		}
	}

	return "", fmt.Errorf("resource %s not found in any .tf file", resourceAddress)
}

// findTerraformFiles returns a list of all .tf files in the repository
func (r *Repository) findTerraformFiles() ([]string, error) {
	var tfFiles []string

	err := filepath.Walk(r.rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .terraform directory and other hidden directories
		if info.IsDir() && (strings.HasPrefix(info.Name(), ".") || info.Name() == ".terraform") {
			return filepath.SkipDir
		}

		// Check for .tf files
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".tf") {
			tfFiles = append(tfFiles, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return tfFiles, nil
}

// isFileTracked checks if a file is tracked by git
func (r *Repository) isFileTracked(filePath string) (bool, error) {
	relPath, err := filepath.Rel(r.rootPath, filePath)
	if err != nil {
		return false, err
	}

	cmd := exec.Command("git", "ls-files", "--error-unmatch", relPath)
	cmd.Dir = r.rootPath
	err = cmd.Run()

	return err == nil, nil
}

// hasUncommittedChanges checks if a file has uncommitted changes
func (r *Repository) hasUncommittedChanges(filePath string) (bool, error) {
	relPath, err := filepath.Rel(r.rootPath, filePath)
	if err != nil {
		return false, err
	}

	cmd := exec.Command("git", "status", "--porcelain", relPath)
	cmd.Dir = r.rootPath
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	// If output is not empty, there are uncommitted changes
	return len(bytes.TrimSpace(output)) > 0, nil
}

// getCurrentBranch returns the current git branch name
func (r *Repository) getCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = r.rootPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	branch := strings.TrimSpace(string(output))
	return branch, nil
}

// commitInfo holds information about a git commit
type commitInfo struct {
	hash        string
	authorName  string
	authorEmail string
	date        time.Time
	message     string
}

// getLastCommitInfo retrieves information about the last commit that modified the file
func (r *Repository) getLastCommitInfo(filePath string) (*commitInfo, error) {
	relPath, err := filepath.Rel(r.rootPath, filePath)
	if err != nil {
		return nil, err
	}

	// Format: hash|author name|author email|timestamp|commit message
	format := "%H|%an|%ae|%at|%s"
	cmd := exec.Command("git", "log", "-1", fmt.Sprintf("--format=%s", format), "--", relPath)
	cmd.Dir = r.rootPath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get commit info: %w", err)
	}

	line := strings.TrimSpace(string(output))
	if line == "" {
		return nil, fmt.Errorf("no commit history found for file")
	}

	parts := strings.SplitN(line, "|", 5)
	if len(parts) != 5 {
		return nil, fmt.Errorf("unexpected git log format")
	}

	// Parse timestamp
	var timestamp int64
	_, err = fmt.Sscanf(parts[3], "%d", &timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse commit timestamp: %w", err)
	}

	return &commitInfo{
		hash:        parts[0],
		authorName:  parts[1],
		authorEmail: parts[2],
		date:        time.Unix(timestamp, 0),
		message:     parts[4],
	}, nil
}

// GetFileHistory returns the full commit history for a file
func (r *Repository) GetFileHistory(filePath string, limit int) ([]commitInfo, error) {
	if !r.isRepo {
		return nil, fmt.Errorf("not a git repository")
	}

	relPath, err := filepath.Rel(r.rootPath, filePath)
	if err != nil {
		return nil, err
	}

	format := "%H|%an|%ae|%at|%s"
	args := []string{"log", fmt.Sprintf("--format=%s", format), "--"}
	if limit > 0 {
		args = append([]string{"log", fmt.Sprintf("-n%d", limit), fmt.Sprintf("--format=%s", format), "--"}, relPath)
	} else {
		args = append(args, relPath)
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = r.rootPath
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get file history: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	commits := make([]commitInfo, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "|", 5)
		if len(parts) != 5 {
			continue
		}

		var timestamp int64
		_, err = fmt.Sscanf(parts[3], "%d", &timestamp)
		if err != nil {
			continue
		}

		commits = append(commits, commitInfo{
			hash:        parts[0],
			authorName:  parts[1],
			authorEmail: parts[2],
			date:        time.Unix(timestamp, 0),
			message:     parts[4],
		})
	}

	return commits, nil
}

// GetFileDiff returns the diff of a file between two commits
func (r *Repository) GetFileDiff(filePath, fromCommit, toCommit string) (string, error) {
	if !r.isRepo {
		return "", fmt.Errorf("not a git repository")
	}

	relPath, err := filepath.Rel(r.rootPath, filePath)
	if err != nil {
		return "", err
	}

	var cmd *exec.Cmd
	if toCommit == "" {
		// Diff against working directory
		cmd = exec.Command("git", "diff", fromCommit, "--", relPath)
	} else {
		cmd = exec.Command("git", "diff", fromCommit, toCommit, "--", relPath)
	}

	cmd.Dir = r.rootPath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get diff: %w", err)
	}

	return string(output), nil
}

// GetRepositoryRoot returns the root path of the repository
func (r *Repository) GetRepositoryRoot() string {
	return r.rootPath
}

// Legacy functions for backward compatibility

// IsGitRepo checks if the current directory is a git repository
func IsGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	err := cmd.Run()
	return err == nil
}

// GetCurrentBranch returns the name of the current git branch
func GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// GetCommitHash returns the current commit hash
func GetCommitHash() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// GetCommitMessage returns the commit message for the given hash
func GetCommitMessage(hash string) (string, error) {
	cmd := exec.Command("git", "log", "-1", "--pretty=%B", hash)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
