package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/yourusername/tplan/internal/git"
	"github.com/yourusername/tplan/internal/models"
	"github.com/yourusername/tplan/internal/parser"
	"github.com/yourusername/tplan/internal/report"
	"github.com/yourusername/tplan/internal/tui"
)

const version = "1.0.0"

func main() {
	// Parse command-line flags
	driftMode := flag.Bool("drift", false, "Enable drift detection and git integration")
	reportMode := flag.Bool("report", false, "Generate a Markdown report (report.md)")
	versionFlag := flag.Bool("version", false, "Show version information")
	flag.BoolVar(versionFlag, "v", false, "Show version information")
	help := flag.Bool("help", false, "Show help message")
	flag.BoolVar(help, "h", false, "Show help message")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("tplan version %s\n", version)
		os.Exit(0)
	}

	if *help {
		printHelp()
		os.Exit(0)
	}

	// Parse Terraform plan from stdin
	p := parser.NewParser()
	planResult, err := p.Parse(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing Terraform plan: %v\n", err)
		fmt.Fprintf(os.Stderr, "\nUsage: terraform plan | tplan\n")
		fmt.Fprintf(os.Stderr, "   or: terraform plan -json | tplan\n")
		os.Exit(1)
	}

	// If drift mode is enabled, enhance the plan with git information
	if *driftMode {
		if err := enrichWithGitInfo(planResult); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Could not get git information: %v\n", err)
			// Continue anyway - we'll show the plan without git info
		}
	}

	// If report mode is enabled, generate the report and exit
	if *reportMode {
		if err := generateReport(planResult, *driftMode); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating report: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✓ Report generated: report.md")
		os.Exit(0)
	}

	// Run the TUI
	if err := tui.Run(planResult); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}

func generateReport(planResult *models.PlanResult, includeDrift bool) error {
	gen := report.NewGenerator(planResult, includeDrift)
	return gen.WriteToFile("report.md")
}

func enrichWithGitInfo(planResult *models.PlanResult) error {
	// Get current directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Initialize git repository
	repo, err := git.NewRepository(cwd)
	if err != nil {
		return fmt.Errorf("failed to initialize git repository: %w", err)
	}

	// For each resource change, try to get git information
	for i := range planResult.Resources {
		resource := &planResult.Resources[i]

		// Get drift info for this resource
		driftInfo, err := repo.GetDriftInfo(resource.Address)
		if err != nil {
			// Not a critical error - just skip this resource
			continue
		}

		// Attach the drift info to the resource
		resource.DriftInfo = driftInfo
	}

	return nil
}

func printHelp() {
	fmt.Println("tplan - Terraform Plan TUI Viewer")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("  terraform plan | tplan [OPTIONS]")
	fmt.Println("  terraform plan -json | tplan [OPTIONS]")
	fmt.Println()
	fmt.Println("OPTIONS:")
	fmt.Println("  -drift        Enable drift detection with git integration")
	fmt.Println("                Shows git commit, branch, and author info for drifted resources")
	fmt.Println("  -report       Generate a Markdown report (report.md) and exit")
	fmt.Println("                Use with -drift to include git information in the report")
	fmt.Println("  -v, -version  Show version information")
	fmt.Println("  -h, -help     Show this help message")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  # Basic usage")
	fmt.Println("  terraform plan | tplan")
	fmt.Println()
	fmt.Println("  # With drift detection")
	fmt.Println("  terraform plan | tplan -drift")
	fmt.Println()
	fmt.Println("  # Generate report")
	fmt.Println("  terraform plan | tplan -report")
	fmt.Println()
	fmt.Println("  # Generate report with drift information")
	fmt.Println("  terraform plan | tplan -report -drift")
	fmt.Println()
	fmt.Println("  # Using JSON format")
	fmt.Println("  terraform plan -json | tplan -drift")
	fmt.Println()
	fmt.Println("KEYBOARD CONTROLS:")
	fmt.Println("  ↑/↓, j/k      Navigate up/down")
	fmt.Println("  Enter, Space  Expand/collapse resource")
	fmt.Println("  e             Expand all")
	fmt.Println("  c             Collapse all")
	fmt.Println("  Tab           Switch between Changes/Errors/Warnings")
	fmt.Println("  g             Jump to top")
	fmt.Println("  G             Jump to bottom")
	fmt.Println("  q, Esc        Quit")
	fmt.Println()
}
