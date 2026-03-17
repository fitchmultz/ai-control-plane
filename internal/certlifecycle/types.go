// Package certlifecycle provides typed Caddy certificate lifecycle workflows.
//
// Purpose:
//   - Define the shared typed models for certificate inspection, validation,
//   - renewal, and automation.
//
// Responsibilities:
//   - Describe certificate metadata captured from storage and live TLS.
//   - Describe certificate health and renewal request/response payloads.
//   - Publish the storage contract used by the renewal orchestration.
//
// Scope:
//   - Shared certificate lifecycle models only.
//
// Usage:
//   - Consumed by `cmd/acpctl`, status collectors, doctor checks, and tests.
//
// Invariants/Assumptions:
//   - The supported path is Caddy-managed certificates on the host-first TLS overlay.
//   - Renewal artifacts remain private local files.
package certlifecycle

import (
	"context"
	"crypto/x509"
	"net"
	"strings"
	"time"
)

const (
	DefaultWarningDays        = 30
	DefaultCriticalDays       = 7
	DefaultRenewThresholdDays = 30
	SummaryJSONName           = "summary.json"
	InventoryName             = "inventory.txt"
	LatestPointer             = "latest"
)

// Status captures the aggregated certificate health state.
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusWarning   Status = "warning"
	StatusUnhealthy Status = "unhealthy"
)

// CertificateInfo captures the parsed metadata for one stored or live certificate.
type CertificateInfo struct {
	Subject           string    `json:"subject"`
	Issuer            string    `json:"issuer"`
	SerialNumber      string    `json:"serial_number"`
	DNSNames          []string  `json:"dns_names,omitempty"`
	IPAddresses       []string  `json:"ip_addresses,omitempty"`
	NotBefore         time.Time `json:"not_before"`
	NotAfter          time.Time `json:"not_after"`
	FingerprintSHA256 string    `json:"fingerprint_sha256"`
	SelfSigned        bool      `json:"self_signed,omitempty"`
	ManagedBy         string    `json:"managed_by,omitempty"`
	StoragePath       string    `json:"storage_path,omitempty"`
	StorageParentPath string    `json:"-"`
}

// AllNames returns the union of DNS and IP SANs.
func (c CertificateInfo) AllNames() []string {
	values := make([]string, 0, len(c.DNSNames)+len(c.IPAddresses))
	values = append(values, c.DNSNames...)
	values = append(values, c.IPAddresses...)
	return values
}

// PrimaryName returns the most operator-friendly name for the certificate.
func (c CertificateInfo) PrimaryName() string {
	if len(c.DNSNames) > 0 && strings.TrimSpace(c.DNSNames[0]) != "" {
		return c.DNSNames[0]
	}
	if len(c.IPAddresses) > 0 && strings.TrimSpace(c.IPAddresses[0]) != "" {
		return c.IPAddresses[0]
	}
	return strings.TrimSpace(c.Subject)
}

// MatchesDomain reports whether the certificate covers the requested host or IP.
func (c CertificateInfo) MatchesDomain(raw string) bool {
	host := strings.TrimSpace(raw)
	if host == "" {
		return true
	}
	if parsed := net.ParseIP(host); parsed != nil {
		for _, ip := range c.IPAddresses {
			if parsed.Equal(net.ParseIP(strings.TrimSpace(ip))) {
				return true
			}
		}
		return false
	}
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	for _, name := range c.DNSNames {
		trimmed := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(name)), ".")
		if trimmed == host {
			return true
		}
		if strings.HasPrefix(trimmed, "*.") {
			suffix := strings.TrimPrefix(trimmed, "*")
			if strings.HasSuffix(host, suffix) && host != strings.TrimPrefix(suffix, ".") {
				return true
			}
		}
	}
	return false
}

// DaysRemaining returns the integer number of days until expiry.
func (c CertificateInfo) DaysRemaining(now time.Time) int {
	return int(c.NotAfter.Sub(now.UTC()).Hours() / 24)
}

// CheckRequest configures certificate validation.
type CheckRequest struct {
	Domain       string
	BaseURL      string
	WarningDays  int
	CriticalDays int
	Now          time.Time
}

// CheckResult captures the current certificate posture.
type CheckResult struct {
	CheckedAt       time.Time         `json:"checked_at"`
	Status          Status            `json:"status"`
	Message         string            `json:"message"`
	Certificates    []CertificateInfo `json:"certificates,omitempty"`
	LiveCertificate *CertificateInfo  `json:"live_certificate,omitempty"`
	ValidationError string            `json:"validation_error,omitempty"`
	Suggestions     []string          `json:"suggestions,omitempty"`
}

// RenewalRequest configures controlled certificate reissuance.
type RenewalRequest struct {
	RepoRoot      string
	Domain        string
	BaseURL       string
	ThresholdDays int
	Force         bool
	DryRun        bool
	OutputRoot    string
	Timeout       time.Duration
}

// RenewalItemResult captures one renewed certificate outcome.
type RenewalItemResult struct {
	Domain     string           `json:"domain"`
	Before     CertificateInfo  `json:"before"`
	After      *CertificateInfo `json:"after,omitempty"`
	BackupPath string           `json:"backup_path,omitempty"`
	Renewed    bool             `json:"renewed"`
}

// RenewalResult captures one renewal run summary.
type RenewalResult struct {
	RunDirectory string              `json:"run_directory,omitempty"`
	CheckedAt    time.Time           `json:"checked_at"`
	Renewed      bool                `json:"renewed"`
	Items        []RenewalItemResult `json:"items,omitempty"`
	Suggestions  []string            `json:"suggestions,omitempty"`
}

// Store captures the certificate storage operations used by validation and renewal.
type Store interface {
	List(ctx context.Context) ([]CertificateInfo, error)
	Snapshot(ctx context.Context, cert CertificateInfo, backupPath string) error
	Remove(ctx context.Context, cert CertificateInfo) error
	Restore(ctx context.Context, backupPath string) error
	Restart(ctx context.Context) error
}

func certificateInfoFromX509(cert *x509.Certificate, managedBy string, storagePath string, storageParentPath string) CertificateInfo {
	issuer := strings.TrimSpace(cert.Issuer.CommonName)
	if issuer == "" {
		issuer = strings.TrimSpace(cert.Issuer.String())
	}
	subject := strings.TrimSpace(cert.Subject.CommonName)
	if subject == "" {
		subject = strings.TrimSpace(cert.Subject.String())
	}
	ips := make([]string, 0, len(cert.IPAddresses))
	for _, ip := range cert.IPAddresses {
		ips = append(ips, ip.String())
	}
	return CertificateInfo{
		Subject:           subject,
		Issuer:            issuer,
		SerialNumber:      strings.ToUpper(cert.SerialNumber.Text(16)),
		DNSNames:          append([]string(nil), cert.DNSNames...),
		IPAddresses:       ips,
		NotBefore:         cert.NotBefore.UTC(),
		NotAfter:          cert.NotAfter.UTC(),
		FingerprintSHA256: fingerprintSHA256(cert.Raw),
		SelfSigned:        cert.Issuer.String() == cert.Subject.String(),
		ManagedBy:         managedBy,
		StoragePath:       storagePath,
		StorageParentPath: storageParentPath,
	}
}
