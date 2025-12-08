package main

import (
	"fmt"
	"os"
	
	"github.com/yourusername/tplan/internal/parser"
)

func main() {
	data, err := os.ReadFile("test_real_plan.txt")
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}
	
	fmt.Printf("File size: %d bytes\n", len(data))
	
	p := parser.NewParser()
	result, err := p.Parse(os.Stdin)
	if err != nil {
		fmt.Printf("Parse error: %v\n", err)
		return
	}
	
	fmt.Printf("\nParse Results:\n")
	fmt.Printf("  Resources: %d\n", len(result.Resources))
	fmt.Printf("  Errors: %d\n", len(result.Errors))
	fmt.Printf("  Warnings: %d\n", len(result.Warnings))
	fmt.Printf("  Format: %s\n", result.InputFormat)
	fmt.Printf("  Version: %s\n", result.TerraformVersion)
	
	fmt.Printf("\nSummary:\n")
	fmt.Printf("  Create: %d\n", result.Summary.ToCreate)
	fmt.Printf("  Update: %d\n", result.Summary.ToUpdate)
	fmt.Printf("  Delete: %d\n", result.Summary.ToDelete)
	fmt.Printf("  Replace: %d\n", result.Summary.ToReplace)
	fmt.Printf("  Total: %d\n", result.Summary.Total)
	
	fmt.Printf("\nResources:\n")
	for i, r := range result.Resources {
		fmt.Printf("  %d. %s [%s]\n", i+1, r.Address, r.Action)
	}
}
