// cmd_key_lifecycle.go - Typed key lifecycle command handlers.
//
// Purpose:
//   - Implement list, inspect, and rotate flows for virtual keys.
//
// Responsibilities:
//   - Bind parsed CLI input into typed lifecycle requests.
//   - Compose gateway and database services for key inspection workflows.
//   - Render human and JSON outputs with stable ACP exit codes.
//
// Scope:
//   - `acpctl key list|inspect|rotate` only.
//
// Usage:
//   - Invoked through the typed `key` command tree.
//
// Invariants/Assumptions:
//   - Lifecycle commands remain make-independent and deterministic.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	acpconfig "github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/db"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/gateway"
	"github.com/mitchfultz/ai-control-plane/internal/keygen"
	"github.com/mitchfultz/ai-control-plane/internal/output"
)

type keyListOptions struct {
	JSON bool
}

type keyInspectOptions struct {
	Alias string
	Month string
	JSON  bool
}

type keyRotateOptions struct {
	Alias            string
	ReplacementAlias string
	Budget           float64
	RPM              int
	TPM              int
	Parallel         int
	Duration         string
	Role             string
	Month            string
	DryRun           bool
	RevokeOld        bool
	JSON             bool
}

func bindKeyInspectOptions(input parsedCommandInput) (keyInspectOptions, error) {
	alias := input.Argument(0)
	if alias == "" {
		return keyInspectOptions{}, fmt.Errorf("alias is required")
	}
	return keyInspectOptions{
		Alias: alias,
		Month: input.String("month"),
		JSON:  input.Bool("json"),
	}, nil
}

func bindKeyRotateOptions(input parsedCommandInput) (keyRotateOptions, error) {
	alias := input.Argument(0)
	if alias == "" {
		return keyRotateOptions{}, fmt.Errorf("alias is required")
	}
	budget, err := input.FloatDefault("budget", 0)
	if err != nil {
		return keyRotateOptions{}, fmt.Errorf("invalid budget: %s", input.String("budget"))
	}
	rpm, err := input.IntDefault("rpm", 0)
	if err != nil {
		return keyRotateOptions{}, fmt.Errorf("invalid RPM: %s", input.String("rpm"))
	}
	tpm, err := input.IntDefault("tpm", 0)
	if err != nil {
		return keyRotateOptions{}, fmt.Errorf("invalid TPM: %s", input.String("tpm"))
	}
	parallel, err := input.IntDefault("parallel", 0)
	if err != nil {
		return keyRotateOptions{}, fmt.Errorf("invalid parallel: %s", input.String("parallel"))
	}
	return keyRotateOptions{
		Alias:            alias,
		ReplacementAlias: input.String("replacement-alias"),
		Budget:           budget,
		RPM:              rpm,
		TPM:              tpm,
		Parallel:         parallel,
		Duration:         input.String("duration"),
		Role:             input.String("role"),
		Month:            input.String("month"),
		DryRun:           input.Bool("dry-run"),
		RevokeOld:        input.Bool("revoke-old"),
		JSON:             input.Bool("json"),
	}, nil
}

func runKeyList(ctx context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(keyListOptions)
	out := output.New()
	client, code := openKeyLifecycleClient(runCtx, out)
	if code != exitcodes.ACPExitSuccess {
		return code
	}

	keys, err := client.ListKeys(ctx)
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, out.Fail("Key listing failed: %v\n"), err)
		return exitcodes.ACPExitRuntime
	}

	if opts.JSON {
		return writeJSONOutput(runCtx, keys)
	}
	printKeyList(runCtx.Stdout, keys)
	return exitcodes.ACPExitSuccess
}

func runKeyInspect(ctx context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(keyInspectOptions)
	out := output.New()
	client, code := openKeyLifecycleClient(runCtx, out)
	if code != exitcodes.ACPExitSuccess {
		return code
	}
	readonly, closeFn, code := openKeyLifecycleReadonly(runCtx, out)
	if code != exitcodes.ACPExitSuccess {
		return code
	}
	defer closeFn()

	inspection, err := keygen.InspectKey(ctx, client, readonly, opts.Alias, opts.Month, time.Now())
	if err != nil {
		return writeKeyLifecycleError(runCtx, out, "Key inspection failed", err)
	}

	if opts.JSON {
		return writeJSONOutput(runCtx, inspection)
	}
	printKeyInspection(runCtx.Stdout, inspection)
	return exitcodes.ACPExitSuccess
}

func runKeyRotate(ctx context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(keyRotateOptions)
	out := output.New()
	client, code := openKeyLifecycleClient(runCtx, out)
	if code != exitcodes.ACPExitSuccess {
		return code
	}
	readonly, closeFn, code := openKeyLifecycleReadonly(runCtx, out)
	if code != exitcodes.ACPExitSuccess {
		return code
	}
	defer closeFn()

	result, err := keygen.RotateKey(ctx, client, readonly, keygen.RotationRequest{
		SourceAlias:      opts.Alias,
		ReplacementAlias: opts.ReplacementAlias,
		Budget:           opts.Budget,
		RPM:              opts.RPM,
		TPM:              opts.TPM,
		Parallel:         opts.Parallel,
		Duration:         opts.Duration,
		Role:             opts.Role,
		ReportMonth:      opts.Month,
		DryRun:           opts.DryRun,
		RevokeOld:        opts.RevokeOld,
	}, time.Now())
	if err != nil {
		return writeKeyLifecycleError(runCtx, out, "Key rotation failed", err)
	}

	if opts.JSON {
		return writeJSONOutput(runCtx, result)
	}
	printKeyRotation(runCtx.Stdout, result)
	return exitcodes.ACPExitSuccess
}

func openKeyLifecycleClient(runCtx commandRunContext, out *output.Output) (*gateway.Client, int) {
	if err := keygen.CheckPrerequisites(true); err != nil {
		fmt.Fprintf(runCtx.Stderr, out.Fail("%v\n"), err)
		fmt.Fprintln(runCtx.Stderr, "Set it in your environment or demo/.env: LITELLM_MASTER_KEY=...")
		return nil, exitcodes.ACPExitPrereq
	}
	gatewaySettings := acpconfig.NewLoader().WithRepoRoot(runCtx.RepoRoot).Gateway(true)
	return gateway.NewClient(
		gateway.WithBaseURL(gatewaySettings.BaseURL),
		gateway.WithMasterKey(gatewaySettings.MasterKey),
	), exitcodes.ACPExitSuccess
}

func openKeyLifecycleReadonly(runCtx commandRunContext, out *output.Output) (*db.ReadonlyService, func(), int) {
	connector := db.NewConnector(runCtx.RepoRoot)
	if err := connector.ConfigError(); err != nil {
		_ = connector.Close()
		fmt.Fprintf(runCtx.Stderr, out.Fail("Database configuration error: %v\n"), err)
		return nil, func() {}, exitcodes.ACPExitUsage
	}
	return db.NewReadonlyService(connector), func() { _ = connector.Close() }, exitcodes.ACPExitSuccess
}

func writeJSONOutput(runCtx commandRunContext, payload any) int {
	encoder := json.NewEncoder(runCtx.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(payload); err != nil {
		fmt.Fprintf(runCtx.Stderr, "Error: encode JSON output: %v\n", err)
		return exitcodes.ACPExitRuntime
	}
	return exitcodes.ACPExitSuccess
}

func writeKeyLifecycleError(runCtx commandRunContext, out *output.Output, prefix string, err error) int {
	fmt.Fprintf(runCtx.Stderr, out.Fail("%s: %v\n"), prefix, err)
	if isKeyLifecycleUsageError(err) {
		return exitcodes.ACPExitUsage
	}
	return exitcodes.ACPExitRuntime
}

func isKeyLifecycleUsageError(err error) bool {
	var validationErr *keygen.ValidationError
	if errors.As(err, &validationErr) {
		return true
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(message, "invalid report month") || strings.Contains(message, "not found")
}
