package main

import (
	"fmt"
	"os"
	
	"github.com/yourusername/tplan/internal/parser"
	"github.com/yourusername/tplan/internal/tui"
)

func main() {
	p := parser.NewParser()
	result, err := p.Parse(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Parse error: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Fprintf(os.Stderr, "Parsed %d resources\n", len(result.Resources))
	fmt.Fprintf(os.Stderr, "Summary: C:%d U:%d D:%d R:%d\n", 
		result.Summary.ToCreate,
		result.Summary.ToUpdate,
		result.Summary.ToDelete,
		result.Summary.ToReplace)
	
	// Create TUI model
	model := tui.NewModel(result)
	fmt.Fprintf(os.Stderr, "Model created with %d nodes\n", len(model.nodes))
	
	fmt.Fprintf(os.Stderr, "About to run TUI...\n")
	if err := tui.Run(result); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}
