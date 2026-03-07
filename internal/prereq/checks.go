// checks.go implements prerequisite command discovery helpers.
//
// Purpose:
//
//	Track required host binaries and return consistent prerequisite errors
//	with installation guidance for operator workflows.
//
// Responsibilities:
//   - Check whether required commands are present on PATH.
//   - Record missing prerequisites for aggregate validation.
//   - Return install hints and ACP exit code mappings for failures.
//
// Scope:
//   - Covers prerequisite presence checks only.
//
// Usage:
//   - Create a checker with `New()`, call `RequireCommand` or `RequireAnyCommand`,
//     then finish with `CheckAll()`.
//
// Invariants/Assumptions:
//   - Commands are checked using `exec.LookPath`.
//   - Install hints are best-effort guidance, not an installer.
package prereq

import (
	"fmt"
	"os/exec"
	"slices"
	"strings"

	"github.com/mitchfultz/ai-control-plane/internal/exitcodes"
)

// Checker provides prerequisite checking functionality
type Checker struct {
	missing []string
}

// New creates a new prerequisite checker
func New() *Checker {
	return &Checker{
		missing: make([]string, 0),
	}
}

// CommandExists checks if a command exists in PATH
func CommandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// RequireCommand checks if a command exists and returns an error if not
func (c *Checker) RequireCommand(cmd string) error {
	if !CommandExists(cmd) {
		c.missing = append(c.missing, cmd)
		hint := getInstallHint(cmd)
		return fmt.Errorf("required binary not found: '%s'\nHint: %s", cmd, hint)
	}
	return nil
}

// RequireAnyCommand checks if at least one of the commands exists
func (c *Checker) RequireAnyCommand(label string, cmds ...string) error {
	if slices.ContainsFunc(cmds, CommandExists) {
		return nil
	}
	c.missing = append(c.missing, label)
	return fmt.Errorf("required binary not found: '%s' (tried: %s)", label, strings.Join(cmds, ", "))
}

// CheckAll returns an error if any prerequisites were missing
func (c *Checker) CheckAll() error {
	if len(c.missing) > 0 {
		return fmt.Errorf("missing prerequisites: %v", c.missing)
	}
	return nil
}

// Missing returns the list of missing commands
func (c *Checker) Missing() []string {
	return c.missing
}

// getInstallHint returns an install hint for a command
func getInstallHint(cmd string) string {
	switch cmd {
	case "docker":
		return "Install Docker from https://docs.docker.com/get-docker/"
	case "curl":
		return "Install curl: apt-get install curl (Debian/Ubuntu) or brew install curl (macOS)"
	case "jq":
		return "Install jq: apt-get install jq (Debian/Ubuntu) or brew install jq (macOS)"
	case "git":
		return "Install git: apt-get install git (Debian/Ubuntu) or brew install git (macOS)"
	case "gzip":
		return "Install gzip: apt-get install gzip (Debian/Ubuntu) - usually preinstalled"
	case "gunzip":
		return "Install gzip: apt-get install gzip (Debian/Ubuntu) - usually preinstalled"
	case "nc":
		return "Install netcat: apt-get install netcat-openbsd (Debian/Ubuntu) or brew install netcat (macOS)"
	case "timeout":
		return "Install coreutils: apt-get install coreutils (Debian/Ubuntu) - usually preinstalled"
	case "gtimeout":
		return "Install coreutils: brew install coreutils (macOS)"
	case "psql":
		return "Install postgresql-client: apt-get install postgresql-client (Debian/Ubuntu)"
	case "python3":
		return "Install Python 3: apt-get install python3 (Debian/Ubuntu) or brew install python (macOS)"
	default:
		return fmt.Sprintf("Install %s using your system package manager", cmd)
	}
}

// ExitCodeForError returns the appropriate exit code for a prerequisite error
func ExitCodeForError(err error) int {
	if err != nil {
		return exitcodes.ACPExitPrereq
	}
	return exitcodes.ACPExitSuccess
}
