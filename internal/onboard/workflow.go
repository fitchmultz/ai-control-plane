// Package onboard implements tool onboarding workflows.
//
// Purpose:
//
//	Coordinate typed onboarding layers for parsing defaults, prerequisite
//	loading, key generation, export rendering, verification, and config writes.
//
// Responsibilities:
//   - Keep the top-level workflow thin and ordered.
//   - Map workflow failures onto canonical ACP exit codes.
//   - Delegate detailed logic to focused package files.
//
// Scope:
//   - Onboarding workflow coordination only.
//
// Usage:
//   - Called by `acpctl onboard`.
//
// Invariants/Assumptions:
//   - The coordinator remains thin while package sublayers own behavior.
package onboard

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/gateway"
	"github.com/mitchfultz/ai-control-plane/internal/keygen"
	"github.com/mitchfultz/ai-control-plane/internal/logging"
)

type GatewayKeyGenerator struct {
	MasterKey string
}

func Run(ctx context.Context, opts Options) Result {
	opts = withDefaults(opts)
	logger := logging.FromContext(ctx).With(slog.String("component", "onboard"))
	if opts.Tool == "" || isHelpToken(opts.Tool) {
		var help strings.Builder
		printMainHelp(&help)
		return Result{ExitCode: exitcodes.ACPExitSuccess, Stdout: help.String()}
	}
	logger = logger.With(slog.String("tool", opts.Tool))
	logger.Info("workflow.start", slog.String("mode", opts.Mode))
	if opts.Mode == "help" {
		var help strings.Builder
		printMainHelp(&help)
		printToolHelp(&help, opts.Tool)
		return Result{ExitCode: exitcodes.ACPExitSuccess, Stdout: help.String()}
	}

	resolved, err := resolveDefaults(opts)
	if err != nil {
		logger.Error("workflow.resolve_defaults_failed", logging.Err(err))
		return Result{ExitCode: exitcodes.ACPExitUsage, Stderr: fmt.Sprintf("ERROR: %v\n", err)}
	}
	prereqs, err := loadPrerequisites(resolved)
	if err != nil {
		logger.Error("workflow.prereq_failed", logging.Err(err))
		return Result{ExitCode: exitcodes.ACPExitPrereq, Stderr: fmt.Sprintf("ERROR: %v\n", err)}
	}

	state, err := prepareRunState(ctx, resolved, prereqs)
	if err != nil {
		logger.Error("workflow.prepare_state_failed", logging.Err(err))
		return Result{ExitCode: exitcodes.ACPExitDomain, Stderr: fmt.Sprintf("ERROR: %v\n", err)}
	}
	if err := verifySubscriptionPrereq(ctx, state); err != nil {
		logger.Warn("workflow.subscription_prereq_failed", logging.Err(err))
		return Result{ExitCode: exitcodes.ACPExitDomain, Stderr: fmt.Sprintf("WARN: %v\n", err)}
	}

	if err := verifyOnboarding(ctx, state); err != nil {
		logger.Error("workflow.verify_failed", logging.Err(err))
		return Result{ExitCode: exitcodes.ACPExitDomain, Stderr: fmt.Sprintf("ERROR: %v\n", err)}
	}
	if err := maybeWriteCodexConfig(state); err != nil {
		logger.Error("workflow.config_write_failed", logging.Err(err))
		return Result{ExitCode: exitcodes.ACPExitRuntime, Stderr: fmt.Sprintf("ERROR: %v\n", err)}
	}

	logger.Info("workflow.complete", slog.String("mode", state.Options.Mode), slog.String("alias", state.GeneratedAlias))
	return Result{ExitCode: exitcodes.ACPExitSuccess, Stdout: renderSummary(state) + "Onboarding complete.\n"}
}

func prepareRunState(ctx context.Context, opts Options, prereqs prerequisites) (runState, error) {
	state := runState{
		Options:        opts,
		Prereqs:        prereqs,
		BaseURL:        buildBaseURL(opts.Host, opts.Port, opts.UseTLS),
		GeneratedAlias: opts.Alias,
	}
	if opts.Mode == "direct" {
		return state, nil
	}
	if opts.KeyGenerator == nil {
		opts.KeyGenerator = GatewayKeyGenerator{MasterKey: prereqs.MasterKey}
		state.Options = opts
	}
	generated, err := state.Options.KeyGenerator.Generate(ctx, KeyRequest{
		Alias:  state.Options.Alias,
		Budget: state.Options.Budget,
		Host:   state.Options.Host,
		Port:   state.Options.Port,
	})
	if err != nil {
		return runState{}, err
	}
	state.GeneratedAlias = generated.Alias
	state.KeyValue = generated.Key
	return state, nil
}

func withDefaults(opts Options) Options {
	if opts.Stdout == nil {
		opts.Stdout = io.Discard
	}
	if opts.Stderr == nil {
		opts.Stderr = io.Discard
	}
	if opts.Now == nil {
		opts.Now = func() time.Time { return time.Now() }
	}
	if opts.HTTPClient == nil {
		opts.HTTPClient = &http.Client{Timeout: 5 * time.Second}
	}
	return opts
}

func buildBaseURL(host string, port string, useTLS bool) string {
	scheme := "http"
	if useTLS {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s:%s", scheme, host, port)
}

func (g GatewayKeyGenerator) Generate(ctx context.Context, req KeyRequest) (GeneratedKey, error) {
	budget, err := strconv.ParseFloat(req.Budget, 64)
	if err != nil {
		return GeneratedKey{}, fmt.Errorf("invalid budget: %s", req.Budget)
	}
	port, err := strconv.Atoi(req.Port)
	if err != nil {
		return GeneratedKey{}, fmt.Errorf("invalid port: %s", req.Port)
	}
	client := gateway.NewClient(
		gateway.WithHost(req.Host),
		gateway.WithPort(port),
		gateway.WithMasterKey(g.MasterKey),
	)
	resp, err := client.GenerateKey(ctx, &gateway.GenerateKeyRequest{
		KeyAlias:       req.Alias,
		MaxBudget:      budget,
		BudgetDuration: "30d",
		Models:         keygen.GetModelsForRole("developer"),
	})
	if err == nil {
		return GeneratedKey{Alias: req.Alias, Key: resp.ExtractKey()}, nil
	}
	if !strings.Contains(strings.ToLower(err.Error()), "already exists") {
		return GeneratedKey{}, err
	}
	retryAlias := fmt.Sprintf("%s-%s", req.Alias, time.Now().Format("20060102150405"))
	resp, retryErr := client.GenerateKey(ctx, &gateway.GenerateKeyRequest{
		KeyAlias:       retryAlias,
		MaxBudget:      budget,
		BudgetDuration: "30d",
		Models:         keygen.GetModelsForRole("developer"),
	})
	if retryErr != nil {
		return GeneratedKey{}, retryErr
	}
	return GeneratedKey{Alias: retryAlias, Key: resp.ExtractKey()}, nil
}

func fprintf(w io.Writer, format string, args ...any) {
	if w == nil {
		return
	}
	_, _ = fmt.Fprintf(w, format, args...)
}
