package main

import (
	"fmt"
	"os"

	"github.com/yourusername/tplan/internal/models"
	"github.com/yourusername/tplan/internal/parser"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run test_parser.go <plan_file>")
		fmt.Println("   or: cat <plan_file> | go run test_parser.go -")
		os.Exit(1)
	}

	p := parser.NewParser()

	var planResult *models.PlanResult
	var err error

	if os.Args[1] == "-" {
		// Read from stdin
		planResult, err = p.Parse(os.Stdin)
	} else {
		// Read from file
		planResult, err = parser.ParseFile(os.Args[1])
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Print summary
	fmt.Println("=== Terraform Plan Parser Results ===")
	fmt.Printf("\nInput Format: %s\n", planResult.InputFormat)
	fmt.Printf("Terraform Version: %s\n", planResult.TerraformVersion)
	fmt.Printf("Format Version: %s\n", planResult.FormatVersion)

	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("Total Resources: %d\n", planResult.Summary.Total)
	fmt.Printf("  To Create:  %d\n", planResult.Summary.ToCreate)
	fmt.Printf("  To Update:  %d\n", planResult.Summary.ToUpdate)
	fmt.Printf("  To Delete:  %d\n", planResult.Summary.ToDelete)
	fmt.Printf("  To Replace: %d\n", planResult.Summary.ToReplace)
	fmt.Printf("  No Change:  %d\n", planResult.Summary.NoOp)

	fmt.Printf("\n=== Resource Changes ===\n")
	for i, rc := range planResult.Resources {
		fmt.Printf("\n%d. [%s] %s\n", i+1, rc.Action, rc.Address)
		fmt.Printf("   Type: %s\n", rc.Type)
		fmt.Printf("   Name: %s\n", rc.Name)
		fmt.Printf("   Mode: %s\n", rc.Mode)
		if rc.Module != "" {
			fmt.Printf("   Module: %s\n", rc.Module)
		}
		if rc.ProviderName != "" {
			fmt.Printf("   Provider: %s\n", rc.ProviderName)
		}
		if rc.ActionReason != "" {
			fmt.Printf("   Reason: %s\n", rc.ActionReason)
		}

		if len(rc.Change.Before) > 0 {
			fmt.Printf("   Before: %d attributes\n", len(rc.Change.Before))
		}
		if len(rc.Change.After) > 0 {
			fmt.Printf("   After: %d attributes\n", len(rc.Change.After))
		}
	}

	if len(planResult.OutputChanges) > 0 {
		fmt.Printf("\n=== Output Changes ===\n")
		for i, oc := range planResult.OutputChanges {
			fmt.Printf("\n%d. %s\n", i+1, oc.Name)
			if oc.Sensitive {
				fmt.Printf("   (sensitive)\n")
			}
			fmt.Printf("   Actions: %v\n", oc.Change.Actions)
		}
	}

	if len(planResult.Errors) > 0 {
		fmt.Printf("\n=== Errors ===\n")
		for i, e := range planResult.Errors {
			fmt.Printf("%d. [%s] %s\n", i+1, e.Severity, e.Message)
			if e.Resource != "" {
				fmt.Printf("   Resource: %s\n", e.Resource)
			}
		}
	}

	if len(planResult.Warnings) > 0 {
		fmt.Printf("\n=== Warnings ===\n")
		for i, w := range planResult.Warnings {
			fmt.Printf("%d. %s\n", i+1, w.Message)
			if w.Resource != "" {
				fmt.Printf("   Resource: %s\n", w.Resource)
			}
		}
	}

	if planResult.DriftDetected {
		fmt.Printf("\n=== Drift Detected ===\n")
		fmt.Printf("Drifted Resources: %d\n", len(planResult.DriftedResources))
		for i, dr := range planResult.DriftedResources {
			fmt.Printf("\n%d. %s\n", i+1, dr.Address)
			fmt.Printf("   Type: %s\n", dr.Type)
			fmt.Printf("   Reason: %s\n", dr.DriftReason)
		}
	}

	fmt.Println("\n=== Parse Complete ===")
}
