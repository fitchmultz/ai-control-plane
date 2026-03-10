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

func bindChargebackReportOptions(input parsedCommandInput) chargebackReportAdapterOptions {
	command := chargeback.ReportCommandInput{
		ReportMonth:          input.NormalizedString("month"),
		Format:               input.NormalizedString("format"),
		ArchiveDir:           input.NormalizedString("archive-dir"),
		VarianceThreshold:    input.NormalizedString("variance-threshold"),
		AnomalyThreshold:     input.NormalizedString("anomaly-threshold"),
		BudgetAlertThreshold: input.NormalizedString("budget-alert-threshold"),
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
	}
}

func bindChargebackRenderOptions(input parsedCommandInput) chargebackRenderAdapterOptions {
	return chargebackRenderAdapterOptions{
		Command: chargeback.RenderCommandInput{
			Format: input.NormalizedString("format"),
		},
	}
}

func bindChargebackPayloadOptions(input parsedCommandInput) chargebackPayloadAdapterOptions {
	return chargebackPayloadAdapterOptions{
		Command: chargeback.PayloadCommandInput{
			Target: input.NormalizedString("target"),
		},
	}
}

func runChargebackReportCommand(ctx context.Context, runCtx commandRunContext, raw any) int {
	options := raw.(chargebackReportAdapterOptions)
	logger := workflowLogger(runCtx, "chargeback_report", "format", options.Command.Format, "notify", options.Command.Notify)
	workflowStart(logger)
	loader := config.NewLoader()
	workflowInput, err := chargeback.NewReportWorkflowInput(options.Command, loader, runCtx.RepoRoot, nil)
	if err != nil {
		workflowFailure(logger, err)
		fmt.Fprintf(runCtx.Stderr, "Error: %v\n", err)
		return exitcodes.ACPExitUsage
	}

	connector := db.NewConnector(runCtx.RepoRoot)
	defer func() { _ = connector.Close() }()
	if err := connector.ConfigError(); err != nil {
		workflowFailure(logger, err)
		fmt.Fprintf(runCtx.Stderr, "Error: %v\n", err)
		return exitcodes.ACPExitUsage
	}
	reader, err := db.NewChargebackReader(connector)
	if err != nil {
		workflowFailure(logger, err)
		fmt.Fprintf(runCtx.Stderr, "Error: %v\n", err)
		return exitcodes.ACPExitUsage
	}
	if options.Verbose {
		logger.Info("workflow.detail", "message", "generating typed chargeback report")
	}
	result, err := chargeback.NewReportWorkflow(chargeback.NewDBStore(reader)).Run(ctx, workflowInput)
	if err != nil {
		workflowFailure(logger, err)
		fmt.Fprintf(runCtx.Stderr, "Error: %v\n", err)
		return exitcodes.ACPExitRuntime
	}
	if err := chargeback.WriteSelectedOutput(runCtx.Stdout, workflowInput.Request.Format, result.Outputs); err != nil {
		workflowFailure(logger, err)
		fmt.Fprintf(runCtx.Stderr, "Error: %v\n", err)
		return exitcodes.ACPExitRuntime
	}
	if options.Verbose {
		logger.Info("workflow.detail", "archived_markdown", result.Outputs.Archived["md"])
	}
	workflowComplete(logger, "variance_exceeded", result.Data.VarianceExceeded, "has_anomalies", result.Data.HasAnomalies)
	if result.Data.VarianceExceeded || result.Data.HasAnomalies {
		return exitcodes.ACPExitDomain
	}
	return exitcodes.ACPExitSuccess
}

func runChargebackRenderCommand(_ context.Context, runCtx commandRunContext, raw any) int {
	options := raw.(chargebackRenderAdapterOptions)
	logger := workflowLogger(runCtx, "chargeback_render", "format", options.Command.Format)
	workflowStart(logger)
	request, err := chargeback.NewRenderRequest(options.Command, config.NewLoader(), nil)
	if err != nil {
		workflowFailure(logger, err)
		fmt.Fprintf(runCtx.Stderr, "Error: %v\n", err)
		return exitcodes.ACPExitUsage
	}
	payload, err := chargeback.RenderToBytes(request)
	if err != nil {
		workflowFailure(logger, err)
		fmt.Fprintf(runCtx.Stderr, "Error: %v\n", err)
		return exitcodes.ACPExitRuntime
	}
	if _, err := runCtx.Stdout.Write(payload); err != nil {
		workflowFailure(logger, err)
		fmt.Fprintf(runCtx.Stderr, "Error: write rendered chargeback output: %v\n", err)
		return exitcodes.ACPExitRuntime
	}
	workflowComplete(logger, "bytes", len(payload))
	return exitcodes.ACPExitSuccess
}

func runChargebackPayloadCommand(_ context.Context, runCtx commandRunContext, raw any) int {
	options := raw.(chargebackPayloadAdapterOptions)
	logger := workflowLogger(runCtx, "chargeback_payload", "target", options.Command.Target)
	workflowStart(logger)
	request, err := chargeback.NewPayloadRequest(options.Command, config.NewLoader(), nil)
	if err != nil {
		workflowFailure(logger, err)
		fmt.Fprintf(runCtx.Stderr, "Error: %v\n", err)
		return exitcodes.ACPExitUsage
	}
	payload, err := chargeback.BuildPayload(request)
	if err != nil {
		workflowFailure(logger, err)
		fmt.Fprintf(runCtx.Stderr, "Error: %v\n", err)
		return exitcodes.ACPExitRuntime
	}
	if _, err := runCtx.Stdout.Write(payload); err != nil {
		workflowFailure(logger, err)
		fmt.Fprintf(runCtx.Stderr, "Error: write chargeback payload: %v\n", err)
		return exitcodes.ACPExitRuntime
	}
	workflowComplete(logger, "bytes", len(payload))
	return exitcodes.ACPExitSuccess
}
