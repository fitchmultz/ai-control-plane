// Package onboard implements tool onboarding workflows.
//
// Purpose:
//
//	Own typed onboarding behavior for CLI and IDE tools that route through the
//	AI Control Plane gateway.
//
// Responsibilities:
//   - Parse onboarding options and resolve tool-specific defaults.
//   - Load required secrets from environment or demo/.env safely.
//   - Generate virtual keys, verify gateway connectivity, and render exports.
//   - Optionally write Codex configuration for LiteLLM-backed profiles.
//
// Scope:
//   - Local onboarding orchestration and output rendering only.
//
// Usage:
//   - Called by `acpctl onboard`.
//
// Invariants/Assumptions:
//   - Secrets are redacted unless explicitly requested.
//   - demo/.env is treated as data, never sourced for execution.
package onboard

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/envfile"
	"github.com/mitchfultz/ai-control-plane/internal/gateway"
	"github.com/mitchfultz/ai-control-plane/internal/keygen"
)

const (
	DefaultHost   = "127.0.0.1"
	DefaultPort   = "4000"
	DefaultBudget = "10.00"
)

type ExitCode int

const (
	ExitSuccess ExitCode = 0
	ExitDomain  ExitCode = 1
	ExitPrereq  ExitCode = 2
	ExitRuntime ExitCode = 3
	ExitUsage   ExitCode = 64
)

type Options struct {
	RepoRoot     string
	Tool         string
	Mode         string
	Alias        string
	Budget       string
	Model        string
	Host         string
	Port         string
	UseTLS       bool
	Verify       bool
	WriteConfig  bool
	ShowKey      bool
	Stdout       io.Writer
	Stderr       io.Writer
	Now          func() time.Time
	KeyGenerator KeyGenerator
	HTTPClient   *http.Client
}

type Result struct {
	ExitCode ExitCode
}

type KeyGenerator interface {
	Generate(ctx context.Context, req KeyRequest) (GeneratedKey, error)
}

type KeyRequest struct {
	Alias  string
	Budget string
	Host   string
	Port   string
}

type GeneratedKey struct {
	Alias string
	Key   string
}

type GatewayKeyGenerator struct {
	MasterKey string
}

func Run(ctx context.Context, opts Options) Result {
	opts = withDefaults(opts)
	if opts.Tool == "" {
		printMainHelp(opts.Stdout)
		return Result{ExitCode: ExitSuccess}
	}
	if opts.Tool == "help" || opts.Tool == "--help" || opts.Tool == "-h" {
		printMainHelp(opts.Stdout)
		return Result{ExitCode: ExitSuccess}
	}

	if err := validateTool(opts.Tool); err != nil {
		fprintf(opts.Stderr, "ERROR: %v\n", err)
		return Result{ExitCode: ExitUsage}
	}

	if opts.Mode == "help" {
		printMainHelp(opts.Stdout)
		printToolHelp(opts.Stdout, opts.Tool)
		return Result{ExitCode: ExitSuccess}
	}

	if err := resolveToolDefaults(&opts); err != nil {
		fprintf(opts.Stderr, "ERROR: %v\n", err)
		return Result{ExitCode: ExitUsage}
	}
	if err := validateMode(opts.Tool, opts.Mode); err != nil {
		fprintf(opts.Stderr, "ERROR: %v\n", err)
		return Result{ExitCode: ExitUsage}
	}

	if err := ensurePrereqs(opts); err != nil {
		fprintf(opts.Stderr, "ERROR: %v\n", err)
		return Result{ExitCode: ExitPrereq}
	}

	masterKey, err := loadRequiredMasterKey(opts)
	if err != nil {
		fprintf(opts.Stderr, "ERROR: %v\n", err)
		return Result{ExitCode: ExitPrereq}
	}

	if opts.Mode == "subscription" && opts.Tool == "codex" {
		healthCode, probeErr := probeStatus(ctx, buildBaseURL(opts.Host, opts.Port, opts.UseTLS)+"/health", "", opts.HTTPClient)
		if probeErr != nil || (healthCode != http.StatusOK && healthCode != http.StatusUnauthorized) {
			fprintf(opts.Stderr, "WARN: Gateway health is not ready for subscription mode (HTTP %d).\n", healthCode)
			fprintf(opts.Stderr, "WARN: Complete ChatGPT device login first: make chatgpt-login\n")
			return Result{ExitCode: ExitDomain}
		}
	}

	keyValue := ""
	generatedAlias := opts.Alias
	if opts.Mode != "direct" {
		if opts.KeyGenerator == nil {
			opts.KeyGenerator = GatewayKeyGenerator{MasterKey: masterKey}
		}
		generated, generateErr := opts.KeyGenerator.Generate(ctx, KeyRequest{
			Alias:  opts.Alias,
			Budget: opts.Budget,
			Host:   opts.Host,
			Port:   opts.Port,
		})
		if generateErr != nil {
			fprintf(opts.Stderr, "ERROR: %v\n", generateErr)
			return Result{ExitCode: ExitDomain}
		}
		generatedAlias = generated.Alias
		keyValue = generated.Key
	}

	baseURL := buildBaseURL(opts.Host, opts.Port, opts.UseTLS)
	fprintf(opts.Stdout, "\nTool: %s\n", opts.Tool)
	fprintf(opts.Stdout, "Mode: %s\n", opts.Mode)
	fprintf(opts.Stdout, "Gateway: %s\n", baseURL)
	fprintf(opts.Stdout, "Model: %s\n", opts.Model)
	if keyValue != "" {
		fprintf(opts.Stdout, "Key alias: %s\n", generatedAlias)
	}
	fprintf(opts.Stdout, "\n")

	if opts.Mode == "subscription" && opts.Tool == "codex" {
		fprintf(opts.Stdout, "INFO: Run 'make chatgpt-login' on this gateway host before launching Codex.\n\n")
	}

	printExports(opts.Stdout, opts.Tool, opts.Mode, baseURL, keyValue, opts.Model, opts.Host, opts.ShowKey)
	fprintf(opts.Stdout, "\n")

	if opts.WriteConfig && opts.Tool == "codex" && opts.Mode != "direct" {
		if err := writeCodexConfig(opts, baseURL); err != nil {
			fprintf(opts.Stderr, "ERROR: %v\n", err)
			return Result{ExitCode: ExitRuntime}
		}
		fprintf(opts.Stdout, "\n")
	}

	if opts.Verify {
		if opts.Mode == "direct" {
			code, verifyErr := probeStatus(ctx, fmt.Sprintf("http://%s:4318/health", opts.Host), "", opts.HTTPClient)
			if verifyErr != nil || code != http.StatusOK {
				fprintf(opts.Stderr, "WARN: OTEL collector check returned HTTP %d at http://%s:4318/health\n", code, opts.Host)
				return Result{ExitCode: ExitDomain}
			}
			fprintf(opts.Stdout, "INFO: OTEL collector health check passed\n")
		} else {
			healthCode, healthErr := probeStatus(ctx, baseURL+"/health", "", opts.HTTPClient)
			if healthErr != nil || (healthCode != http.StatusOK && healthCode != http.StatusUnauthorized) {
				fprintf(opts.Stderr, "ERROR: Gateway /health returned HTTP %d\n", healthCode)
				return Result{ExitCode: ExitDomain}
			}
			modelCode, modelErr := probeStatus(ctx, baseURL+"/v1/models", keyValue, opts.HTTPClient)
			if modelErr != nil || modelCode != http.StatusOK {
				fprintf(opts.Stderr, "ERROR: Authorized /v1/models check returned HTTP %d\n", modelCode)
				return Result{ExitCode: ExitDomain}
			}
			fprintf(opts.Stdout, "INFO: Gateway health and authorized model checks passed\n")
		}
	}

	fprintf(opts.Stdout, "Onboarding complete.\n")
	return Result{ExitCode: ExitSuccess}
}

func ParseArgs(args []string, repoRoot string, stdout io.Writer, stderr io.Writer) (Options, error) {
	opts := Options{
		RepoRoot: repoRoot,
		Stdout:   stdout,
		Stderr:   stderr,
	}
	if len(args) == 0 {
		return opts, nil
	}
	opts.Tool = args[0]
	if isHelpToken(opts.Tool) {
		return opts, nil
	}
	args = args[1:]

	opts.Alias = opts.Tool + "-cli"
	opts.Budget = DefaultBudget
	opts.Host = DefaultHost
	opts.Port = DefaultPort

	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "--mode":
			index++
			if index >= len(args) {
				return Options{}, errors.New("missing value for --mode")
			}
			opts.Mode = args[index]
		case "--alias":
			index++
			if index >= len(args) {
				return Options{}, errors.New("missing value for --alias")
			}
			opts.Alias = args[index]
		case "--budget":
			index++
			if index >= len(args) {
				return Options{}, errors.New("missing value for --budget")
			}
			opts.Budget = args[index]
		case "--model":
			index++
			if index >= len(args) {
				return Options{}, errors.New("missing value for --model")
			}
			opts.Model = args[index]
		case "--host":
			index++
			if index >= len(args) {
				return Options{}, errors.New("missing value for --host")
			}
			opts.Host = args[index]
		case "--port":
			index++
			if index >= len(args) {
				return Options{}, errors.New("missing value for --port")
			}
			opts.Port = args[index]
		case "--tls":
			opts.UseTLS = true
		case "--verify":
			opts.Verify = true
		case "--write-config":
			opts.WriteConfig = true
		case "--show-key":
			opts.ShowKey = true
		case "--help", "-h":
			opts.Mode = "help"
		default:
			return Options{}, fmt.Errorf("unknown option: %s", args[index])
		}
	}

	return opts, nil
}

func printMainHelp(w io.Writer) {
	fprintf(w, "Usage: acpctl onboard <tool> [options]\n\n")
	fprintf(w, "Tools:\n")
	fprintf(w, "  codex\n")
	fprintf(w, "  claude\n")
	fprintf(w, "  opencode\n")
	fprintf(w, "  cursor\n")
	fprintf(w, "  copilot\n\n")
	fprintf(w, "Options:\n")
	fprintf(w, "  --mode <mode>          auth mode (tool-dependent)\n")
	fprintf(w, "  --alias <alias>        virtual key alias (default: <tool>-cli)\n")
	fprintf(w, "  --budget <usd>         key budget in USD (default: 10.00)\n")
	fprintf(w, "  --model <model>        model alias override\n")
	fprintf(w, "  --host <host>          gateway host (default: 127.0.0.1)\n")
	fprintf(w, "  --port <port>          gateway port (default: 4000)\n")
	fprintf(w, "  --tls                  use https for base URL\n")
	fprintf(w, "  --verify               run authorized gateway checks\n")
	fprintf(w, "  --write-config         write ~/.codex/config.toml (Codex only)\n")
	fprintf(w, "  --show-key             print full key value\n")
	fprintf(w, "  --help, -h             show help\n\n")
	fprintf(w, "Codex modes:\n")
	fprintf(w, "  subscription           routed through gateway; upstream via ChatGPT provider (default)\n")
	fprintf(w, "  api-key                routed through gateway; upstream via API-key providers\n")
	fprintf(w, "  direct                 no gateway routing; OTEL visibility only\n\n")
	fprintf(w, "Examples:\n")
	fprintf(w, "  acpctl onboard codex --mode subscription --verify\n")
	fprintf(w, "  acpctl onboard codex --mode api-key --write-config\n")
	fprintf(w, "  acpctl onboard claude --mode api-key --verify\n")
}

func printToolHelp(w io.Writer, tool string) {
	switch tool {
	case "codex":
		fprintf(w, "Codex notes:\n")
		fprintf(w, "  - For subscription mode, run `make chatgpt-login` on the gateway host first.\n")
		fprintf(w, "  - Codex uses OPENAI_BASE_URL without /v1.\n")
		fprintf(w, "  - --write-config writes ~/.codex/config.toml for a LiteLLM provider profile.\n")
	case "claude":
		fprintf(w, "Claude notes:\n")
		fprintf(w, "  - Exports ANTHROPIC_BASE_URL and ANTHROPIC_API_KEY for gateway routing.\n")
		fprintf(w, "  - Keep mode=api-key unless you have a separate subscription OAuth flow configured.\n")
	case "opencode", "cursor", "copilot":
		fprintf(w, "OpenAI-compatible tool notes:\n")
		fprintf(w, "  - Exports OPENAI_BASE_URL and OPENAI_API_KEY for gateway routing.\n")
	}
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

func validateTool(tool string) error {
	switch tool {
	case "codex", "claude", "opencode", "cursor", "copilot":
		return nil
	default:
		return fmt.Errorf("unsupported tool: %s", tool)
	}
}

func resolveToolDefaults(opts *Options) error {
	if opts.Mode == "" {
		switch opts.Tool {
		case "codex":
			opts.Mode = "subscription"
		default:
			opts.Mode = "api-key"
		}
	}
	if opts.Model != "" {
		return nil
	}
	switch opts.Tool {
	case "codex":
		if opts.Mode == "subscription" {
			opts.Model = "chatgpt-gpt5.3-codex"
		} else {
			opts.Model = "openai-gpt5.2"
		}
	case "claude":
		opts.Model = "claude-haiku-4-5"
	default:
		opts.Model = "openai-gpt5.2"
	}
	return nil
}

func validateMode(tool string, mode string) error {
	if mode == "direct" && tool != "codex" {
		return errors.New("mode 'direct' is only supported for codex")
	}
	switch tool {
	case "codex":
		switch mode {
		case "subscription", "api-key", "direct":
			return nil
		}
	default:
		if mode == "api-key" {
			return nil
		}
	}
	return fmt.Errorf("unsupported mode %q for %s", mode, tool)
}

func ensurePrereqs(opts Options) error {
	if opts.RepoRoot == "" {
		return errors.New("repo root is required")
	}
	envPath := filepath.Join(opts.RepoRoot, "demo", ".env")
	if _, err := os.Stat(envPath); err != nil {
		return fmt.Errorf("missing %s. Run: make install-env", envPath)
	}
	return nil
}

func loadRequiredMasterKey(opts Options) (string, error) {
	if value := strings.TrimSpace(os.Getenv("LITELLM_MASTER_KEY")); value != "" {
		return value, nil
	}
	envPath := filepath.Join(opts.RepoRoot, "demo", ".env")
	value, ok, err := envfile.LookupFile(envPath, "LITELLM_MASTER_KEY")
	if err != nil {
		return "", err
	}
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("LITELLM_MASTER_KEY is not set (%s)", envPath)
	}
	return strings.TrimSpace(value), nil
}

func buildBaseURL(host string, port string, useTLS bool) string {
	scheme := "http"
	if useTLS {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s:%s", scheme, host, port)
}

func probeStatus(ctx context.Context, url string, key string, client *http.Client) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	if strings.TrimSpace(key) != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

func printExports(w io.Writer, tool string, mode string, baseURL string, key string, model string, host string, showKey bool) {
	printedKey := redactKey(key)
	if showKey {
		printedKey = key
	}
	if tool == "claude" {
		fprintf(w, "export ANTHROPIC_BASE_URL=\"%s\"\n", baseURL)
		fprintf(w, "export ANTHROPIC_API_KEY=\"%s\"\n", printedKey)
		fprintf(w, "export ANTHROPIC_MODEL=\"%s\"\n", model)
		return
	}
	if mode == "direct" {
		fprintf(w, "export OTEL_EXPORTER_OTLP_ENDPOINT=\"http://%s:4317\"\n", host)
		fprintf(w, "export OTEL_EXPORTER_OTLP_PROTOCOL=\"grpc\"\n")
		fprintf(w, "export OTEL_SERVICE_NAME=\"codex-cli\"\n")
		return
	}
	fprintf(w, "export OPENAI_BASE_URL=\"%s\"\n", baseURL)
	fprintf(w, "export OPENAI_API_KEY=\"%s\"\n", printedKey)
	fprintf(w, "export OPENAI_MODEL=\"%s\"\n", model)
}

func redactKey(key string) string {
	if len(key) <= 12 {
		return "***"
	}
	return key[:8] + "..." + key[len(key)-4:]
}

func writeCodexConfig(opts Options, baseURL string) error {
	configDir := filepath.Join(os.Getenv("HOME"), ".codex")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}
	configPath := filepath.Join(configDir, "config.toml")
	if data, err := os.ReadFile(configPath); err == nil {
		backupPath := fmt.Sprintf("%s.bak.%s", configPath, opts.Now().Format("20060102150405"))
		if err := os.WriteFile(backupPath, data, 0o644); err != nil {
			return err
		}
	}
	content := fmt.Sprintf("model = %q\nmodel_provider = %q\n\n[model_providers.litellm]\nname = %q\nbase_url = %q\nwire_api = %q\nenv_key = %q\n",
		opts.Model,
		"litellm",
		"LiteLLM",
		baseURL+"/v1",
		"responses",
		"OPENAI_API_KEY",
	)
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		return err
	}
	fprintf(opts.Stdout, "INFO: Wrote %s\n", configPath)
	return nil
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

func isHelpToken(value string) bool {
	switch value {
	case "help", "--help", "-h":
		return true
	default:
		return false
	}
}
