package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/yourusername/tplan/internal/parser"
)

func main() {
	// Example 1: Parse from stdin
	fmt.Println("=== Terraform Plan Parser Example ===\n")

	// Example text plan output
	textPlan := `
Terraform will perform the following actions:

  # aws_instance.example will be created
  + resource "aws_instance" "example" {
      + ami                          = "ami-12345678"
      + instance_type                = "t2.micro"
      + id                           = (known after apply)
    }

  # aws_s3_bucket.data will be updated in-place
  ~ resource "aws_s3_bucket" "data" {
        id     = "my-bucket"
      ~ tags   = {
          - "old" = "value"
          + "new" = "value"
        }
    }

  # aws_security_group.old will be destroyed
  - resource "aws_security_group" "old" {
      - id   = "sg-12345"
      - name = "old-security-group"
    }

Plan: 1 to add, 1 to change, 1 to destroy.

Warning: This is a warning message
Error: Failed to load module
`

	// Parse the text plan
	p := parser.NewParser()
	result, err := p.Parse(strings.NewReader(textPlan))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing plan: %v\n", err)
		os.Exit(1)
	}

	// Display results
	fmt.Printf("Parsed Plan Summary:\n")
	fmt.Printf("  Format: %s\n", result.InputFormat)
	fmt.Printf("  Terraform Version: %s\n", result.TerraformVersion)
	fmt.Printf("  Total Resources: %d\n", result.Summary.Total)
	fmt.Printf("  To Create: %d\n", result.Summary.ToCreate)
	fmt.Printf("  To Update: %d\n", result.Summary.ToUpdate)
	fmt.Printf("  To Delete: %d\n", result.Summary.ToDelete)
	fmt.Printf("  To Replace: %d\n", result.Summary.ToReplace)
	fmt.Printf("\n")

	// Display resource changes
	fmt.Println("Resource Changes:")
	for i, rc := range result.Resources {
		fmt.Printf("  %d. [%s] %s\n", i+1, rc.Action, rc.Address)
		fmt.Printf("     Type: %s, Name: %s, Mode: %s\n", rc.Type, rc.Name, rc.Mode)
		if len(rc.Change.Before) > 0 {
			fmt.Printf("     Before attributes: %d\n", len(rc.Change.Before))
		}
		if len(rc.Change.After) > 0 {
			fmt.Printf("     After attributes: %d\n", len(rc.Change.After))
		}
	}
	fmt.Printf("\n")

	// Display errors and warnings
	if len(result.Errors) > 0 {
		fmt.Println("Errors:")
		for i, e := range result.Errors {
			fmt.Printf("  %d. %s\n", i+1, e.Message)
		}
		fmt.Printf("\n")
	}

	if len(result.Warnings) > 0 {
		fmt.Println("Warnings:")
		for i, w := range result.Warnings {
			fmt.Printf("  %d. %s\n", i+1, w.Message)
		}
		fmt.Printf("\n")
	}

	if result.DriftDetected {
		fmt.Println("Drift Detected!")
		fmt.Printf("  Drifted Resources: %d\n", len(result.DriftedResources))
		fmt.Printf("\n")
	}

	// Example 2: Parse JSON format
	jsonPlan := `{
		"format_version": "1.1",
		"terraform_version": "1.5.0",
		"resource_changes": [
			{
				"address": "aws_instance.web",
				"mode": "managed",
				"type": "aws_instance",
				"name": "web",
				"provider_name": "aws",
				"change": {
					"actions": ["create"],
					"before": null,
					"after": {
						"ami": "ami-abc123",
						"instance_type": "t3.micro"
					}
				}
			}
		]
	}`

	fmt.Println("\n=== Parsing JSON Plan ===\n")
	result2, err := parser.ParseString(jsonPlan)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON plan: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("JSON Plan Summary:\n")
	fmt.Printf("  Format Version: %s\n", result2.FormatVersion)
	fmt.Printf("  Terraform Version: %s\n", result2.TerraformVersion)
	fmt.Printf("  Total Resources: %d\n", result2.Summary.Total)
	fmt.Printf("  To Create: %d\n", result2.Summary.ToCreate)
	fmt.Printf("\n")

	fmt.Println("Success! Parser is working correctly.")
}
