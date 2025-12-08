package main

import (
	"fmt"
	"github.com/yourusername/tplan/internal/git"
)

func main() {
	// Test 1: Create repository
	fmt.Println("=== Git Integration Test ===\n")
	
	repo, err := git.NewRepository(".")
	if err != nil {
		fmt.Printf("Error creating repository: %v\n", err)
		return
	}
	
	// Test 2: Check if git repository
	fmt.Printf("Is Git Repository: %v\n", repo.IsGitRepository())
	
	if !repo.IsGitRepository() {
		fmt.Println("Not a git repository - skipping further tests")
		return
	}
	
	// Test 3: Test with a hypothetical resource
	fmt.Println("\n=== Testing GetDriftInfo ===")
	driftInfo, err := repo.GetDriftInfo("aws_instance.web")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		if driftInfo.IsValid() {
			fmt.Printf("✓ Resource: %s\n", driftInfo.ResourceName)
			fmt.Printf("✓ File: %s\n", driftInfo.FilePath)
			fmt.Printf("✓ Branch: %s\n", driftInfo.BranchName)
			fmt.Printf("✓ Commit: %s\n", driftInfo.ShortCommitID())
			fmt.Printf("✓ Author: %s <%s>\n", driftInfo.AuthorName, driftInfo.AuthorEmail)
			fmt.Printf("✓ Date: %s\n", driftInfo.CommitDate.Format("2006-01-02 15:04:05"))
			fmt.Printf("✓ Message: %s\n", driftInfo.CommitMessage)
			fmt.Printf("✓ Status: %s\n", driftInfo.StatusSummary())
		} else {
			fmt.Printf("✗ Error: %s\n", driftInfo.Error)
			fmt.Printf("  (This is expected if the resource doesn't exist)\n")
		}
	}
	
	// Test 4: Test legacy functions
	fmt.Println("\n=== Testing Legacy Functions ===")
	if git.IsGitRepo() {
		fmt.Println("✓ IsGitRepo() works")
	}
	
	branch, err := git.GetCurrentBranch()
	if err == nil {
		fmt.Printf("✓ Current branch: %s\n", branch)
	}
	
	hash, err := git.GetCommitHash()
	if err == nil {
		fmt.Printf("✓ Current commit: %s\n", hash[:8])
	}
	
	fmt.Println("\n=== All Tests Completed ===")
}
