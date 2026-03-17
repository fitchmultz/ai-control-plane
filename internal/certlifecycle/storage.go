// Package certlifecycle provides typed Caddy certificate lifecycle workflows.
//
// Purpose:
//   - Implement certificate storage operations against the supported Caddy
//   - host-first runtime.
//
// Responsibilities:
//   - Discover the running Caddy container through the canonical ACP compose scope.
//   - List stored certificates from the Caddy data volume.
//   - Snapshot, remove, restore, and restart certificate storage for renewal.
//
// Scope:
//   - Caddy container and storage interaction only.
//
// Usage:
//   - Construct with `NewStore(repoRoot)` and pass to `Check` or `Renew`.
//
// Invariants/Assumptions:
//   - Certificates live under `/data/caddy/certificates` inside the Caddy container.
//   - The supported renewal path preserves per-certificate parent directories as rollback artifacts.
package certlifecycle

import (
	"context"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/docker"
	"github.com/mitchfultz/ai-control-plane/internal/fsutil"
	"github.com/mitchfultz/ai-control-plane/internal/proc"
)

const certRestartTimeout = 60 * time.Second

// DockerStore implements certificate lifecycle storage against the running Caddy container.
type DockerStore struct {
	repoRoot string
}

// NewStore constructs the supported Caddy-backed certificate store.
func NewStore(repoRoot string) *DockerStore {
	return &DockerStore{repoRoot: strings.TrimSpace(repoRoot)}
}

// List returns parsed certificate metadata from the Caddy data store.
func (s *DockerStore) List(ctx context.Context) ([]CertificateInfo, error) {
	containerID, err := s.caddyContainerID(ctx)
	if err != nil {
		return nil, err
	}
	output, err := docker.ExecInContainer(ctx, containerID, "sh", "-lc", `find /data/caddy/certificates -type f \( -name '*.crt' -o -name '*.pem' \) | sort`)
	if err != nil {
		return nil, wrap(ErrorKindPrereq, fmt.Errorf("list Caddy certificates: %w", err))
	}

	paths := filterNonEmpty(strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n"))
	certs := make([]CertificateInfo, 0, len(paths))
	seen := make(map[string]struct{}, len(paths))
	for _, certPath := range paths {
		cert, err := s.readCertificate(ctx, containerID, certPath)
		if err != nil {
			return nil, err
		}
		key := cert.FingerprintSHA256 + "|" + cert.StoragePath
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		certs = append(certs, cert)
	}
	sort.Slice(certs, func(i, j int) bool {
		left := strings.ToLower(certs[i].PrimaryName())
		right := strings.ToLower(certs[j].PrimaryName())
		if left == right {
			return certs[i].NotAfter.Before(certs[j].NotAfter)
		}
		return left < right
	})
	return certs, nil
}

// Snapshot writes a base64-decoded tarball backup of the certificate parent directory.
func (s *DockerStore) Snapshot(ctx context.Context, cert CertificateInfo, backupPath string) error {
	containerID, err := s.caddyContainerID(ctx)
	if err != nil {
		return err
	}
	relParent, err := certificateRelativeParent(cert)
	if err != nil {
		return err
	}
	archive64, err := docker.ExecInContainer(ctx, containerID, "sh", "-lc", fmt.Sprintf(`tar czf - -C /data %q | base64 | tr -d '\n'`, relParent))
	if err != nil {
		return wrap(ErrorKindRuntime, fmt.Errorf("snapshot certificate %s: %w", cert.PrimaryName(), err))
	}
	archive, err := base64.StdEncoding.DecodeString(strings.TrimSpace(archive64))
	if err != nil {
		return wrap(ErrorKindRuntime, fmt.Errorf("decode certificate snapshot %s: %w", cert.PrimaryName(), err))
	}
	if err := fsutil.EnsurePrivateDir(filepath.Dir(backupPath)); err != nil {
		return wrap(ErrorKindRuntime, fmt.Errorf("create certificate snapshot directory: %w", err))
	}
	if err := fsutil.AtomicWritePrivateFile(backupPath, archive); err != nil {
		return wrap(ErrorKindRuntime, fmt.Errorf("write certificate snapshot %s: %w", backupPath, err))
	}
	return nil
}

// Remove deletes the stored certificate parent directory so Caddy can reissue it.
func (s *DockerStore) Remove(ctx context.Context, cert CertificateInfo) error {
	containerID, err := s.caddyContainerID(ctx)
	if err != nil {
		return err
	}
	relParent, err := certificateRelativeParent(cert)
	if err != nil {
		return err
	}
	if _, err := docker.ExecInContainer(ctx, containerID, "sh", "-lc", fmt.Sprintf(`rm -rf %q`, path.Join("/data", relParent))); err != nil {
		return wrap(ErrorKindRuntime, fmt.Errorf("remove certificate %s: %w", cert.PrimaryName(), err))
	}
	return nil
}

// Restore replays a saved tarball into the Caddy data volume.
func (s *DockerStore) Restore(ctx context.Context, backupPath string) error {
	containerID, err := s.caddyContainerID(ctx)
	if err != nil {
		return err
	}
	archive, err := os.ReadFile(backupPath)
	if err != nil {
		return wrap(ErrorKindRuntime, fmt.Errorf("read certificate snapshot %s: %w", backupPath, err))
	}
	request := proc.Request{
		Name:    "docker",
		Args:    []string{"exec", "-i", containerID, "sh", "-lc", "base64 -d | tar xzf - -C /data"},
		Stdin:   strings.NewReader(base64.StdEncoding.EncodeToString(archive)),
		Timeout: certRestartTimeout,
	}
	result := proc.Run(ctx, request)
	if result.Err != nil {
		return wrap(ErrorKindRuntime, fmt.Errorf("restore certificate snapshot %s: %w", backupPath, result.Err))
	}
	return nil
}

// Restart restarts the running Caddy container.
func (s *DockerStore) Restart(ctx context.Context) error {
	containerID, err := s.caddyContainerID(ctx)
	if err != nil {
		return err
	}
	res := proc.Run(ctx, proc.Request{
		Name:    "docker",
		Args:    []string{"restart", containerID},
		Timeout: certRestartTimeout,
	})
	if res.Err != nil {
		return wrap(ErrorKindRuntime, fmt.Errorf("restart Caddy container: %w", res.Err))
	}
	return nil
}

func (s *DockerStore) caddyContainerID(ctx context.Context) (string, error) {
	compose, err := docker.NewACPCompose(s.repoRoot, []string{"docker-compose.yml", "docker-compose.tls.yml"})
	if err != nil {
		return "", wrap(ErrorKindPrereq, fmt.Errorf("resolve Docker Compose for certificate lifecycle: %w", err))
	}
	containerID, err := compose.ContainerID(ctx, "caddy")
	if err != nil {
		return "", wrap(ErrorKindPrereq, fmt.Errorf("caddy service not available; enable the tls overlay before running certificate workflows: %w", err))
	}
	return strings.TrimSpace(containerID), nil
}

func (s *DockerStore) readCertificate(ctx context.Context, containerID string, certPath string) (CertificateInfo, error) {
	encoded, err := docker.ExecInContainer(ctx, containerID, "sh", "-lc", fmt.Sprintf(`base64 %q | tr -d '\n'`, certPath))
	if err != nil {
		return CertificateInfo{}, wrap(ErrorKindRuntime, fmt.Errorf("read certificate %s: %w", certPath, err))
	}
	payload, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return CertificateInfo{}, wrap(ErrorKindRuntime, fmt.Errorf("decode certificate %s: %w", certPath, err))
	}
	block, _ := pem.Decode(payload)
	for block != nil && block.Type != "CERTIFICATE" {
		block, payload = pem.Decode(payload)
	}
	if block == nil {
		return CertificateInfo{}, wrap(ErrorKindRuntime, fmt.Errorf("parse certificate %s: PEM certificate block not found", certPath))
	}
	parsed, err := x509ParseCertificate(block.Bytes)
	if err != nil {
		return CertificateInfo{}, wrap(ErrorKindRuntime, fmt.Errorf("parse certificate %s: %w", certPath, err))
	}
	storagePath := path.Clean(certPath)
	storageParentPath := path.Dir(storagePath)
	return certificateInfoFromX509(parsed, managedByFromPath(storagePath), storagePath, storageParentPath), nil
}

func certificateRelativeParent(cert CertificateInfo) (string, error) {
	parent := strings.TrimSpace(cert.StorageParentPath)
	if parent == "" {
		return "", wrap(ErrorKindDomain, fmt.Errorf("certificate %s has no storage parent path", cert.PrimaryName()))
	}
	cleaned := path.Clean(parent)
	if !strings.HasPrefix(cleaned, "/data/") {
		return "", wrap(ErrorKindDomain, fmt.Errorf("certificate %s storage path %s is outside /data", cert.PrimaryName(), cleaned))
	}
	return strings.TrimPrefix(cleaned, "/data/"), nil
}

func managedByFromPath(certPath string) string {
	lower := strings.ToLower(certPath)
	switch {
	case strings.Contains(lower, "/local/"):
		return "caddy-internal-ca"
	case strings.Contains(lower, "acme"):
		return "lets-encrypt"
	default:
		return "caddy"
	}
}

func filterNonEmpty(values []string) []string {
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			filtered = append(filtered, trimmed)
		}
	}
	return filtered
}
