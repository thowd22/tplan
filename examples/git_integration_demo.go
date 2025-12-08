package main

import (
	"fmt"
	"log"
	"time"

	"github.com/yourusername/tplan/internal/git"
	"github.com/yourusername/tplan/internal/models"
)

// This example demonstrates complete usage of the git integration

func main() {
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║    Terraform Plan Git Integration - Complete Demo         ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Step 1: Initialize repository
	repo, err := git.NewRepository(".")
	if err != nil {
		log.Fatalf("Failed to initialize repository: %v", err)
	}

	// Step 2: Verify it's a git repository
	fmt.Println("📁 Repository Detection")
	fmt.Println("   " + repeat("─", 50))
	if repo.IsGitRepository() {
		fmt.Println("   ✓ Git repository detected")
		fmt.Printf("   ✓ Root: %s\n", repo.GetRepositoryRoot())
	} else {
		fmt.Println("   ✗ Not a git repository")
		return
	}
	fmt.Println()

	// Step 3: Demonstrate GetDriftInfo with multiple resources
	fmt.Println("🔍 Resource Drift Analysis")
	fmt.Println("   " + repeat("─", 50))
	
	resources := []string{
		"aws_instance.web",
		"aws_s3_bucket.data",
		"module.vpc.aws_subnet.private",
	}

	var validDrifts []*models.DriftInfo
	
	for _, resource := range resources {
		fmt.Printf("\n   Analyzing: %s\n", resource)
		driftInfo, err := repo.GetDriftInfo(resource)
		if err != nil {
			fmt.Printf("   ✗ Error: %v\n", err)
			continue
		}

		if driftInfo.IsValid() {
			validDrifts = append(validDrifts, driftInfo)
			printDriftInfo(driftInfo)
		} else {
			fmt.Printf("   ⚠ %s\n", driftInfo.Error)
		}
	}
	fmt.Println()

	// Step 4: Show detailed history for valid drifts
	if len(validDrifts) > 0 {
		fmt.Println("📜 Commit History")
		fmt.Println("   " + repeat("─", 50))
		
		for _, drift := range validDrifts {
			fmt.Printf("\n   File: %s\n", drift.FilePath)
			
			history, err := repo.GetFileHistory(drift.FilePath, 3)
			if err != nil {
				fmt.Printf("   ✗ Error getting history: %v\n", err)
				continue
			}

			for i, commit := range history {
				fmt.Printf("   %d. [%s] %s\n", i+1, commit.hash[:8], commit.message)
				fmt.Printf("      by %s on %s\n", 
					commit.authorName,
					commit.date.Format("Jan 02, 2006"),
				)
			}
		}
		fmt.Println()

		// Step 5: Show diffs
		fmt.Println("📝 Recent Changes")
		fmt.Println("   " + repeat("─", 50))
		
		for _, drift := range validDrifts {
			if drift.CommitID == "" {
				continue
			}

			fmt.Printf("\n   File: %s\n", drift.FilePath)
			fmt.Printf("   Commit: %s\n", drift.ShortCommitID())
			
			// Get diff for the last commit
			diff, err := repo.GetFileDiff(drift.FilePath, drift.CommitID+"^", drift.CommitID)
			if err != nil {
				fmt.Printf("   ✗ Error getting diff: %v\n", err)
				continue
			}

			if diff != "" {
				// Show first few lines of diff
				lines := splitLines(diff, 10)
				for _, line := range lines {
					fmt.Printf("   %s\n", line)
				}
				if len(lines) >= 10 {
					fmt.Println("   ... (truncated)")
				}
			} else {
				fmt.Println("   (No changes)")
			}
		}
		fmt.Println()
	}

	// Step 6: Summary report
	fmt.Println("📊 Summary")
	fmt.Println("   " + repeat("─", 50))
	fmt.Printf("   Resources analyzed: %d\n", len(resources))
	fmt.Printf("   Valid results: %d\n", len(validDrifts))
	fmt.Printf("   Errors: %d\n", len(resources)-len(validDrifts))
	fmt.Println()

	// Step 7: Show edge cases
	demonstrateEdgeCases(repo)
}

func printDriftInfo(info *models.DriftInfo) {
	fmt.Printf("   ✓ File: %s\n", info.FilePath)
	fmt.Printf("     Branch: %s\n", info.BranchName)
	fmt.Printf("     Commit: %s\n", info.ShortCommitID())
	fmt.Printf("     Author: %s <%s>\n", info.AuthorName, info.AuthorEmail)
	fmt.Printf("     Date: %s (%s ago)\n", 
		info.CommitDate.Format("2006-01-02 15:04:05"),
		formatDuration(time.Since(info.CommitDate)),
	)
	fmt.Printf("     Message: %s\n", info.CommitMessage)
	fmt.Printf("     Status: %s\n", info.StatusSummary())
}

func demonstrateEdgeCases(repo *git.Repository) {
	fmt.Println("🧪 Edge Cases")
	fmt.Println("   " + repeat("─", 50))
	
	// Test 1: Non-existent resource
	fmt.Println("\n   Test 1: Non-existent resource")
	drift, _ := repo.GetDriftInfo("nonexistent.resource")
	fmt.Printf("   Result: %s\n", drift.Error)
	
	// Test 2: Invalid resource format
	fmt.Println("\n   Test 2: Invalid resource format")
	drift, _ = repo.GetDriftInfo("invalid")
	fmt.Printf("   Result: %s\n", drift.Error)
	
	// Test 3: Module resource
	fmt.Println("\n   Test 3: Module resource (if exists)")
	drift, _ = repo.GetDriftInfo("module.test.aws_instance.example")
	if drift.IsValid() {
		fmt.Println("   ✓ Successfully handled module resource")
	} else {
		fmt.Printf("   Result: %s\n", drift.Error)
	}
	
	fmt.Println()
}

// Helper functions

func repeat(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}

func splitLines(s string, max int) []string {
	lines := []string{}
	current := ""
	count := 0
	
	for _, c := range s {
		if c == '\n' {
			lines = append(lines, current)
			current = ""
			count++
			if count >= max {
				break
			}
		} else {
			current += string(c)
		}
	}
	
	if current != "" && count < max {
		lines = append(lines, current)
	}
	
	return lines
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", mins)
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", hours)
	}
	days := int(d.Hours() / 24)
	if days == 1 {
		return "1 day"
	}
	return fmt.Sprintf("%d days", days)
}
