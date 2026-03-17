// Package onboard implements tool onboarding workflows.
//
// Purpose:
//
//	Coordinate typed onboarding layers for prompting, defaults,
//	prerequisite loading, key generation, export rendering, verification, and
//	config writes.
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

	"github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/gateway"
	"github.com/mitchfultz/ai-control-plane/internal/keygen"
	"github.com/mitchfultz/ai-control-plane/internal/logging"
)

type GatewayKeyGenerator struct {
	MasterKey  string
	HTTPClient *http.Client
}

func Run(ctx context.Context, opts Options) Result {
	opts = withDefaults(opts)
	logger := logging.FromContext(ctx).With(slog.String("component", "onboard"))

	var err error
	if opts.Stdin != nil {
		opts, err = promptForWizardOptions(opts)
		if err != nil {
			logger.Error("wizard.prompt_failed", logging.Err(err))
			return Result{ExitCode: exitcodes.ACPExitUsage, Stderr: fmt.Sprintf("ERROR: %v\n", err)}
		}
	}
	if strings.TrimSpace(opts.Tool) == "" {
		return Result{ExitCode: exitcodes.ACPExitUsage, Stderr: "ERROR: tool is required\n"}
	}

	logger = logger.With(slog.String("tool", opts.Tool), slog.String("mode", opts.Mode))
	logger.Info("workflow.start")

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

	state.ToolConfig, err = maybeWriteCodexConfig(state)
	if err != nil {
		logger.Error("workflow.config_write_failed", logging.Err(err))
		return Result{ExitCode: exitcodes.ACPExitRuntime, Stderr: fmt.Sprintf("ERROR: %v\n", err)}
	}

	state.Verification = verifyOnboarding(ctx, state)

	var stdout strings.Builder
	stdout.WriteString(renderSummary(state))
	if state.Verification.HasFailures() {
		logger.Warn("workflow.verification_failed", slog.Int("issues", len(state.Verification.Issues)))
		stdout.WriteString("Onboarding incomplete.\n")
		return Result{
			ExitCode: exitcodes.ACPExitDomain,
			Stdout:   stdout.String(),
			Stderr:   "ERROR: onboarding verification failed; review the verification section above for remediation.\n",
		}
	}

	stdout.WriteString(renderFullKeyReveal(state))
	stdout.WriteString("Onboarding complete.\n")

	logger.Info("workflow.complete", slog.String("alias", state.GeneratedAlias))
	return Result{ExitCode: exitcodes.ACPExitSuccess, Stdout: stdout.String()}
}

func prepareRunState(ctx context.Context, opts Options, prereqs prerequisites) (runState, error) {
	gatewaySettings := config.ResolveGatewaySettings(config.GatewayResolveInput{
		Host: opts.Host,
		Port: opts.Port,
		TLS:  strconv.FormatBool(opts.UseTLS),
	})

	state := runState{
		Options:        opts,
		Prereqs:        prereqs,
		Gateway:        gatewaySettings,
		GeneratedAlias: opts.Alias,
	}
	if opts.Mode == "direct" {
		return state, nil
	}
	if opts.KeyGenerator == nil {
		opts.KeyGenerator = GatewayKeyGenerator{MasterKey: prereqs.MasterKey, HTTPClient: opts.HTTPClient}
		state.Options = opts
	}
	generated, err := state.Options.KeyGenerator.Generate(ctx, KeyRequest{
		Alias:   state.Options.Alias,
		Budget:  state.Options.Budget,
		BaseURL: state.Gateway.BaseURL,
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
		opts.HTTPClient = &http.Client{Timeout: config.DefaultHTTPTimeout}
	}

	loader := config.NewLoader()
	includeRepoFallback := false
	if strings.TrimSpace(opts.RepoRoot) != "" {
		loader = loader.WithRepoRoot(opts.RepoRoot)
		includeRepoFallback = true
	}
	gatewaySettings := loader.Gateway(includeRepoFallback)
	if strings.TrimSpace(opts.Host) == "" {
		opts.Host = gatewaySettings.Host
	}
	if strings.TrimSpace(opts.Port) == "" {
		opts.Port = gatewaySettings.Port
	}
	if !opts.UseTLS && gatewaySettings.TLSEnabled {
		opts.UseTLS = true
	}
	return opts
}

func (g GatewayKeyGenerator) Generate(ctx context.Context, req KeyRequest) (GeneratedKey, error) {
	budget, err := strconv.ParseFloat(req.Budget, 64)
	if err != nil {
		return GeneratedKey{}, fmt.Errorf("invalid budget: %s", req.Budget)
	}
	client := gateway.NewClient(
		gateway.WithBaseURL(req.BaseURL),
		gateway.WithMasterKey(g.MasterKey),
		gateway.WithHTTPClient(g.HTTPClient),
	)
	plan, err := keygen.PlanGenerateRequest(keygen.GenerateRequestConfig{
		Alias:    req.Alias,
		Budget:   budget,
		Duration: "30d",
		Role:     "developer",
	})
	if err != nil {
		return GeneratedKey{}, err
	}

	resp, err := client.GenerateKey(ctx, &plan.Request)
	if err == nil {
		return GeneratedKey{Alias: req.Alias, Key: resp.ExtractKey()}, nil
	}
	if !strings.Contains(strings.ToLower(err.Error()), "already exists") {
		return GeneratedKey{}, err
	}
	retryAlias := fmt.Sprintf("%s-%s", req.Alias, time.Now().Format("20060102150405"))
	retryPlan := plan.WithAlias(retryAlias)
	resp, retryErr := client.GenerateKey(ctx, &retryPlan.Request)
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
