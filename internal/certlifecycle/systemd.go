// Package certlifecycle provides typed Caddy certificate lifecycle workflows.
//
// Purpose:
//   - Install and enable the supported host-first certificate renewal timer.
//
// Responsibilities:
//   - Render tracked certificate-renewal systemd unit templates.
//   - Persist them to `/etc/systemd/system` with stable permissions.
//   - Reload systemd and enable/start the renewal timer.
//
// Scope:
//   - Local systemd installation only.
//
// Usage:
//   - Used by `acpctl cert renew-auto`.
//
// Invariants/Assumptions:
//   - Must run with sufficient permissions to write systemd units.
//   - Uses the tracked template files under `deploy/systemd/`.
package certlifecycle

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/mitchfultz/ai-control-plane/internal/fsutil"
	"github.com/mitchfultz/ai-control-plane/internal/proc"
)

// InstallTimerOptions configures certificate renewal timer installation.
type InstallTimerOptions struct {
	RepoRoot        string
	WorkingDir      string
	EnvFile         string
	ServiceUser     string
	ServiceGroup    string
	OnCalendar      string
	RandomizedDelay string
	ThresholdDays   int
}

// InstallTimerResult captures the installed systemd paths.
type InstallTimerResult struct {
	ServicePath string `json:"service_path"`
	TimerPath   string `json:"timer_path"`
	TimerName   string `json:"timer_name"`
}

// InstallAutoRenewTimer renders and enables the certificate renewal timer.
func InstallAutoRenewTimer(ctx context.Context, opts InstallTimerOptions) (InstallTimerResult, error) {
	if os.Geteuid() != 0 {
		return InstallTimerResult{}, wrap(ErrorKindDomain, fmt.Errorf("cert renew-auto must run as root"))
	}
	workingDir := strings.TrimSpace(opts.WorkingDir)
	if workingDir == "" {
		workingDir = strings.TrimSpace(opts.RepoRoot)
	}
	if workingDir == "" {
		return InstallTimerResult{}, wrap(ErrorKindDomain, fmt.Errorf("repository root is required"))
	}
	serviceUser := strings.TrimSpace(opts.ServiceUser)
	if serviceUser == "" {
		serviceUser = "root"
	}
	serviceGroup := strings.TrimSpace(opts.ServiceGroup)
	if serviceGroup == "" {
		serviceGroup = serviceUser
	}
	onCalendar := strings.TrimSpace(opts.OnCalendar)
	if onCalendar == "" {
		onCalendar = "daily"
	}
	randomizedDelay := strings.TrimSpace(opts.RandomizedDelay)
	if randomizedDelay == "" {
		randomizedDelay = "30m"
	}
	thresholdDays := opts.ThresholdDays
	if thresholdDays <= 0 {
		thresholdDays = DefaultRenewThresholdDays
	}

	serviceBody, err := renderTemplate(filepath.Join(opts.RepoRoot, "deploy", "systemd", "ai-control-plane-cert-renewal.service.tmpl"), map[string]any{
		"SERVICE_NAME":         "ai-control-plane",
		"SERVICE_USER":         serviceUser,
		"SERVICE_GROUP":        serviceGroup,
		"WORKING_DIR":          workingDir,
		"ENV_FILE":             opts.EnvFile,
		"RENEW_THRESHOLD_DAYS": thresholdDays,
	})
	if err != nil {
		return InstallTimerResult{}, wrap(ErrorKindRuntime, err)
	}
	timerBody, err := renderTemplate(filepath.Join(opts.RepoRoot, "deploy", "systemd", "ai-control-plane-cert-renewal.timer.tmpl"), map[string]any{
		"SERVICE_NAME":         "ai-control-plane",
		"RENEW_ON_CALENDAR":    onCalendar,
		"RANDOMIZED_DELAY_SEC": randomizedDelay,
	})
	if err != nil {
		return InstallTimerResult{}, wrap(ErrorKindRuntime, err)
	}

	servicePath := "/etc/systemd/system/ai-control-plane-cert-renewal.service"
	timerPath := "/etc/systemd/system/ai-control-plane-cert-renewal.timer"
	if err := fsutil.AtomicWriteFile(servicePath, []byte(serviceBody), fsutil.PublicFilePerm); err != nil {
		return InstallTimerResult{}, wrap(ErrorKindRuntime, fmt.Errorf("write %s: %w", servicePath, err))
	}
	if err := fsutil.AtomicWriteFile(timerPath, []byte(timerBody), fsutil.PublicFilePerm); err != nil {
		return InstallTimerResult{}, wrap(ErrorKindRuntime, fmt.Errorf("write %s: %w", timerPath, err))
	}

	for _, args := range [][]string{{"daemon-reload"}, {"enable", "--now", "ai-control-plane-cert-renewal.timer"}} {
		res := proc.Run(ctx, proc.Request{Name: "systemctl", Args: args, Timeout: 30 * time.Second})
		if res.Err != nil {
			return InstallTimerResult{}, wrap(ErrorKindRuntime, fmt.Errorf("systemctl %s: %w", strings.Join(args, " "), res.Err))
		}
	}
	return InstallTimerResult{ServicePath: servicePath, TimerPath: timerPath, TimerName: "ai-control-plane-cert-renewal.timer"}, nil
}

func renderTemplate(path string, data map[string]any) (string, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read template %s: %w", path, err)
	}
	tmpl, err := template.New(filepath.Base(path)).Option("missingkey=error").Parse(string(body))
	if err != nil {
		return "", fmt.Errorf("parse template %s: %w", path, err)
	}
	var builder strings.Builder
	if err := tmpl.Execute(&builder, data); err != nil {
		return "", fmt.Errorf("render template %s: %w", path, err)
	}
	return builder.String(), nil
}
