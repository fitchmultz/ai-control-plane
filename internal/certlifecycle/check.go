// Package certlifecycle provides typed Caddy certificate lifecycle workflows.
//
// Purpose:
//   - Validate stored and live TLS certificate state for the supported host-first path.
//
// Responsibilities:
//   - Inspect stored Caddy-managed certificates.
//   - Validate expiry thresholds.
//   - Compare the live served certificate with stored metadata when HTTPS is enabled.
//
// Scope:
//   - Certificate inspection and health evaluation only.
//
// Usage:
//   - Called by `acpctl cert check`, status collectors, and doctor checks.
//
// Invariants/Assumptions:
//   - HTTPS live validation is only attempted when a TLS base URL is supplied.
//   - Warning and critical thresholds are expressed in whole days.
package certlifecycle

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"sort"
	"strings"
	"time"
)

// Check returns the current certificate posture for the requested host.
func Check(ctx context.Context, store Store, req CheckRequest) (CheckResult, error) {
	if store == nil {
		return CheckResult{}, wrap(ErrorKindRuntime, fmt.Errorf("certificate store is required"))
	}
	now := req.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	warningDays := req.WarningDays
	if warningDays <= 0 {
		warningDays = DefaultWarningDays
	}
	criticalDays := req.CriticalDays
	if criticalDays <= 0 {
		criticalDays = DefaultCriticalDays
	}

	certs, err := store.List(ctx)
	if err != nil {
		return CheckResult{}, err
	}
	selected := filterCertificatesByDomain(certs, req.Domain)
	if len(selected) == 0 {
		return CheckResult{}, wrap(ErrorKindDomain, fmt.Errorf("no stored certificates found for %q", strings.TrimSpace(req.Domain)))
	}
	sortCertificates(selected)

	result := CheckResult{
		CheckedAt:    now,
		Status:       StatusHealthy,
		Certificates: selected,
		Message:      fmt.Sprintf("Certificate valid for %d day(s)", selected[0].DaysRemaining(now)),
	}

	minDays := selected[0].DaysRemaining(now)
	if minDays <= criticalDays {
		result.Status = StatusUnhealthy
		result.Message = fmt.Sprintf("Certificate expires in %d day(s)", minDays)
		result.Suggestions = append(result.Suggestions,
			fmt.Sprintf("Renew immediately: ./scripts/acpctl.sh cert renew --threshold-days %d", warningDays),
			"Confirm Caddy can complete ACME or internal-CA issuance for the configured hostname",
		)
	} else if minDays <= warningDays {
		result.Status = StatusWarning
		result.Message = fmt.Sprintf("Certificate expires in %d day(s)", minDays)
		result.Suggestions = append(result.Suggestions,
			fmt.Sprintf("Renew soon: ./scripts/acpctl.sh cert renew --threshold-days %d", warningDays),
		)
	}

	live, liveErr := fetchLiveCertificate(req.BaseURL, req.Domain)
	if liveErr != nil {
		result.Status = StatusUnhealthy
		result.ValidationError = liveErr.Error()
		result.Message = fmt.Sprintf("Live TLS validation failed: %v", liveErr)
		result.Suggestions = append(result.Suggestions,
			"Verify the tls overlay is active and Caddy is serving the configured hostname",
			"Inspect stored certificates: ./scripts/acpctl.sh cert list",
		)
		return result, nil
	}
	if live != nil {
		result.LiveCertificate = live
		if !matchesStoredFingerprint(selected, live.FingerprintSHA256) {
			result.Status = StatusUnhealthy
			result.ValidationError = "live served certificate does not match stored Caddy certificate"
			result.Message = "Live served certificate does not match stored Caddy certificate"
			result.Suggestions = append(result.Suggestions,
				"Verify Caddy is serving the expected site and storage volume",
				"Run controlled renewal: ./scripts/acpctl.sh cert renew",
			)
			return result, nil
		}
	}

	return result, nil
}

func filterCertificatesByDomain(certs []CertificateInfo, domain string) []CertificateInfo {
	if strings.TrimSpace(domain) == "" {
		return append([]CertificateInfo(nil), certs...)
	}
	filtered := make([]CertificateInfo, 0, len(certs))
	for _, cert := range certs {
		if cert.MatchesDomain(domain) {
			filtered = append(filtered, cert)
		}
	}
	return filtered
}

func sortCertificates(certs []CertificateInfo) {
	sort.Slice(certs, func(i, j int) bool {
		if certs[i].NotAfter.Equal(certs[j].NotAfter) {
			return strings.ToLower(certs[i].PrimaryName()) < strings.ToLower(certs[j].PrimaryName())
		}
		return certs[i].NotAfter.Before(certs[j].NotAfter)
	})
}

func matchesStoredFingerprint(certs []CertificateInfo, fingerprint string) bool {
	for _, cert := range certs {
		if strings.EqualFold(cert.FingerprintSHA256, fingerprint) {
			return true
		}
	}
	return false
}

func fetchLiveCertificate(rawBaseURL string, domain string) (*CertificateInfo, error) {
	trimmed := strings.TrimSpace(rawBaseURL)
	if trimmed == "" {
		return nil, nil
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("parse base URL: %w", err)
	}
	if !strings.EqualFold(parsed.Scheme, "https") {
		return nil, nil
	}
	host := parsed.Hostname()
	if host == "" {
		return nil, fmt.Errorf("base URL host is empty")
	}
	port := parsed.Port()
	if strings.TrimSpace(port) == "" {
		port = "443"
	}
	serverName := strings.TrimSpace(domain)
	if serverName == "" {
		serverName = host
	}
	if net.ParseIP(serverName) != nil {
		serverName = ""
	}

	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", net.JoinHostPort(host, port), &tls.Config{
		InsecureSkipVerify: true, //nolint:gosec // certificate lifecycle intentionally inspects presented certs regardless of trust.
		ServerName:         serverName,
	})
	if err != nil {
		return nil, fmt.Errorf("dial live TLS endpoint %s: %w", net.JoinHostPort(host, port), err)
	}
	defer conn.Close()
	state := conn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return nil, fmt.Errorf("live TLS endpoint returned no peer certificates")
	}
	info := certificateInfoFromX509(state.PeerCertificates[0], "live-tls", "", "")
	return &info, nil
}

func x509ParseCertificate(raw []byte) (*x509.Certificate, error) {
	return x509.ParseCertificate(raw)
}

func fingerprintSHA256(raw []byte) string {
	digest := sha256.Sum256(raw)
	return strings.ToUpper(hex.EncodeToString(digest[:]))
}

func insecureTLSConfig() *tls.Config {
	return &tls.Config{InsecureSkipVerify: true} //nolint:gosec // certificate lifecycle intentionally inspects presented certs regardless of trust.
}
