package git

import (
	"fmt"
	"log"
)

// ExampleUsage demonstrates how to use the git integration
func ExampleUsage() {
	// Create a repository instance for the current directory
	repo, err := NewRepository(".")
	if err != nil {
		log.Fatalf("Failed to create repository: %v", err)
	}

	// Check if this is a git repository
	if !repo.IsGitRepository() {
		fmt.Println("Not a git repository")
		return
	}

	fmt.Println("Git repository detected!")

	// Get drift information for a specific resource
	resourceAddress := "aws_instance.web"
	driftInfo, err := repo.GetDriftInfo(resourceAddress)
	if err != nil {
		log.Fatalf("Failed to get drift info: %v", err)
	}

	// Display the drift information
	if driftInfo.IsValid() {
		fmt.Printf("\nDrift Information for %s:\n", driftInfo.ResourceName)
		fmt.Printf("  File: %s\n", driftInfo.FilePath)
		fmt.Printf("  Branch: %s\n", driftInfo.BranchName)
		fmt.Printf("  Last Commit: %s\n", driftInfo.ShortCommitID())
		fmt.Printf("  Author: %s <%s>\n", driftInfo.AuthorName, driftInfo.AuthorEmail)
		fmt.Printf("  Date: %s\n", driftInfo.CommitDate.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Message: %s\n", driftInfo.CommitMessage)
		fmt.Printf("  Status: %s\n", driftInfo.StatusSummary())
	} else {
		fmt.Printf("Error getting drift info: %s\n", driftInfo.Error)
	}

	// Get file history (last 5 commits)
	if driftInfo.IsValid() {
		fmt.Printf("\nRecent commit history for %s:\n", driftInfo.FilePath)
		history, err := repo.GetFileHistory(driftInfo.FilePath, 5)
		if err != nil {
			log.Printf("Failed to get file history: %v", err)
		} else {
			for i, commit := range history {
				fmt.Printf("  %d. %s - %s by %s (%s)\n",
					i+1,
					commit.hash[:8],
					commit.message,
					commit.authorName,
					commit.date.Format("2006-01-02"),
				)
			}
		}
	}

	// Get diff for the file
	if driftInfo.IsValid() && driftInfo.CommitID != "" {
		fmt.Printf("\nGetting diff for the last commit...\n")
		diff, err := repo.GetFileDiff(driftInfo.FilePath, driftInfo.CommitID+"^", driftInfo.CommitID)
		if err != nil {
			log.Printf("Failed to get diff: %v", err)
		} else {
			if diff != "" {
				fmt.Printf("Diff:\n%s\n", diff)
			} else {
				fmt.Println("No diff available")
			}
		}
	}
}
