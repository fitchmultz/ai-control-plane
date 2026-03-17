// Package certlifecycle provides typed Caddy certificate lifecycle workflows.
//
// Purpose:
//   - Execute controlled certificate reissuance with rollback artifacts.
//
// Responsibilities:
//   - Select renewal candidates from the typed certificate check results.
//   - Snapshot certificate storage before mutation.
//   - Remove stale certificate state, restart Caddy, and verify reissuance.
//   - Restore snapshots when verification fails.
//
// Scope:
//   - Certificate renewal orchestration only.
//
// Usage:
//   - Called by `acpctl cert renew` and doctor fix paths.
//
// Invariants/Assumptions:
//   - Renewal artifacts are written under a private artifact-run directory.
//   - Rollback artifacts are preserved even when later steps fail.
package certlifecycle

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/artifactrun"
	repopath "github.com/mitchfultz/ai-control-plane/internal/paths"
)

// Renew performs a controlled threshold-based certificate reissuance.
func Renew(ctx context.Context, store Store, req RenewalRequest) (RenewalResult, error) {
	if store == nil {
		return RenewalResult{}, wrap(ErrorKindRuntime, fmt.Errorf("certificate store is required"))
	}
	now := time.Now().UTC()
	threshold := req.ThresholdDays
	if threshold <= 0 {
		threshold = DefaultRenewThresholdDays
	}

	check, err := Check(ctx, store, CheckRequest{
		Domain:       req.Domain,
		BaseURL:      req.BaseURL,
		WarningDays:  threshold,
		CriticalDays: DefaultCriticalDays,
		Now:          now,
	})
	if err != nil {
		return RenewalResult{}, err
	}

	candidates := append([]CertificateInfo(nil), check.Certificates...)
	if !req.Force {
		filtered := make([]CertificateInfo, 0, len(candidates))
		for _, cert := range candidates {
			if cert.DaysRemaining(now) <= threshold {
				filtered = append(filtered, cert)
			}
		}
		candidates = filtered
	}
	if len(candidates) == 0 {
		return RenewalResult{
			CheckedAt: now,
			Renewed:   false,
			Suggestions: []string{
				"No certificates are within the configured renewal threshold.",
			},
		}, nil
	}

	if req.DryRun {
		items := make([]RenewalItemResult, 0, len(candidates))
		for _, cert := range candidates {
			items = append(items, RenewalItemResult{Domain: cert.PrimaryName(), Before: cert, Renewed: false})
		}
		return RenewalResult{
			CheckedAt: now,
			Renewed:   false,
			Items:     items,
			Suggestions: []string{
				"Dry run only. Re-run without --dry-run to perform controlled reissuance.",
			},
		}, nil
	}

	outputRoot := strings.TrimSpace(req.OutputRoot)
	if outputRoot == "" {
		outputRoot = repopath.DemoLogsPath(req.RepoRoot, "cert-renewals")
	}
	run, err := artifactrun.Create(outputRoot, "cert-renewal", now)
	if err != nil {
		return RenewalResult{}, wrap(ErrorKindRuntime, err)
	}

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	items := make([]RenewalItemResult, 0, len(candidates))
	for _, cert := range candidates {
		backupPath := filepath.Join(run.Directory, sanitizeArtifactName(cert.PrimaryName())+".tar.gz")
		if err := store.Snapshot(ctx, cert, backupPath); err != nil {
			return RenewalResult{}, err
		}
		if err := store.Remove(ctx, cert); err != nil {
			return RenewalResult{}, err
		}
		if err := store.Restart(ctx); err != nil {
			_ = store.Restore(ctx, backupPath)
			return RenewalResult{}, err
		}
		after, err := waitForRenewedCertificate(ctx, store, cert, req.BaseURL, timeout)
		if err != nil {
			_ = store.Restore(ctx, backupPath)
			_ = store.Restart(ctx)
			return RenewalResult{}, wrap(ErrorKindRuntime, fmt.Errorf("verify renewed certificate for %s: %w", cert.PrimaryName(), err))
		}
		items = append(items, RenewalItemResult{
			Domain:     cert.PrimaryName(),
			Before:     cert,
			After:      &after,
			BackupPath: backupPath,
			Renewed:    true,
		})
	}

	summary := RenewalResult{
		RunDirectory: run.Directory,
		CheckedAt:    now,
		Renewed:      true,
		Items:        items,
		Suggestions: []string{
			fmt.Sprintf("Rollback artifacts preserved at %s", run.Directory),
		},
	}
	if err := artifactrun.WriteJSON(filepath.Join(run.Directory, SummaryJSONName), summary); err != nil {
		return RenewalResult{}, wrap(ErrorKindRuntime, err)
	}
	if _, err := artifactrun.Finalize(run.Directory, outputRoot, artifactrun.FinalizeOptions{
		InventoryName:  InventoryName,
		LatestPointers: []string{LatestPointer},
	}); err != nil {
		return RenewalResult{}, wrap(ErrorKindRuntime, err)
	}
	return summary, nil
}

func waitForRenewedCertificate(ctx context.Context, store Store, before CertificateInfo, baseURL string, timeout time.Duration) (CertificateInfo, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		_ = touchGateway(baseURL)
		certs, err := store.List(ctx)
		if err == nil {
			for _, cert := range certs {
				if !cert.MatchesDomain(before.PrimaryName()) {
					continue
				}
				if !strings.EqualFold(cert.FingerprintSHA256, before.FingerprintSHA256) && cert.NotAfter.After(before.NotAfter) {
					return cert, nil
				}
			}
		}
		sleepFor := 5 * time.Second
		if remaining := time.Until(deadline); remaining < sleepFor {
			sleepFor = remaining
		}
		if sleepFor > 0 {
			time.Sleep(sleepFor)
		}
	}
	return CertificateInfo{}, fmt.Errorf("replacement certificate did not appear within %s", timeout)
}

func touchGateway(baseURL string) error {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		return nil
	}
	transport := &http.Transport{}
	if strings.HasPrefix(strings.ToLower(trimmed), "https://") {
		transport.TLSClientConfig = insecureTLSConfig()
	}
	client := &http.Client{Timeout: 10 * time.Second, Transport: transport}
	resp, err := client.Get(strings.TrimRight(trimmed, "/") + "/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func sanitizeArtifactName(value string) string {
	replacer := strings.NewReplacer("*", "wildcard", ".", "_", ":", "_", "/", "_")
	return replacer.Replace(strings.TrimSpace(value))
}
