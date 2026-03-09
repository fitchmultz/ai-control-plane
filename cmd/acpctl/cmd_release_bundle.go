// cmd_release_bundle.go - Release bundle command implementation
//
// Purpose: Build and verify versioned deployment handoff bundles.
//
// Responsibilities:
//   - Define the typed release-bundle command tree.
//   - Dispatch to internal/bundle modules for build and verify flows.
//   - Display operator-facing output with stable exit codes.
//
// Non-scope:
//   - Actual bundle building logic.
//   - Bundle verification internals.
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
	"log/slog"
	"os"

	"github.com/mitchfultz/ai-control-plane/internal/bundle"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/logging"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	repopath "github.com/mitchfultz/ai-control-plane/internal/paths"
	"github.com/mitchfultz/ai-control-plane/internal/prereq"
)

func releaseBundleCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "release-bundle",
		Summary:     "Build deployment release bundle",
		Description: "Build and verify versioned deployment handoff bundles.",
		Children: []*commandSpec{
			newNativeCommandSpec(nativeCommandConfig{
				Name:        "build",
				Summary:     "Build a versioned deployment bundle",
				Description: "Build a versioned deployment bundle with checksums and install manifest.",
				Options: []commandOptionSpec{
					{Name: "version", ValueName: "VERSION", Summary: "Version tag for the bundle", Type: optionValueString, DefaultText: "git short sha"},
					{Name: "output-dir", ValueName: "DIR", Summary: "Output directory for the bundle", Type: optionValueString, DefaultText: "demo/logs/release-bundles"},
					{Name: "verbose", Summary: "Enable verbose output", Type: optionValueBool},
				},
				Bind: bindReleaseBundleBuildOptions,
				Run:  runReleaseBundleBuildTyped,
			}),
			newNativeCommandSpec(nativeCommandConfig{
				Name:        "verify",
				Summary:     "Verify bundle integrity using checksums",
				Description: "Verify bundle integrity using sha256 checksums.",
				Options: []commandOptionSpec{
					{Name: "bundle", ValueName: "PATH", Summary: "Path to the tarball to verify", Type: optionValueString, Required: true},
					{Name: "verbose", Summary: "Enable verbose output", Type: optionValueBool},
				},
				Bind: bindReleaseBundleVerifyOptions,
				Run:  runReleaseBundleVerifyTyped,
			}),
		},
	}
}

func bindReleaseBundleBuildOptions(bindCtx commandBindContext, input parsedCommandInput) (any, error) {
	repoRoot, err := requireCommandRepoRoot(bindCtx)
	if err != nil {
		return nil, err
	}
	version := input.String("version")
	if version == "" {
		version = bundle.GetDefaultVersion(repoRoot)
	}
	outputDir := input.String("output-dir")
	if outputDir == "" {
		outputDir = repopath.ReleaseBundlesPath(repoRoot)
	} else {
		outputDir = resolveRepoInput(repoRoot, outputDir)
	}
	return &bundle.Config{
		Command:   "build",
		Version:   version,
		OutputDir: outputDir,
		Verbose:   input.Bool("verbose"),
	}, nil
}

func bindReleaseBundleVerifyOptions(bindCtx commandBindContext, input parsedCommandInput) (any, error) {
	repoRoot, err := requireCommandRepoRoot(bindCtx)
	if err != nil {
		return nil, err
	}
	return &bundle.Config{
		Command: "verify",
		Bundle:  resolveRepoInput(repoRoot, input.String("bundle")),
		Verbose: input.Bool("verbose"),
	}, nil
}

func runReleaseBundleBuildTyped(ctx context.Context, runCtx commandRunContext, raw any) int {
	config := raw.(*bundle.Config)
	out := output.New()
	ctx = logging.WithLogger(ctx, ensureWorkflowLogger(runCtx).With(slog.String("workflow", "release_bundle_build")))

	if err := bundle.ValidateVersion(config.Version); err != nil {
		fmt.Fprintln(runCtx.Stderr, err)
		return exitcodes.ACPExitUsage
	}

	plan, err := bundle.CreatePlan(config, runCtx.RepoRoot)
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, out.Fail("Failed to create plan: %v\n"), err)
		return exitcodes.ACPExitRuntime
	}

	fmt.Fprint(runCtx.Stdout, out.Bold("Building release bundle")+"\n")
	fmt.Fprintf(runCtx.Stdout, "  Version: %s\n", config.Version)
	fmt.Fprintf(runCtx.Stdout, "  Output: %s\n", plan.BundlePath)

	if !prereq.CommandExists("tar") {
		fmt.Fprintln(runCtx.Stderr, out.Fail("tar not found"))
		return exitcodes.ACPExitPrereq
	}

	fmt.Fprint(runCtx.Stdout, out.Bold("Validating source files...")+"\n")
	_, err = bundle.ValidateSourceFiles(runCtx.RepoRoot, config.Verbose)
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, out.Fail("%v\n"), err)
		return exitcodes.ACPExitDomain
	}

	fmt.Fprint(runCtx.Stdout, out.Bold("Assembling payload...")+"\n")
	builderInstance := bundle.NewBuilder(runCtx.RepoRoot, config.Verbose)
	if err := builderInstance.Build(ctx, plan); err != nil {
		fmt.Fprintf(runCtx.Stderr, out.Fail("%v\n"), err)
		return exitcodes.ACPExitRuntime
	}

	info, err := os.Stat(plan.BundlePath)
	var sizeStr string
	if err == nil {
		sizeStr = bundle.HumanReadableSize(info.Size())
	}

	fmt.Fprintln(runCtx.Stdout, "")
	fmt.Fprint(runCtx.Stdout, out.Green(out.Bold("Bundle build complete"))+"\n")
	fmt.Fprintf(runCtx.Stdout, "  Bundle: %s\n", plan.BundlePath)
	fmt.Fprintf(runCtx.Stdout, "  Size: %s\n", sizeStr)
	fmt.Fprintf(runCtx.Stdout, "  Files: %d\n", len(bundle.CanonicalPaths))
	fmt.Fprintln(runCtx.Stdout, "")
	fmt.Fprintln(runCtx.Stdout, "To verify this bundle:")
	fmt.Fprintf(runCtx.Stdout, "  acpctl deploy release-bundle verify --bundle %s\n", plan.BundlePath)

	return exitcodes.ACPExitSuccess
}

func runReleaseBundleVerifyTyped(ctx context.Context, runCtx commandRunContext, raw any) int {
	config := raw.(*bundle.Config)
	out := output.New()
	ctx = logging.WithLogger(ctx, ensureWorkflowLogger(runCtx).With(slog.String("workflow", "release_bundle_verify")))

	bundlePath := config.Bundle
	if bundlePath == "" {
		return failCommand(runCtx.Stderr, out, exitcodes.ACPExitUsage, nil, "bundle path is required")
	}

	fmt.Fprint(runCtx.Stdout, out.Bold("Verifying release bundle")+"\n")
	fmt.Fprintf(runCtx.Stdout, "  Bundle: %s\n", bundlePath)

	verifier := bundle.NewVerifier(config.Verbose)
	result, err := verifier.Verify(ctx, bundlePath)
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, out.Fail("%v\n"), err)
		if os.IsNotExist(err) {
			return exitcodes.ACPExitDomain
		}
		return exitcodes.ACPExitRuntime
	}

	if !result.SidecarValid {
		fmt.Fprintln(runCtx.Stderr, out.Fail("Bundle tarball checksum mismatch - possible tampering"))
		return exitcodes.ACPExitDomain
	}
	if !result.StructureValid {
		fmt.Fprintln(runCtx.Stderr, out.Fail("Bundle structure validation failed"))
		return exitcodes.ACPExitDomain
	}
	if !result.PayloadValid {
		fmt.Fprintln(runCtx.Stderr, out.Fail("Payload checksum verification failed"))
		return exitcodes.ACPExitDomain
	}

	fmt.Fprintf(runCtx.Stdout, "  %s Tarball checksum verified (sidecar)\n", out.Pass(""))
	fmt.Fprintf(runCtx.Stdout, "  %s Required bundle structure verified\n", out.Pass(""))
	fmt.Fprintf(runCtx.Stdout, "  %s Payload checksum verification passed\n", out.Pass(""))
	fmt.Fprintln(runCtx.Stdout, "")
	fmt.Fprint(runCtx.Stdout, out.Green(out.Bold("Bundle verification complete"))+"\n")
	fmt.Fprintf(runCtx.Stdout, "  Files in manifest: %d\n", len(bundle.CanonicalPaths))
	fmt.Fprintln(runCtx.Stdout, "  Tarball validated: yes")
	fmt.Fprintln(runCtx.Stdout, "  Payload integrity: verified")

	return exitcodes.ACPExitSuccess
}

func runReleaseBundleCommand(ctx context.Context, args []string, stdout *os.File, stderr *os.File) int {
	return runCommandPath(ctx, []string{"deploy", "release-bundle"}, args, stdout, stderr)
}
