// cmd_chargeback_adapters.go - Chargeback CLI adapters.
//
// Purpose:
//   - Bridge parsed acpctl chargeback arguments into typed internal workflows.
//
// Responsibilities:
//   - Convert parsed command input into typed chargeback command inputs.
//   - Compose database-backed report workflows and env-backed render/payload
//     requests.
//   - Map workflow results onto stdout, stderr, and ACP exit codes.
//
// Non-scope:
//   - Does not own chargeback business defaults or env parsing.
//   - Does not implement report rendering, archival, or notification delivery.
//
// Invariants/Assumptions:
//   - Internal chargeback adapters are the single owner of defaults and
//     validation.
//   - CLI error mapping remains deterministic.
//
// Scope:
//   - Chargeback CLI adapter logic only.
//
// Usage:
//   - Used by cmd_chargeback.go backend hooks.
package main

import (
	"context"
	"fmt"

	"github.com/mitchfultz/ai-control-plane/internal/chargeback"
	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/db"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
)

type chargebackReportAdapterOptions struct {
	Command chargeback.ReportCommandInput
	Verbose bool
}

type chargebackRenderAdapterOptions struct {
	Command chargeback.RenderCommandInput
}

type chargebackPayloadAdapterOptions struct {
	Command chargeback.PayloadCommandInput
}

func bindChargebackReportOptions(_ commandBindContext, input parsedCommandInput) (any, error) {
	command := chargeback.ReportCommandInput{
		ReportMonth:          input.String("month"),
		Format:               input.String("format"),
		ArchiveDir:           input.String("archive-dir"),
		VarianceThreshold:    input.String("variance-threshold"),
		AnomalyThreshold:     input.String("anomaly-threshold"),
		BudgetAlertThreshold: input.String("budget-alert-threshold"),
		Notify:               input.Bool("notify"),
	}
	if input.Bool("forecast") {
		value := true
		command.ForecastEnabled = &value
	}
	if input.Bool("no-forecast") {
		value := false
		command.ForecastEnabled = &value
	}
	return chargebackReportAdapterOptions{
		Command: command,
		Verbose: input.Bool("verbose"),
	}, nil
}

func bindChargebackRenderOptions(_ commandBindContext, input parsedCommandInput) (any, error) {
	return chargebackRenderAdapterOptions{
		Command: chargeback.RenderCommandInput{
			Format: input.String("format"),
		},
	}, nil
}

func bindChargebackPayloadOptions(_ commandBindContext, input parsedCommandInput) (any, error) {
	return chargebackPayloadAdapterOptions{
		Command: chargeback.PayloadCommandInput{
			Target: input.String("target"),
		},
	}, nil
}

func runChargebackReportCommand(ctx context.Context, runCtx commandRunContext, raw any) int {
	options := raw.(chargebackReportAdapterOptions)
	loader := config.NewLoader()
	workflowInput, err := chargeback.NewReportWorkflowInput(options.Command, loader, runCtx.RepoRoot, nil)
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, "Error: %v\n", err)
		return exitcodes.ACPExitUsage
	}

	connector := db.NewConnector(runCtx.RepoRoot)
	defer func() { _ = connector.Close() }()
	if err := connector.ConfigError(); err != nil {
		fmt.Fprintf(runCtx.Stderr, "Error: %v\n", err)
		return exitcodes.ACPExitUsage
	}
	reader, err := db.NewChargebackReader(connector)
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, "Error: %v\n", err)
		return exitcodes.ACPExitUsage
	}
	if options.Verbose {
		fmt.Fprintln(runCtx.Stderr, "INFO: Generating typed chargeback report")
	}
	result, err := chargeback.NewReportWorkflow(chargeback.NewDBStore(reader)).Run(ctx, workflowInput)
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, "Error: %v\n", err)
		return exitcodes.ACPExitRuntime
	}
	if err := chargeback.WriteSelectedOutput(runCtx.Stdout, workflowInput.Request.Format, result.Outputs); err != nil {
		fmt.Fprintf(runCtx.Stderr, "Error: %v\n", err)
		return exitcodes.ACPExitRuntime
	}
	if options.Verbose {
		fmt.Fprintf(runCtx.Stderr, "INFO: Archived artifacts under %s\n", result.Outputs.Archived["md"])
	}
	if result.Data.VarianceExceeded || result.Data.HasAnomalies {
		return exitcodes.ACPExitDomain
	}
	return exitcodes.ACPExitSuccess
}

func runChargebackRenderCommand(_ context.Context, runCtx commandRunContext, raw any) int {
	options := raw.(chargebackRenderAdapterOptions)
	request, err := chargeback.NewRenderRequest(options.Command, config.NewLoader(), nil)
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, "Error: %v\n", err)
		return exitcodes.ACPExitUsage
	}
	payload, err := chargeback.RenderToBytes(request)
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, "Error: %v\n", err)
		return exitcodes.ACPExitRuntime
	}
	if _, err := runCtx.Stdout.Write(payload); err != nil {
		fmt.Fprintf(runCtx.Stderr, "Error: write rendered chargeback output: %v\n", err)
		return exitcodes.ACPExitRuntime
	}
	return exitcodes.ACPExitSuccess
}

func runChargebackPayloadCommand(_ context.Context, runCtx commandRunContext, raw any) int {
	options := raw.(chargebackPayloadAdapterOptions)
	request, err := chargeback.NewPayloadRequest(options.Command, config.NewLoader(), nil)
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, "Error: %v\n", err)
		return exitcodes.ACPExitUsage
	}
	payload, err := chargeback.BuildPayload(request)
	if err != nil {
		fmt.Fprintf(runCtx.Stderr, "Error: %v\n", err)
		return exitcodes.ACPExitRuntime
	}
	if _, err := runCtx.Stdout.Write(payload); err != nil {
		fmt.Fprintf(runCtx.Stderr, "Error: write chargeback payload: %v\n", err)
		return exitcodes.ACPExitRuntime
	}
	return exitcodes.ACPExitSuccess
}
