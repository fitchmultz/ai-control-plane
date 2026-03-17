// Package collectors provides status collectors backed by typed ACP services.
//
// Purpose:
//   - Expose certificate lifecycle status for the supported Caddy-managed TLS path.
//
// Responsibilities:
//   - Adapt typed certificate lifecycle health into status.ComponentStatus.
//   - Distinguish disabled TLS from unhealthy TLS.
//   - Surface issuer, expiry, and storage metadata for operators.
//
// Scope:
//   - Certificate lifecycle status only.
//
// Usage:
//   - Construct with `NewCertificateCollector(repoRoot)` and call `Collect(ctx)`.
//
// Invariants/Assumptions:
//   - TLS lifecycle is only applicable when the effective gateway URL is HTTPS.
package collectors

import (
	"context"
	"fmt"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/certlifecycle"
	acpconfig "github.com/mitchfultz/ai-control-plane/internal/config"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

var certificateCheck = certlifecycle.Check

var newCertificateStore = func(repoRoot string) certlifecycle.Store {
	return certlifecycle.NewStore(repoRoot)
}

// CertificateCollector checks certificate lifecycle health.
type CertificateCollector struct {
	repoRoot string
}

// NewCertificateCollector creates a certificate collector for the repository runtime.
func NewCertificateCollector(repoRoot string) CertificateCollector {
	return CertificateCollector{repoRoot: repoRoot}
}

// Name returns the collector's domain name.
func (c CertificateCollector) Name() string {
	return "certificate"
}

// Collect gathers certificate lifecycle information.
func (c CertificateCollector) Collect(ctx context.Context) status.ComponentStatus {
	gateway := acpconfig.NewLoader().WithRepoRoot(c.repoRoot).Gateway(true)
	if !gateway.TLSEnabled {
		return componentStatus(c.Name(), status.HealthLevelHealthy, "TLS overlay inactive; certificate lifecycle not applicable", status.ComponentDetails{TLSEnabled: false})
	}

	result, err := certificateCheck(ctx, newCertificateStore(c.repoRoot), certlifecycle.CheckRequest{
		Domain:       gateway.Host,
		BaseURL:      gateway.BaseURL,
		WarningDays:  certlifecycle.DefaultWarningDays,
		CriticalDays: certlifecycle.DefaultCriticalDays,
		Now:          time.Now().UTC(),
	})
	if err != nil {
		return componentStatus(
			c.Name(),
			status.HealthLevelUnhealthy,
			fmt.Sprintf("Certificate inspection failed: %v", err),
			withDetailError(status.ComponentDetails{TLSEnabled: true}, err),
			"Inspect stored certificates: ./scripts/acpctl.sh cert list",
			"Validate live TLS: ./scripts/acpctl.sh cert check",
		)
	}

	primary := result.Certificates[0]
	details := status.ComponentDetails{
		TLSEnabled:        true,
		Domain:            primary.PrimaryName(),
		Domains:           primary.AllNames(),
		Issuer:            primary.Issuer,
		Subject:           primary.Subject,
		SerialNumber:      primary.SerialNumber,
		NotBefore:         primary.NotBefore.Format(time.RFC3339),
		NotAfter:          primary.NotAfter.Format(time.RFC3339),
		DaysRemaining:     primary.DaysRemaining(result.CheckedAt),
		CertificateCount:  len(result.Certificates),
		SelfSigned:        primary.SelfSigned,
		ManagedBy:         primary.ManagedBy,
		StoragePath:       primary.StoragePath,
		FingerprintSHA256: primary.FingerprintSHA256,
		Error:             result.ValidationError,
	}

	switch result.Status {
	case certlifecycle.StatusUnhealthy:
		return componentStatus(c.Name(), status.HealthLevelUnhealthy, result.Message, details, result.Suggestions...)
	case certlifecycle.StatusWarning:
		return componentStatus(c.Name(), status.HealthLevelWarning, result.Message, details, result.Suggestions...)
	default:
		return componentStatus(c.Name(), status.HealthLevelHealthy, result.Message, details)
	}
}
