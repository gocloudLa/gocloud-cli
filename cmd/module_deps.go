package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"gocloud-cli/internal/moddeps"

	"github.com/spf13/cobra"
)

var (
	depsRoot     string
	depsJSON     bool
	depsBumpPlan bool
)

var depsCmd = &cobra.Command{
	Use:   "deps",
	Short: "Inspect Terraform registry pins (.tf modules + required_providers)",
	Long: `Commands to scan Terraform files under a repo root, compare public Registry pins to
latest versions, and optionally rewrite a module pin (same behavior as the legacy Python checker).`,
}

var moduleDepsCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check Terraform module/provider pins against registry.terraform.io",
	Long: `Walks *.tf under the scan root (skipping .terraform/), compares registry module versions and
required_providers constraints to the public Terraform Registry. Private modules show as not found.

Plain output exits with status 1 when any public registry module is outdated (for CI).`,
	RunE: runModuleDepsCheck,
}

var moduleDepsUpdateCmd = &cobra.Command{
	Use:   "update <module_source> <current_version> <new_version> <file>",
	Short: "Rewrite one module version pin in a single .tf file",
	Long: `Replaces version for blocks where source and version use double quotes exactly like:

  source = "<module_source>"
  version = "<current_version>"

<file> is relative to --dir / scan root when not absolute.`,
	Args: cobra.ExactArgs(4),
	RunE: runModuleDepsUpdate,
}

func init() {
	moduleCmd.AddCommand(depsCmd)
	depsCmd.AddCommand(moduleDepsCheckCmd)
	depsCmd.AddCommand(moduleDepsUpdateCmd)

	depsCmd.PersistentFlags().StringVar(&depsRoot, "dir", "", `Root directory for *.tf scan (default: current working directory)`)

	moduleDepsCheckCmd.Flags().BoolVar(&depsJSON, "json", false, "Print JSON for the same module lines as plain output (no providers)")
	moduleDepsCheckCmd.Flags().BoolVar(&depsBumpPlan, "bump-plan", false, "Print bump plan JSON (legacy --bump-plan-json)")
}

func runModuleDepsCheck(_ *cobra.Command, _ []string) error {
	if depsJSON && depsBumpPlan {
		return errors.New("use only one of --json or --bump-plan")
	}
	root, err := moddeps.ResolveRoot(depsRoot)
	if err != nil {
		return err
	}
	ctx := context.Background()
	client := &moddeps.Client{}

	if depsBumpPlan {
		plan, err := client.BuildBumpPlan(ctx, root)
		if err != nil {
			return err
		}
		return moddeps.EncodeJSON(os.Stdout, plan)
	}
	if depsJSON {
		rep, err := client.BuildModulesCheckJSON(ctx, root)
		if err != nil {
			return err
		}
		return moddeps.EncodeJSON(os.Stdout, rep)
	}

	n, err := client.WriteHumanReport(ctx, os.Stdout, root)
	if err != nil {
		return err
	}
	if n > 0 {
		os.Exit(1)
	}
	return nil
}

func runModuleDepsUpdate(_ *cobra.Command, args []string) error {
	root, err := moddeps.ResolveRoot(depsRoot)
	if err != nil {
		return err
	}
	source := args[0]
	cur := args[1]
	newVer := args[2]
	fileArg := args[3]

	n, err := moddeps.ApplyModuleVersionBump(root, source, cur, newVer, fileArg)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Updated %d version pin(s).\n", n)
	if n == 0 {
		os.Exit(1)
	}
	return nil
}
