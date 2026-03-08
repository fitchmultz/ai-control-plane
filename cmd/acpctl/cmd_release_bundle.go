// cmd_release_bundle.go - Release bundle command implementation
//
// Purpose: Build and verify versioned deployment handoff bundles
//
// Responsibilities:
//   - Parse command arguments and dispatch to internal/release modules
//   - Display output and handle exit codes
//
// Non-scope:
//   - Actual bundle building (see internal/release/builder.go)
//   - Bundle verification logic (see internal/release/verifier.go)
//   - Argument parsing logic (see internal/release/parser.go)
//
// Scope:
//   - File-local implementation and interfaces only.
//
// Usage:
//   - Used through its package exports and CLI entrypoints as applicable.
//
// Invariants/Assumptions:
//   - Behavior must remain deterministic for equivalent inputs.

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/prereq"
	"github.com/mitchfultz/ai-control-plane/internal/release"
)

func runReleaseBundleCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	if len(args) == 0 {
		printReleaseBundleHelp(stdout)
		return exitcodes.ACPExitUsage
	}

	// Handle help for specific commands
	if len(args) > 1 && (args[1] == "--help" || args[1] == "-h") {
		switch args[0] {
		case "build":
			printReleaseBundleBuildHelp(stdout)
			return exitcodes.ACPExitSuccess
		case "verify":
			printReleaseBundleVerifyHelp(stdout)
			return exitcodes.ACPExitSuccess
		default:
			printReleaseBundleHelp(stdout)
			return exitcodes.ACPExitSuccess
		}
	}

	repoRoot := detectRepoRootWithContext(ctx)

	// Parse arguments using the parser module
	config, err := release.ParseArgs(args, repoRoot, release.GetDefaultVersion)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return exitcodes.ACPExitUsage
	}

	// Dispatch to appropriate command handler
	switch config.Command {
	case "build":
		return runReleaseBundleBuild(ctx, config, stdout, stderr)
	case "verify":
		return runReleaseBundleVerify(ctx, config, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "Unknown command: %s\n", config.Command)
		printReleaseBundleHelp(stderr)
		return exitcodes.ACPExitUsage
	}
}

func runReleaseBundleBuild(ctx context.Context, config *release.Config, stdout *os.File, stderr *os.File) int {
	out := output.New()
	repoRoot := detectRepoRootWithContext(ctx)

	// Validate version
	if err := release.ValidateVersion(config.Version); err != nil {
		fmt.Fprintln(stderr, err)
		return exitcodes.ACPExitUsage
	}

	// Create plan using the planner module
	plan, err := release.CreatePlan(config, repoRoot)
	if err != nil {
		fmt.Fprintf(stderr, out.Fail("Failed to create plan: %v\n"), err)
		return exitcodes.ACPExitRuntime
	}

	fmt.Fprint(stdout, out.Bold("Building release bundle")+"\n")
	fmt.Fprintf(stdout, "  Version: %s\n", config.Version)
	fmt.Fprintf(stdout, "  Output: %s\n", plan.BundlePath)

	// Check prerequisites
	if !prereq.CommandExists("tar") {
		fmt.Fprintln(stderr, out.Fail("tar not found"))
		return exitcodes.ACPExitPrereq
	}

	// Validate source files exist
	fmt.Fprint(stdout, out.Bold("Validating source files...")+"\n")
	_, err = release.ValidateSourceFiles(repoRoot, config.Verbose)
	if err != nil {
		fmt.Fprintf(stderr, out.Fail("%v\n"), err)
		return exitcodes.ACPExitDomain
	}

	// Build the bundle using the builder module
	fmt.Fprint(stdout, out.Bold("Assembling payload...")+"\n")
	builder := release.NewBuilder(repoRoot, config.Verbose)
	if err := builder.Build(plan, stdout); err != nil {
		fmt.Fprintf(stderr, out.Fail("%v\n"), err)
		return exitcodes.ACPExitRuntime
	}

	// Get bundle size
	info, err := os.Stat(plan.BundlePath)
	var sizeStr string
	if err == nil {
		sizeStr = release.HumanReadableSize(info.Size())
	}

	fmt.Fprintln(stdout, "")
	fmt.Fprint(stdout, out.Green(out.Bold("Bundle build complete"))+"\n")
	fmt.Fprintf(stdout, "  Bundle: %s\n", plan.BundlePath)
	fmt.Fprintf(stdout, "  Size: %s\n", sizeStr)
	fmt.Fprintf(stdout, "  Files: %d\n", len(release.CanonicalPaths))
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "To verify this bundle:")
	fmt.Fprintf(stdout, "  acpctl deploy release-bundle verify --bundle %s\n", plan.BundlePath)

	return exitcodes.ACPExitSuccess
}

func runReleaseBundleVerify(_ context.Context, config *release.Config, stdout *os.File, stderr *os.File) int {
	out := output.New()

	if config.Bundle == "" {
		fmt.Fprintln(stderr, "Missing required option: --bundle")
		return exitcodes.ACPExitUsage
	}

	// Resolve to absolute path
	bundlePath := config.Bundle
	if !filepath.IsAbs(bundlePath) {
		wd, _ := os.Getwd()
		bundlePath = filepath.Join(wd, bundlePath)
	}

	fmt.Fprint(stdout, out.Bold("Verifying release bundle")+"\n")
	fmt.Fprintf(stdout, "  Bundle: %s\n", bundlePath)

	// Verify using the verifier module
	verifier := release.NewVerifier(config.Verbose)
	result, err := verifier.Verify(bundlePath, stdout)
	if err != nil {
		fmt.Fprintf(stderr, out.Fail("%v\n"), err)
		if os.IsNotExist(err) {
			return exitcodes.ACPExitDomain
		}
		return exitcodes.ACPExitRuntime
	}

	// Check verification results
	if !result.SidecarValid {
		fmt.Fprintln(stderr, out.Fail("Bundle tarball checksum mismatch - possible tampering"))
		return exitcodes.ACPExitDomain
	}

	if !result.StructureValid {
		fmt.Fprintln(stderr, out.Fail("Bundle structure validation failed"))
		return exitcodes.ACPExitDomain
	}

	if !result.PayloadValid {
		fmt.Fprintln(stderr, out.Fail("Payload checksum verification failed"))
		return exitcodes.ACPExitDomain
	}

	fmt.Fprintf(stdout, "  %s Tarball checksum verified (sidecar)\n", out.Pass(""))
	fmt.Fprintf(stdout, "  %s Required bundle structure verified\n", out.Pass(""))
	fmt.Fprintf(stdout, "  %s Payload checksum verification passed\n", out.Pass(""))

	fmt.Fprintln(stdout, "")
	fmt.Fprint(stdout, out.Green(out.Bold("Bundle verification complete"))+"\n")
	fmt.Fprintf(stdout, "  Files in manifest: %d\n", len(release.CanonicalPaths))
	fmt.Fprintln(stdout, "  Tarball validated: yes")
	fmt.Fprintln(stdout, "  Payload integrity: verified")

	return exitcodes.ACPExitSuccess
}

func printReleaseBundleHelp(out *os.File) {
	fmt.Fprint(out, `Usage: acpctl deploy release-bundle <command> [OPTIONS]

Commands:
  build    Build a versioned deployment bundle
  verify   Verify bundle integrity using checksums

Examples:
  # Build a bundle with git sha version
  acpctl deploy release-bundle build

  # Build with explicit version
  acpctl deploy release-bundle build --version v2026.02.11

  # Verify a bundle
  acpctl deploy release-bundle verify --bundle path/to/bundle.tar.gz

Exit codes:
  0   - Success
  1   - Domain failure (verification failed, missing files)
  2   - Prerequisites not ready (missing tools)
  3   - Runtime/internal error
  64  - Usage error (invalid arguments)
`)
}

func printReleaseBundleBuildHelp(out *os.File) {
	fmt.Fprint(out, `Usage: acpctl deploy release-bundle build [OPTIONS]

Build a versioned deployment bundle with checksums and install manifest.

Options:
  --version VERSION    Version tag for the bundle (default: git short sha, or 'dev')
  --output-dir DIR     Output directory for bundle (default: demo/logs/release-bundles)
  --verbose            Enable verbose output
  --help               Show this help message

Exit codes:
  0   - Success
  1   - Domain failure (missing required files)
  2   - Prerequisites not ready
  3   - Runtime/internal error
  64  - Usage error
`)
}

func printReleaseBundleVerifyHelp(out *os.File) {
	fmt.Fprint(out, `Usage: acpctl deploy release-bundle verify [OPTIONS]

Verify bundle integrity using sha256 checksums.

Options:
  --bundle PATH        Path to the tarball to verify (requires PATH.sha256 sidecar)
  --verbose            Enable verbose output
  --help               Show this help message

Exit codes:
  0   - Verification succeeded
  1   - Verification failed (tampering detected or missing files)
  2   - Prerequisites not ready
  3   - Runtime/internal error
  64  - Usage error
`)
}
