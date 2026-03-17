// cmd_cert.go - Typed certificate lifecycle command handlers.
//
// Purpose:
//   - Implement operator-facing certificate list, inspect, check, renew, and
//   - renewal-automation flows.
//
// Responsibilities:
//   - Bind parsed CLI input into typed certificate lifecycle requests.
//   - Render human and JSON outputs with stable ACP exit codes.
//   - Install the supported host-first certificate renewal timer.
//
// Scope:
//   - `acpctl cert ...` only.
//
// Usage:
//   - Invoked through the typed `cert` command tree.
//
// Invariants/Assumptions:
//   - Certificate operations target the supported Caddy host-first TLS path.
//   - Renewal preserves rollback artifacts under `demo/logs/cert-renewals`.
package main

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/certlifecycle"
	acpconfig "github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/output"
)

type certDomainOptions struct {
	Domain string
	JSON   bool
}

type certCheckOptions struct {
	Domain        string
	ThresholdDays int
	CriticalDays  int
	JSON          bool
}

type certRenewOptions struct {
	Domain        string
	ThresholdDays int
	Force         bool
	DryRun        bool
	JSON          bool
	OutputDir     string
}

type certRenewAutoOptions struct {
	EnvFile         string
	ServiceUser     string
	ServiceGroup    string
	OnCalendar      string
	RandomizedDelay string
	ThresholdDays   int
	JSON            bool
}

func certCommandSpec() *commandSpec {
	return &commandSpec{
		Name:        "cert",
		Summary:     "TLS certificate lifecycle operations",
		Description: "Inspect, validate, renew, and automate Caddy-managed TLS certificates.",
		Examples: []string{
			"acpctl cert list",
			"acpctl cert inspect --domain gateway.example.com",
			"acpctl cert check --threshold-days 30",
			"acpctl cert renew --domain gateway.example.com",
			"sudo acpctl cert renew-auto --env-file /etc/ai-control-plane/secrets.env",
		},
		Children: []*commandSpec{
			newNativeCommandSpec(nativeCommandConfig{
				Name:        "list",
				Summary:     "List tracked TLS certificates",
				Description: "List tracked TLS certificates from the Caddy data store.",
				Options:     []commandOptionSpec{{Name: "json", Summary: "Output JSON", Type: optionValueBool}},
				Bind: bindParsedValue(func(input parsedCommandInput) certDomainOptions {
					return certDomainOptions{JSON: input.Bool("json")}
				}),
				Run: runCertList,
			}),
			newNativeCommandSpec(nativeCommandConfig{
				Name:        "inspect",
				Summary:     "Inspect one certificate",
				Description: "Inspect one certificate by domain or, when only one exists, inspect the lone stored certificate.",
				Options: []commandOptionSpec{
					{Name: "domain", ValueName: "HOST", Summary: "Certificate domain", Type: optionValueString},
					{Name: "json", Summary: "Output JSON", Type: optionValueBool},
				},
				Bind: bindParsedValue(func(input parsedCommandInput) certDomainOptions {
					return certDomainOptions{Domain: input.NormalizedString("domain"), JSON: input.Bool("json")}
				}),
				Run: runCertInspect,
			}),
			newNativeCommandSpec(nativeCommandConfig{
				Name:        "check",
				Summary:     "Validate certificate expiry and live TLS state",
				Description: "Validate certificate expiry thresholds and the live served TLS certificate.",
				Options: []commandOptionSpec{
					{Name: "domain", ValueName: "HOST", Summary: "Certificate domain", Type: optionValueString},
					{Name: "threshold-days", ValueName: "N", Summary: "Warning threshold", Type: optionValueInt, DefaultText: "30"},
					{Name: "critical-days", ValueName: "N", Summary: "Failure threshold", Type: optionValueInt, DefaultText: "7"},
					{Name: "json", Summary: "Output JSON", Type: optionValueBool},
				},
				Bind: bindParsed(func(input parsedCommandInput) (certCheckOptions, error) {
					warningDays, err := input.IntDefault("threshold-days", certlifecycle.DefaultWarningDays)
					if err != nil {
						return certCheckOptions{}, fmt.Errorf("invalid threshold-days")
					}
					criticalDays, err := input.IntDefault("critical-days", certlifecycle.DefaultCriticalDays)
					if err != nil {
						return certCheckOptions{}, fmt.Errorf("invalid critical-days")
					}
					return certCheckOptions{
						Domain:        input.NormalizedString("domain"),
						ThresholdDays: warningDays,
						CriticalDays:  criticalDays,
						JSON:          input.Bool("json"),
					}, nil
				}),
				Run: runCertCheck,
			}),
			newNativeCommandSpec(nativeCommandConfig{
				Name:        "renew",
				Summary:     "Trigger controlled certificate reissuance",
				Description: "Trigger controlled certificate reissuance with rollback artifacts preserved.",
				Options: []commandOptionSpec{
					{Name: "domain", ValueName: "HOST", Summary: "Certificate domain", Type: optionValueString},
					{Name: "threshold-days", ValueName: "N", Summary: "Only renew certificates at or below threshold days", Type: optionValueInt, DefaultText: "30"},
					{Name: "force", Summary: "Renew even when above threshold", Type: optionValueBool},
					{Name: "dry-run", Summary: "Preview renewal candidates without changing storage", Type: optionValueBool},
					{Name: "output-dir", ValueName: "DIR", Summary: "Renewal artifact output root", Type: optionValueString, DefaultText: "demo/logs/cert-renewals"},
					{Name: "json", Summary: "Output JSON", Type: optionValueBool},
				},
				Bind: bindRepoParsed(func(bindCtx commandBindContext, input parsedCommandInput) (certRenewOptions, error) {
					repoRoot, err := requireCommandRepoRoot(bindCtx)
					if err != nil {
						return certRenewOptions{}, err
					}
					thresholdDays, err := input.IntDefault("threshold-days", certlifecycle.DefaultRenewThresholdDays)
					if err != nil {
						return certRenewOptions{}, fmt.Errorf("invalid threshold-days")
					}
					return certRenewOptions{
						Domain:        input.NormalizedString("domain"),
						ThresholdDays: thresholdDays,
						Force:         input.Bool("force"),
						DryRun:        input.Bool("dry-run"),
						JSON:          input.Bool("json"),
						OutputDir:     resolveRepoInput(repoRoot, input.StringDefault("output-dir", "demo/logs/cert-renewals")),
					}, nil
				}),
				Run: runCertRenew,
			}),
			newNativeCommandSpec(nativeCommandConfig{
				Name:        "renew-auto",
				Summary:     "Install the automatic certificate renewal timer",
				Description: "Render and enable the supported host-first certificate renewal timer.",
				Options: []commandOptionSpec{
					{Name: "env-file", ValueName: "PATH", Summary: "Canonical secrets/env file", Type: optionValueString, DefaultText: defaultHostSecretsEnvFile},
					{Name: "service-user", ValueName: "USER", Summary: "Systemd service user", Type: optionValueString},
					{Name: "service-group", ValueName: "GROUP", Summary: "Systemd service group", Type: optionValueString},
					{Name: "on-calendar", ValueName: "CALENDAR", Summary: "Systemd timer calendar", Type: optionValueString, DefaultText: "daily"},
					{Name: "randomized-delay", ValueName: "DURATION", Summary: "Randomized delay", Type: optionValueString, DefaultText: "30m"},
					{Name: "threshold-days", ValueName: "N", Summary: "Renewal threshold", Type: optionValueInt, DefaultText: "30"},
					{Name: "json", Summary: "Output JSON", Type: optionValueBool},
				},
				Bind: bindParsed(func(input parsedCommandInput) (certRenewAutoOptions, error) {
					thresholdDays, err := input.IntDefault("threshold-days", certlifecycle.DefaultRenewThresholdDays)
					if err != nil {
						return certRenewAutoOptions{}, fmt.Errorf("invalid threshold-days")
					}
					return certRenewAutoOptions{
						EnvFile:         input.StringDefault("env-file", defaultHostSecretsEnvFile),
						ServiceUser:     input.NormalizedString("service-user"),
						ServiceGroup:    input.NormalizedString("service-group"),
						OnCalendar:      input.StringDefault("on-calendar", "daily"),
						RandomizedDelay: input.StringDefault("randomized-delay", "30m"),
						ThresholdDays:   thresholdDays,
						JSON:            input.Bool("json"),
					}, nil
				}),
				Run: runCertRenewAuto,
			}),
		},
	}
}

func runCertList(ctx context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(certDomainOptions)
	store := certlifecycle.NewStore(runCtx.RepoRoot)
	certs, err := store.List(ctx)
	if err != nil {
		return writeCertError(runCtx, err)
	}
	if opts.JSON {
		return writeJSONOutput(runCtx, certs)
	}
	printCertList(runCtx.Stdout, certs)
	return exitcodes.ACPExitSuccess
}

func runCertInspect(ctx context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(certDomainOptions)
	store := certlifecycle.NewStore(runCtx.RepoRoot)
	certs, err := store.List(ctx)
	if err != nil {
		return writeCertError(runCtx, err)
	}
	selected := filterCertDomain(certs, opts.Domain)
	switch {
	case len(selected) == 0:
		fmt.Fprintf(runCtx.Stderr, "Error: no certificate found for %q\n", opts.Domain)
		return exitcodes.ACPExitUsage
	case strings.TrimSpace(opts.Domain) == "" && len(selected) > 1:
		fmt.Fprintln(runCtx.Stderr, "Error: multiple certificates found; pass --domain to select one")
		return exitcodes.ACPExitUsage
	}
	if opts.JSON {
		return writeJSONOutput(runCtx, selected[0])
	}
	printCertInspect(runCtx.Stdout, selected[0])
	return exitcodes.ACPExitSuccess
}

func runCertCheck(ctx context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(certCheckOptions)
	gateway := acpconfig.NewLoader().WithRepoRoot(runCtx.RepoRoot).Gateway(true)
	result, err := certlifecycle.Check(ctx, certlifecycle.NewStore(runCtx.RepoRoot), certlifecycle.CheckRequest{
		Domain:       firstNonBlank(opts.Domain, gateway.Host),
		BaseURL:      gateway.BaseURL,
		WarningDays:  opts.ThresholdDays,
		CriticalDays: opts.CriticalDays,
		Now:          time.Now().UTC(),
	})
	if err != nil {
		return writeCertError(runCtx, err)
	}
	if opts.JSON {
		return writeJSONOutput(runCtx, result)
	}
	printCertCheck(runCtx.Stdout, result)
	if result.Status == certlifecycle.StatusHealthy {
		return exitcodes.ACPExitSuccess
	}
	return exitcodes.ACPExitDomain
}

func runCertRenew(ctx context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(certRenewOptions)
	gateway := acpconfig.NewLoader().WithRepoRoot(runCtx.RepoRoot).Gateway(true)
	result, err := certlifecycle.Renew(ctx, certlifecycle.NewStore(runCtx.RepoRoot), certlifecycle.RenewalRequest{
		RepoRoot:      runCtx.RepoRoot,
		Domain:        firstNonBlank(opts.Domain, gateway.Host),
		BaseURL:       gateway.BaseURL,
		ThresholdDays: opts.ThresholdDays,
		Force:         opts.Force,
		DryRun:        opts.DryRun,
		OutputRoot:    opts.OutputDir,
	})
	if err != nil {
		return writeCertError(runCtx, err)
	}
	if opts.JSON {
		return writeJSONOutput(runCtx, result)
	}
	printCertRenew(runCtx.Stdout, result)
	return exitcodes.ACPExitSuccess
}

func runCertRenewAuto(ctx context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(certRenewAutoOptions)
	result, err := certlifecycle.InstallAutoRenewTimer(ctx, certlifecycle.InstallTimerOptions{
		RepoRoot:        runCtx.RepoRoot,
		WorkingDir:      runCtx.RepoRoot,
		EnvFile:         opts.EnvFile,
		ServiceUser:     opts.ServiceUser,
		ServiceGroup:    opts.ServiceGroup,
		OnCalendar:      opts.OnCalendar,
		RandomizedDelay: opts.RandomizedDelay,
		ThresholdDays:   opts.ThresholdDays,
	})
	if err != nil {
		return writeCertError(runCtx, err)
	}
	if opts.JSON {
		return writeJSONOutput(runCtx, result)
	}
	fmt.Fprintf(runCtx.Stdout, "Installed %s\n", result.TimerName)
	printCommandDetail(runCtx.Stdout, "Service", result.ServicePath)
	printCommandDetail(runCtx.Stdout, "Timer", result.TimerPath)
	return exitcodes.ACPExitSuccess
}

func writeCertError(runCtx commandRunContext, err error) int {
	out := output.New()
	fmt.Fprintf(runCtx.Stderr, out.Fail("Certificate workflow failed: %v\n"), err)
	switch {
	case certlifecycle.IsKind(err, certlifecycle.ErrorKindPrereq):
		return exitcodes.ACPExitPrereq
	case certlifecycle.IsKind(err, certlifecycle.ErrorKindDomain):
		return exitcodes.ACPExitDomain
	default:
		return exitcodes.ACPExitRuntime
	}
}

func filterCertDomain(certs []certlifecycle.CertificateInfo, domain string) []certlifecycle.CertificateInfo {
	if strings.TrimSpace(domain) == "" {
		return append([]certlifecycle.CertificateInfo(nil), certs...)
	}
	filtered := make([]certlifecycle.CertificateInfo, 0, len(certs))
	for _, cert := range certs {
		if cert.MatchesDomain(domain) {
			filtered = append(filtered, cert)
		}
	}
	return filtered
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func printCertList(w io.Writer, certs []certlifecycle.CertificateInfo) {
	now := time.Now().UTC()
	fmt.Fprintf(w, "%-32s %-18s %-25s %-6s %s\n", "DOMAIN", "MANAGED_BY", "EXPIRES", "DAYS", "ISSUER")
	for _, cert := range certs {
		fmt.Fprintf(w, "%-32s %-18s %-25s %-6d %s\n",
			cert.PrimaryName(),
			cert.ManagedBy,
			cert.NotAfter.Format(time.RFC3339),
			cert.DaysRemaining(now),
			cert.Issuer,
		)
	}
}

func printCertInspect(w io.Writer, cert certlifecycle.CertificateInfo) {
	printCommandSection(w, output.New(), "Certificate")
	printCommandDetail(w, "Domain", cert.PrimaryName())
	printCommandDetail(w, "Names", strings.Join(cert.AllNames(), ", "))
	printCommandDetail(w, "Managed by", cert.ManagedBy)
	printCommandDetail(w, "Issuer", cert.Issuer)
	printCommandDetail(w, "Subject", cert.Subject)
	printCommandDetail(w, "Serial", cert.SerialNumber)
	printCommandDetail(w, "Not before", cert.NotBefore.Format(time.RFC3339))
	printCommandDetail(w, "Not after", cert.NotAfter.Format(time.RFC3339))
	printCommandDetail(w, "Fingerprint", cert.FingerprintSHA256)
	if strings.TrimSpace(cert.StoragePath) != "" {
		printCommandDetail(w, "Storage path", cert.StoragePath)
	}
}

func printCertCheck(w io.Writer, result certlifecycle.CheckResult) {
	printCommandSection(w, output.New(), "Certificate check")
	printCommandDetail(w, "Status", result.Status)
	printCommandDetail(w, "Message", result.Message)
	if len(result.Certificates) > 0 {
		printCommandDetail(w, "Domain", result.Certificates[0].PrimaryName())
		printCommandDetail(w, "Expires", result.Certificates[0].NotAfter.Format(time.RFC3339))
		printCommandDetail(w, "Days remaining", result.Certificates[0].DaysRemaining(result.CheckedAt))
	}
	if result.LiveCertificate != nil {
		printCommandDetail(w, "Live fingerprint", result.LiveCertificate.FingerprintSHA256)
	}
	if strings.TrimSpace(result.ValidationError) != "" {
		printCommandDetail(w, "Validation error", result.ValidationError)
	}
	for _, suggestion := range result.Suggestions {
		fmt.Fprintf(w, "  - %s\n", suggestion)
	}
}

func printCertRenew(w io.Writer, result certlifecycle.RenewalResult) {
	printCommandSection(w, output.New(), "Certificate renewal")
	printCommandDetail(w, "Renewed", result.Renewed)
	if strings.TrimSpace(result.RunDirectory) != "" {
		printCommandDetail(w, "Run directory", result.RunDirectory)
	}
	for _, item := range result.Items {
		fmt.Fprintf(w, "  - %s: renewed=%t", item.Domain, item.Renewed)
		if item.After != nil {
			fmt.Fprintf(w, " new_expiry=%s", item.After.NotAfter.Format(time.RFC3339))
		}
		fmt.Fprintln(w)
	}
	for _, suggestion := range result.Suggestions {
		fmt.Fprintf(w, "  - %s\n", suggestion)
	}
}
