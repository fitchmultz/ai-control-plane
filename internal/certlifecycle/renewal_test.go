// renewal_test.go - Tests for certificate renewal orchestration.
//
// Purpose:
//   - Verify threshold selection, forced renewal, and rollback behavior.
//
// Responsibilities:
//   - Ensure snapshots occur before storage removal.
//   - Ensure restore is attempted when verification fails.
//   - Ensure dry-run does not mutate storage.
//
// Scope:
//   - Renewal workflow tests only.
//
// Usage:
//   - Run via `go test ./internal/certlifecycle`.
//
// Invariants/Assumptions:
//   - Tests use a stubbed certificate store and never touch Docker or live TLS.
//   - Renewal verification is satisfied by synthetic post-renewal certificate state.
package certlifecycle

import (
	"context"
	"testing"
	"time"
)

type fakeStore struct {
	list        []CertificateInfo
	snapshots   []string
	removed     []string
	restored    []string
	restarts    int
	next        *CertificateInfo
	listCalls   int
	snapshotErr error
	removeErr   error
	restoreErr  error
	restartErr  error
}

func (f *fakeStore) List(context.Context) ([]CertificateInfo, error) {
	f.listCalls++
	if f.next != nil && f.listCalls > 1 {
		return []CertificateInfo{*f.next}, nil
	}
	return append([]CertificateInfo(nil), f.list...), nil
}

func (f *fakeStore) Snapshot(_ context.Context, cert CertificateInfo, path string) error {
	f.snapshots = append(f.snapshots, path)
	return f.snapshotErr
}

func (f *fakeStore) Remove(_ context.Context, cert CertificateInfo) error {
	f.removed = append(f.removed, cert.PrimaryName())
	return f.removeErr
}

func (f *fakeStore) Restore(_ context.Context, path string) error {
	f.restored = append(f.restored, path)
	return f.restoreErr
}

func (f *fakeStore) Restart(context.Context) error {
	f.restarts++
	return f.restartErr
}

func TestRenewDryRunDoesNotMutateStorage(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Add(10 * 24 * time.Hour)
	store := &fakeStore{list: []CertificateInfo{{DNSNames: []string{"gateway.example.com"}, NotAfter: now, FingerprintSHA256: "OLD"}}}

	result, err := Renew(context.Background(), store, RenewalRequest{RepoRoot: t.TempDir(), Domain: "gateway.example.com", ThresholdDays: 30, Force: true, DryRun: true})
	if err != nil {
		t.Fatalf("Renew returned error: %v", err)
	}
	if result.Renewed {
		t.Fatalf("expected dry-run not to renew")
	}
	if len(store.snapshots) != 0 || len(store.removed) != 0 || store.restarts != 0 {
		t.Fatalf("expected no mutations during dry-run")
	}
}

func TestRenewRestoresSnapshotOnVerificationFailure(t *testing.T) {
	t.Parallel()

	store := &fakeStore{list: []CertificateInfo{{DNSNames: []string{"gateway.example.com"}, NotAfter: time.Now().UTC().Add(2 * 24 * time.Hour), FingerprintSHA256: "OLD"}}}

	_, err := Renew(context.Background(), store, RenewalRequest{RepoRoot: t.TempDir(), Domain: "gateway.example.com", ThresholdDays: 30, Force: true, Timeout: 10 * time.Millisecond})
	if err == nil {
		t.Fatalf("expected verification failure")
	}
	if len(store.snapshots) != 1 {
		t.Fatalf("expected snapshot before mutation")
	}
	if len(store.removed) != 1 {
		t.Fatalf("expected certificate removal")
	}
	if len(store.restored) != 1 {
		t.Fatalf("expected rollback restore")
	}
}

func TestRenewUsesNewFingerprintWhenReplacementAppears(t *testing.T) {
	t.Parallel()

	after := CertificateInfo{DNSNames: []string{"gateway.example.com"}, NotAfter: time.Now().UTC().Add(90 * 24 * time.Hour), FingerprintSHA256: "NEW"}
	store := &fakeStore{
		list: []CertificateInfo{{DNSNames: []string{"gateway.example.com"}, NotAfter: time.Now().UTC().Add(2 * 24 * time.Hour), FingerprintSHA256: "OLD"}},
		next: &after,
	}

	result, err := Renew(context.Background(), store, RenewalRequest{RepoRoot: t.TempDir(), Domain: "gateway.example.com", ThresholdDays: 30, Force: true, BaseURL: "", Timeout: 100 * time.Millisecond})
	if err != nil {
		t.Fatalf("Renew returned error: %v", err)
	}
	if !result.Renewed || len(result.Items) != 1 || result.Items[0].After == nil || result.Items[0].After.FingerprintSHA256 != "NEW" {
		t.Fatalf("unexpected renewal result: %+v", result)
	}
}
