// Package doctor provides environment preflight diagnostics.
//
// Purpose:
//   - Adapt certificate lifecycle health into doctor output.
//
// Responsibilities:
//   - Reuse the shared certificate component from runtime inspection.
//   - Provide actionable renewal guidance.
//   - Trigger threshold-based renewal when `--fix` is used on TLS-enabled runtimes.
//
// Scope:
//   - Certificate lifecycle diagnostics only.
//
// Usage:
//   - Registered in `DefaultChecks()`.
//
// Invariants/Assumptions:
//   - Doctor must not duplicate certificate parsing logic.
package doctor

import (
	"context"

	"github.com/mitchfultz/ai-control-plane/internal/certlifecycle"
	sharedhealth "github.com/mitchfultz/ai-control-plane/internal/health"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

var renewCertificateLifecycle = certlifecycle.Renew

type certificateHealthyCheck struct{}

func (c certificateHealthyCheck) ID() string { return "certificate_healthy" }

func (c certificateHealthyCheck) Run(_ context.Context, opts Options) CheckResult {
	return runtimeComponentCheck(opts, c.ID(), "Certificate Healthy", "certificate", "Certificate", func(component status.ComponentStatus) CheckResult {
		result := componentCheckResult(c.ID(), "Certificate Healthy", component, severityForLevel(component.Level))
		if component.Level != sharedhealth.LevelHealthy {
			result.Suggestions = append(result.Suggestions,
				"List certificates: ./scripts/acpctl.sh cert list",
				"Inspect details: ./scripts/acpctl.sh cert inspect --domain <host>",
				"Run validation: ./scripts/acpctl.sh cert check",
				"Trigger renewal: ./scripts/acpctl.sh cert renew",
			)
		}
		return result
	})
}

func (c certificateHealthyCheck) Fix(ctx context.Context, opts Options) (bool, string, error) {
	if !opts.Gateway.TLSEnabled {
		return false, "", nil
	}
	_, err := renewCertificateLifecycle(ctx, certlifecycle.NewStore(opts.RepoRoot), certlifecycle.RenewalRequest{
		RepoRoot:      opts.RepoRoot,
		Domain:        opts.Gateway.Host,
		BaseURL:       opts.Gateway.BaseURL,
		ThresholdDays: certlifecycle.DefaultRenewThresholdDays,
	})
	if err != nil {
		return false, "", err
	}
	return true, "Triggered threshold-based certificate renewal", nil
}
