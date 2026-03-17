// cmd_host_preflight.go - Native host preflight command implementation.
//
// Purpose:
//   - Own the typed host-first preflight gate without routing through an
//     internal bridge script.
//
// Responsibilities:
//   - Validate the supported host boundary for production host deployments.
//   - Verify tracked host deployment assets exist locally.
//   - Run the canonical production deployment-contract validation.
//
// Scope:
//   - Local host preflight checks only.
//
// Usage:
//   - Invoked through `acpctl host preflight`.
//
// Invariants/Assumptions:
//   - Only the production profile is supported.
//   - The supported host boundary is Debian 12+ or Ubuntu 24.04+ with systemd,
//     apt, Docker, and Docker Compose available.
//   - Relative paths resolve from the repository root.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/docker"
	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
	"github.com/mitchfultz/ai-control-plane/internal/output"
	"github.com/mitchfultz/ai-control-plane/internal/prereq"
	"github.com/mitchfultz/ai-control-plane/internal/validation"
)

const defaultHostSecretsEnvFile = "/etc/ai-control-plane/secrets.env"

var (
	hostRuntimeGOOS       = runtime.GOOS
	hostOSReleasePath     = "/etc/os-release"
	hostSystemdRuntimeDir = "/run/systemd/system"
)

type hostPreflightOptions struct {
	Profile        string
	SecretsEnvFile string
}

type hostOSRelease struct {
	ID        string
	VersionID string
}

func hostPreflightCommandSpec() *commandSpec {
	return newNativeCommandSpec(nativeCommandConfig{
		Name:        "preflight",
		Summary:     "Validate host readiness",
		Description: "Validate the local host-first runtime contract before installation or startup.",
		Examples: []string{
			"acpctl host preflight",
			"acpctl host preflight --secrets-env-file /etc/ai-control-plane/secrets.env",
		},
		Options: []commandOptionSpec{
			{Name: "profile", ValueName: "NAME", Summary: "Deployment profile (only production is supported)", Type: optionValueString, DefaultText: "production"},
			{Name: "secrets-env-file", ValueName: "PATH", Summary: "Canonical production secrets file", Type: optionValueString, DefaultText: defaultHostSecretsEnvFile},
		},
		Bind: bindRepoParsed(bindHostPreflightOptions),
		Run:  runHostPreflight,
	})
}

func bindHostPreflightOptions(bindCtx commandBindContext, input parsedCommandInput) (hostPreflightOptions, error) {
	repoRoot, err := requireCommandRepoRoot(bindCtx)
	if err != nil {
		return hostPreflightOptions{}, err
	}
	profile := input.StringDefault("profile", "production")
	if profile != "production" {
		return hostPreflightOptions{}, fmt.Errorf("unsupported profile: %s", profile)
	}
	return hostPreflightOptions{
		Profile:        profile,
		SecretsEnvFile: resolveRepoInput(repoRoot, input.StringDefault("secrets-env-file", defaultHostSecretsEnvFile)),
	}, nil
}

func runHostPreflight(_ context.Context, runCtx commandRunContext, raw any) int {
	opts := raw.(hostPreflightOptions)
	out := output.New()
	logger := workflowLogger(runCtx, "host_preflight", "profile", opts.Profile)
	workflowStart(logger)
	printCommandSection(runCtx.Stdout, out, "=== Host Preflight ===")
	fmt.Fprintf(runCtx.Stdout, "Profile: %s\n", opts.Profile)
	fmt.Fprintf(runCtx.Stdout, "Secrets file: %s\n", opts.SecretsEnvFile)

	if err := requireHostPreflightPrereqs(); err != nil {
		workflowFailure(logger, err)
		return failCommand(runCtx.Stderr, out, exitcodes.ACPExitPrereq, err, "Host preflight prerequisites failed")
	}

	requiredTemplates := []string{
		filepath.Join(runCtx.RepoRoot, "deploy", "systemd", "ai-control-plane.service.tmpl"),
		filepath.Join(runCtx.RepoRoot, "deploy", "systemd", "ai-control-plane-backup.service.tmpl"),
		filepath.Join(runCtx.RepoRoot, "deploy", "systemd", "ai-control-plane-backup.timer.tmpl"),
	}
	for _, templatePath := range requiredTemplates {
		if _, err := os.Stat(templatePath); err != nil {
			workflowFailure(logger, err, "template", templatePath)
			return failCommand(runCtx.Stderr, out, exitcodes.ACPExitRuntime, err, "Missing systemd template")
		}
	}

	issues, err := validation.ValidateDeploymentConfig(runCtx.RepoRoot, validation.ConfigValidationOptions{
		Profile:        validation.ConfigValidationProfileProduction,
		SecretsEnvFile: opts.SecretsEnvFile,
	})
	if err != nil {
		workflowFailure(logger, err)
		return failCommand(runCtx.Stderr, out, exitcodes.ACPExitRuntime, err, "Host preflight validation failed")
	}
	if len(issues) > 0 {
		workflowWarn(logger, "issues", len(issues))
		return failValidation(runCtx.Stderr, out, issues, "Host preflight failed")
	}

	fmt.Fprintln(runCtx.Stdout, out.Green("Host preflight passed"))
	workflowComplete(logger)
	return exitcodes.ACPExitSuccess
}

func requireHostPreflightPrereqs() error {
	if hostRuntimeGOOS != "linux" {
		return fmt.Errorf("unsupported host OS: %s (supported: Debian 12+ or Ubuntu 24.04+ on Linux)", hostRuntimeGOOS)
	}
	osRelease, err := loadHostOSRelease(hostOSReleasePath)
	if err != nil {
		return err
	}
	if !supportedHostOS(osRelease) {
		return fmt.Errorf("unsupported host distribution: %s %s (supported: Debian 12+ or Ubuntu 24.04+)", osRelease.ID, osRelease.VersionID)
	}
	if _, err := os.Stat(hostSystemdRuntimeDir); err != nil {
		return fmt.Errorf("systemd runtime not detected at %s", hostSystemdRuntimeDir)
	}
	if !prereq.CommandExists("apt-get") {
		return fmt.Errorf("required command not found: apt-get")
	}
	if !prereq.CommandExists("docker") {
		return fmt.Errorf("required command not found: docker")
	}
	if !prereq.CommandExists("systemctl") {
		return fmt.Errorf("required command not found: systemctl")
	}
	if _, err := docker.DetectCompose(); err != nil {
		return fmt.Errorf("docker compose not available: %w", err)
	}
	return nil
}

func loadHostOSRelease(path string) (hostOSRelease, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return hostOSRelease{}, fmt.Errorf("read host os-release: %w", err)
	}
	info := hostOSRelease{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		value = strings.Trim(strings.TrimSpace(value), `"`)
		switch strings.TrimSpace(key) {
		case "ID":
			info.ID = strings.ToLower(value)
		case "VERSION_ID":
			info.VersionID = value
		}
	}
	if info.ID == "" || info.VersionID == "" {
		return hostOSRelease{}, fmt.Errorf("host os-release missing ID or VERSION_ID")
	}
	return info, nil
}

func supportedHostOS(info hostOSRelease) bool {
	switch strings.ToLower(strings.TrimSpace(info.ID)) {
	case "debian":
		return versionAtLeast(info.VersionID, "12")
	case "ubuntu":
		return versionAtLeast(info.VersionID, "24.04")
	default:
		return false
	}
}

func versionAtLeast(actual string, minimum string) bool {
	actualParts := parseVersionParts(actual)
	minimumParts := parseVersionParts(minimum)
	maxParts := len(actualParts)
	if len(minimumParts) > maxParts {
		maxParts = len(minimumParts)
	}
	for i := 0; i < maxParts; i++ {
		actualValue := 0
		if i < len(actualParts) {
			actualValue = actualParts[i]
		}
		minimumValue := 0
		if i < len(minimumParts) {
			minimumValue = minimumParts[i]
		}
		if actualValue > minimumValue {
			return true
		}
		if actualValue < minimumValue {
			return false
		}
	}
	return true
}

func parseVersionParts(value string) []int {
	fields := strings.FieldsFunc(strings.TrimSpace(value), func(r rune) bool {
		return r == '.' || r == '-'
	})
	parts := make([]int, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		n, err := strconv.Atoi(field)
		if err != nil {
			break
		}
		parts = append(parts, n)
	}
	return parts
}
