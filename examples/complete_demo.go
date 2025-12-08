package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/yourusername/tplan/internal/git"
	"github.com/yourusername/tplan/internal/models"
	"github.com/yourusername/tplan/internal/parser"
	"github.com/yourusername/tplan/internal/tui"
)

// This is a complete demo showing how tplan works end-to-end
func main() {
	fmt.Println("=== tplan Complete Demo ===\n")

	// Create sample Terraform plan output (JSON format)
	samplePlan := createSamplePlanJSON()

	// Step 1: Parse the plan
	fmt.Println("Step 1: Parsing Terraform plan...")
	p := parser.NewParser()
	planResult, err := p.Parse(strings.NewReader(samplePlan))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing plan: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  ✓ Parsed %d resource changes\n", len(planResult.Resources))
	fmt.Printf("  ✓ Found %d errors, %d warnings\n\n", len(planResult.Errors), len(planResult.Warnings))

	// Step 2: Add git information (simulate drift mode)
	fmt.Println("Step 2: Adding git information (drift mode)...")
	if err := enrichWithGitInfo(planResult); err != nil {
		fmt.Printf("  ⚠ Git info not available: %v\n\n", err)
	} else {
		fmt.Println("  ✓ Git information added\n")
	}

	// Step 3: Display summary
	fmt.Println("Step 3: Plan Summary:")
	fmt.Printf("  Creates:  %d\n", planResult.Summary.ToCreate)
	fmt.Printf("  Updates:  %d\n", planResult.Summary.ToUpdate)
	fmt.Printf("  Deletes:  %d\n", planResult.Summary.ToDelete)
	fmt.Printf("  Replaces: %d\n\n", planResult.Summary.ToReplace)

	// Step 4: Show resource details
	fmt.Println("Step 4: Resource Details:")
	for _, res := range planResult.Resources {
		fmt.Printf("  [%s] %s\n", res.Action, res.Address)
		if res.DriftInfo != nil && res.DriftInfo.IsValid() {
			fmt.Printf("    Git: %s @ %s (by %s)\n",
				res.DriftInfo.FilePath,
				res.DriftInfo.ShortCommitID(),
				res.DriftInfo.AuthorName)
		}
	}
	fmt.Println()

	// Step 5: Launch TUI
	fmt.Println("Step 5: Launching TUI...")
	fmt.Println("  Press Enter to continue (or Ctrl+C to skip)...")
	fmt.Scanln()

	if err := tui.Run(planResult); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n=== Demo Complete ===")
}

func createSamplePlanJSON() string {
	return `{
  "format_version": "1.0",
  "terraform_version": "1.5.0",
  "resource_changes": [
    {
      "address": "aws_instance.web",
      "mode": "managed",
      "type": "aws_instance",
      "name": "web",
      "provider_name": "registry.terraform.io/hashicorp/aws",
      "change": {
        "actions": ["create"],
        "before": null,
        "after": {
          "ami": "ami-12345678",
          "instance_type": "t2.micro",
          "tags": {
            "Name": "web-server"
          }
        }
      }
    },
    {
      "address": "aws_security_group.allow_http",
      "mode": "managed",
      "type": "aws_security_group",
      "name": "allow_http",
      "provider_name": "registry.terraform.io/hashicorp/aws",
      "change": {
        "actions": ["update"],
        "before": {
          "ingress": [
            {
              "from_port": 80,
              "to_port": 80,
              "protocol": "tcp"
            }
          ]
        },
        "after": {
          "ingress": [
            {
              "from_port": 80,
              "to_port": 80,
              "protocol": "tcp"
            },
            {
              "from_port": 443,
              "to_port": 443,
              "protocol": "tcp"
            }
          ]
        }
      }
    },
    {
      "address": "aws_s3_bucket.old_bucket",
      "mode": "managed",
      "type": "aws_s3_bucket",
      "name": "old_bucket",
      "provider_name": "registry.terraform.io/hashicorp/aws",
      "change": {
        "actions": ["delete"],
        "before": {
          "bucket": "my-old-bucket",
          "acl": "private"
        },
        "after": null
      }
    }
  ]
}`
}

func enrichWithGitInfo(planResult *models.PlanResult) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	repo, err := git.NewRepository(cwd)
	if err != nil {
		return err
	}

	// Try to get git info for each resource
	for i := range planResult.Resources {
		resource := &planResult.Resources[i]

		driftInfo, err := repo.GetDriftInfo(resource.Address)
		if err != nil {
			// Create mock drift info for demo purposes
			driftInfo = &models.DriftInfo{
				ResourceName:          resource.Address,
				FilePath:              "main.tf",
				CommitID:              "a1b2c3d4e5f6g7h8",
				BranchName:            "main",
				AuthorName:            "Jane Doe",
				AuthorEmail:           "jane@example.com",
				CommitDate:            time.Now().Add(-24 * time.Hour),
				CommitMessage:         "Update infrastructure configuration",
				IsTracked:             true,
				HasUncommittedChanges: false,
			}
		}

		resource.DriftInfo = driftInfo
	}

	return nil
}
