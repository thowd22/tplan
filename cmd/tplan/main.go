package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/yourusername/tplan/internal/git"
	"github.com/yourusername/tplan/internal/models"
	"github.com/yourusername/tplan/internal/parser"
	"github.com/yourusername/tplan/internal/report"
	"github.com/yourusername/tplan/internal/tui"
)

// Version is set via ldflags during build
var Version = "dev"

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
		fmt.Printf("tplan version %s\n", Version)
		os.Exit(0)
	}

	if *help {
		printHelp()
		os.Exit(0)
	}

	// Check if terraform or tofu is installed
	tfCmd := findTerraformCommand()
	if tfCmd == "" {
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "═══════════════════════════════════════════════════════════════\n")
		fmt.Fprintf(os.Stderr, "  ERROR: Neither Terraform nor OpenTofu is installed\n")
		fmt.Fprintf(os.Stderr, "═══════════════════════════════════════════════════════════════\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "tplan requires either Terraform or OpenTofu to be installed.\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Install Terraform:\n")
		fmt.Fprintf(os.Stderr, "  https://developer.hashicorp.com/terraform/install\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Or install OpenTofu:\n")
		fmt.Fprintf(os.Stderr, "  https://opentofu.org/docs/intro/install/\n")
		fmt.Fprintf(os.Stderr, "\n")
		os.Exit(1)
	}

	fmt.Printf("Using: %s\n", tfCmd)

	// Create temporary plan file
	planFile := filepath.Join(".", ".tplan-temp.tfplan")

	// Ensure cleanup on exit
	defer func() {
		if err := os.Remove(planFile); err != nil && !os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Warning: failed to clean up temp file %s: %v\n", planFile, err)
		}
	}()

	// Get any additional arguments to pass to terraform plan
	planArgs := flag.Args()

	// Run terraform plan -out=<planfile>
	fmt.Printf("\nRunning: %s plan -out=%s", tfCmd, planFile)
	if len(planArgs) > 0 {
		fmt.Printf(" %v", planArgs)
	}
	fmt.Println()

	if err := runTerraformPlan(tfCmd, planFile, planArgs); err != nil {
		fmt.Fprintf(os.Stderr, "\nError running terraform plan: %v\n", err)
		os.Exit(1)
	}

	// Run terraform show -json <planfile>
	fmt.Println("\nGenerating JSON output...")
	jsonOutput, err := runTerraformShow(tfCmd, planFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating JSON output: %v\n", err)
		os.Exit(1)
	}

	// Parse the JSON output
	p := parser.NewParser()
	planResult, err := p.ParseBytes(jsonOutput)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing plan: %v\n", err)
		os.Exit(1)
	}

	// Always enrich with file information for grouping
	// This populates the FilePath in DriftInfo even without full drift mode
	if err := enrichWithFileInfo(planResult, *driftMode); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not get file information: %v\n", err)
		// Continue anyway - we'll show the plan without file info
	}

	// If report mode is enabled, generate the report and exit
	if *reportMode {
		if err := generateReport(planResult, *driftMode); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating report: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("\n✓ Report generated: report.md")
		os.Exit(0)
	}

	// Run the TUI
	fmt.Println("\nLaunching TUI...")
	if err := tui.Run(planResult); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}

// findTerraformCommand checks for terraform or tofu and returns the command to use
func findTerraformCommand() string {
	// Check for terraform first
	if _, err := exec.LookPath("terraform"); err == nil {
		return "terraform"
	}

	// Check for tofu as fallback
	if _, err := exec.LookPath("tofu"); err == nil {
		return "tofu"
	}

	return ""
}

// runTerraformPlan runs terraform/tofu plan and saves to a file
func runTerraformPlan(tfCmd, planFile string, extraArgs []string) error {
	args := []string{"plan", "-out=" + planFile}
	args = append(args, extraArgs...)

	cmd := exec.Command(tfCmd, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// runTerraformShow runs terraform/tofu show -json and returns the output
func runTerraformShow(tfCmd, planFile string) ([]byte, error) {
	var stdout bytes.Buffer

	cmd := exec.Command(tfCmd, "show", "-json", planFile)
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	return stdout.Bytes(), nil
}

func generateReport(planResult *models.PlanResult, includeDrift bool) error {
	gen := report.NewGenerator(planResult, includeDrift)
	return gen.WriteToFile("report.md")
}

func enrichWithFileInfo(planResult *models.PlanResult, fullDriftMode bool) error {
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

	// For each resource change, try to get git/file information
	for i := range planResult.Resources {
		resource := &planResult.Resources[i]

		// Get full drift info for this resource (includes file path and git info)
		driftInfo, err := repo.GetDriftInfo(resource.Address)
		if err != nil {
			// Not a critical error - just skip this resource
			continue
		}

		// Always attach the full drift info
		// This provides file grouping and git information
		resource.DriftInfo = driftInfo
	}

	// Second pass: for deleted resources without file info, try to find their replacement
	for i := range planResult.Resources {
		resource := &planResult.Resources[i]

		// Only process deleted resources without drift info
		if resource.Action != models.ActionDelete || resource.DriftInfo != nil {
			continue
		}

		// Look for a create operation with the same type and index
		for j := range planResult.Resources {
			other := &planResult.Resources[j]

			// Check if this is a potential replacement (same type, same index, create action)
			if other.Action == models.ActionCreate &&
				other.Type == resource.Type &&
				indexMatches(other.Index, resource.Index) &&
				other.DriftInfo != nil {
				// Copy the drift info from the replacement
				resource.DriftInfo = other.DriftInfo
				break
			}
		}
	}

	return nil
}

// indexMatches checks if two resource indices match
func indexMatches(idx1, idx2 interface{}) bool {
	// Handle nil cases
	if idx1 == nil && idx2 == nil {
		return true
	}
	if idx1 == nil || idx2 == nil {
		return false
	}

	// Compare as strings to handle both int and string indices
	return fmt.Sprintf("%v", idx1) == fmt.Sprintf("%v", idx2)
}

func printHelp() {
	fmt.Println("tplan - Terraform Plan TUI Viewer")
	fmt.Println()
	fmt.Println("A wrapper around terraform/tofu that runs plan and displays results")
	fmt.Println("in an interactive terminal UI.")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("  tplan [OPTIONS] [TERRAFORM_ARGS...]")
	fmt.Println()
	fmt.Println("  tplan runs 'terraform plan' (or 'tofu plan'), captures the output,")
	fmt.Println("  and displays it in an interactive TUI.")
	fmt.Println()
	fmt.Println("OPTIONS:")
	fmt.Println("  -drift        Enable drift detection with git integration")
	fmt.Println("                Shows git commit, branch, and author info for resources")
	fmt.Println("  -report       Generate a Markdown report (report.md) and exit")
	fmt.Println("                Use with -drift to include git information in the report")
	fmt.Println("  -v, -version  Show version information")
	fmt.Println("  -h, -help     Show this help message")
	fmt.Println()
	fmt.Println("TERRAFORM ARGUMENTS:")
	fmt.Println("  Any additional arguments are passed directly to terraform/tofu plan.")
	fmt.Println()
	fmt.Println("  Examples:")
	fmt.Println("    tplan -target=aws_instance.example")
	fmt.Println("    tplan -var-file=prod.tfvars")
	fmt.Println("    tplan -destroy")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  # Basic usage - just replace 'terraform plan' with 'tplan'")
	fmt.Println("  tplan")
	fmt.Println()
	fmt.Println("  # With drift detection")
	fmt.Println("  tplan -drift")
	fmt.Println()
	fmt.Println("  # Generate report")
	fmt.Println("  tplan -report")
	fmt.Println()
	fmt.Println("  # Generate report with drift information")
	fmt.Println("  tplan -report -drift")
	fmt.Println()
	fmt.Println("  # Target specific resource")
	fmt.Println("  tplan -target=aws_instance.web")
	fmt.Println()
	fmt.Println("  # Use variable file")
	fmt.Println("  tplan -var-file=production.tfvars")
	fmt.Println()
	fmt.Println("KEYBOARD CONTROLS:")
	fmt.Println("  ↑/↓, j/k      Navigate up/down")
	fmt.Println("  Enter, Space  Expand/collapse resource")
	fmt.Println("  e             Expand all")
	fmt.Println("  c             Collapse all")
	fmt.Println("  Tab           Switch between Changes/Errors/Warnings")
	fmt.Println("  g             Jump to top")
	fmt.Println("  G             Jump to bottom")
	fmt.Println("  q             Quit")
	fmt.Println()
	fmt.Println("REQUIREMENTS:")
	fmt.Println("  Either Terraform or OpenTofu must be installed and available in PATH.")
	fmt.Println()
}
