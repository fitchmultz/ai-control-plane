// Package collectors provides status collectors backed by typed ACP services.
//
// Purpose:
//
//	Expose the latest local database backup signal for operator status and
//	dashboard views.
//
// Responsibilities:
//   - Inspect the canonical local backup directory for the newest backup file.
//   - Surface backup freshness, path, size, and age in one typed status result.
//   - Preserve clear guidance when no backups exist or the latest backup is stale.
//
// Scope:
//   - Local backup artifact status collection only.
//
// Usage:
//   - Construct with NewBackupCollector(repoRoot) and call Collect(ctx).
//
// Invariants/Assumptions:
//   - Canonical backup artifacts are private `.sql.gz` files under demo/backups/.
//   - Freshness thresholds are tuned for the supported daily host backup timer.
package collectors

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/status"
)

const (
	backupWarningAge  = 48 * time.Hour
	backupCriticalAge = 7 * 24 * time.Hour
)

// BackupCollector summarizes the latest local database backup artifact.
type BackupCollector struct {
	repoRoot string
}

// NewBackupCollector creates a backup collector for the repository runtime.
func NewBackupCollector(repoRoot string) BackupCollector {
	return BackupCollector{repoRoot: repoRoot}
}

// Name returns the collector's domain name.
func (c BackupCollector) Name() string {
	return "backup"
}

// Collect gathers backup freshness information.
func (c BackupCollector) Collect(context.Context) status.ComponentStatus {
	backupDir := filepath.Join(c.repoRoot, "demo", "backups")
	info, err := latestBackupFile(backupDir)
	if err != nil {
		return componentStatus(c.Name(), status.HealthLevelWarning, "No database backups found", status.ComponentDetails{},
			"Run make db-backup to create a fresh backup",
			"Keep off-host copies for recovery and host-loss scenarios",
		)
	}

	age := time.Since(info.ModTime()).Round(time.Minute)
	details := status.ComponentDetails{
		BackupPath:      info.path,
		BackupSizeBytes: info.Size(),
		LastBackupUTC:   info.ModTime().UTC().Format(time.RFC3339),
		BackupAge:       age.String(),
	}
	switch {
	case age > backupCriticalAge:
		return componentStatus(c.Name(), status.HealthLevelUnhealthy,
			fmt.Sprintf("Latest backup is stale (%s old)", age),
			details,
			"Run make db-backup immediately",
			"Verify restore posture with make dr-drill",
		)
	case age > backupWarningAge:
		return componentStatus(c.Name(), status.HealthLevelWarning,
			fmt.Sprintf("Latest backup is aging (%s old)", age),
			details,
			"Run make db-backup to refresh the local recovery point",
		)
	default:
		return componentStatus(c.Name(), status.HealthLevelHealthy,
			fmt.Sprintf("Latest backup captured %s ago", age),
			details,
		)
	}
}

type backupFileInfo struct {
	path string
	os.FileInfo
}

func latestBackupFile(backupDir string) (backupFileInfo, error) {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return backupFileInfo{}, err
	}
	backups := make([]backupFileInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql.gz") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return backupFileInfo{}, err
		}
		backups = append(backups, backupFileInfo{
			path:     filepath.Join(backupDir, entry.Name()),
			FileInfo: info,
		})
	}
	if len(backups) == 0 {
		return backupFileInfo{}, os.ErrNotExist
	}
	sort.SliceStable(backups, func(i, j int) bool {
		if backups[i].ModTime().Equal(backups[j].ModTime()) {
			return backups[i].path > backups[j].path
		}
		return backups[i].ModTime().After(backups[j].ModTime())
	})
	return backups[0], nil
}
