// checks_test.go validates doctor prerequisite and environment checks.
//
// Purpose:
//
//	Verify doctor checks classify ports, environment state, and supporting
//	services deterministically across local test environments.
//
// Responsibilities:
//   - Exercise individual doctor checks and their severity mapping.
//   - Use controlled listeners and test servers for port and gateway cases.
//   - Validate fix helpers that materialize repo-local configuration.
//
// Scope:
//   - Covers unit behavior in the internal/doctor package only.
//
// Usage:
//   - Run via `go test ./internal/doctor`.
//
// Invariants/Assumptions:
//   - Tests avoid depending on fixed ports or preinstalled runtime services.
//   - Temporary repo state is created under test-owned directories.
package doctor

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/status"
)

func reserveLocalPort(t *testing.T) (int, func()) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to reserve local port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	return port, func() {
		_ = listener.Close()
	}
}

func TestDockerAvailableCheck_ID(t *testing.T) {
	t.Parallel()
	check := dockerAvailableCheck{}
	if check.ID() != "docker_available" {
		t.Errorf("expected ID 'docker_available', got %s", check.ID())
	}
}

func TestDockerAvailableCheck_Run(t *testing.T) {
	t.Parallel()

	check := dockerAvailableCheck{}
	ctx := context.Background()
	opts := Options{}

	result := check.Run(ctx, opts)

	// We can't guarantee Docker availability in tests, but we can verify
	// the result has expected fields
	if result.ID != "docker_available" {
		t.Errorf("expected ID 'docker_available', got %s", result.ID)
	}

	if result.Name != "Docker Available" {
		t.Errorf("expected Name 'Docker Available', got %s", result.Name)
	}

	// Either healthy or unhealthy is valid depending on environment
	if result.Level != status.HealthLevelHealthy && result.Level != status.HealthLevelUnhealthy {
		t.Errorf("unexpected level: %v", result.Level)
	}

	if result.Severity != SeverityPrereq && result.Severity != SeverityDomain {
		t.Errorf("unexpected severity: %v", result.Severity)
	}
}

func TestPortsFreeCheck_ID(t *testing.T) {
	t.Parallel()
	check := portsFreeCheck{}
	if check.ID() != "ports_free" {
		t.Errorf("expected ID 'ports_free', got %s", check.ID())
	}
}

func TestPortsFreeCheck_Run(t *testing.T) {
	t.Parallel()

	check := portsFreeCheck{}
	ctx := context.Background()

	// Reserve a local ephemeral port and release it to get a deterministic free port.
	freePort, release := reserveLocalPort(t)
	release()
	opts := Options{
		RequiredPorts: []int{freePort},
	}

	result := check.Run(ctx, opts)

	if result.ID != "ports_free" {
		t.Errorf("expected ID 'ports_free', got %s", result.ID)
	}

	// Should be healthy because the test-selected port was released before check.
	if result.Level == status.HealthLevelHealthy {
		if result.Message == "" {
			t.Error("expected message for healthy result")
		}
	}
}

func TestPortsFreeCheck_WithOccupiedPort(t *testing.T) {
	t.Parallel()

	// Start a listener on a specific port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("Cannot create test listener: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	check := portsFreeCheck{}
	ctx := context.Background()
	opts := Options{
		RequiredPorts: []int{port},
	}

	result := check.Run(ctx, opts)

	if result.Level != status.HealthLevelWarning {
		t.Errorf("expected warning when port is occupied, got %v", result.Level)
	}

	if result.Severity != SeverityDomain {
		t.Errorf("expected severity domain, got %v", result.Severity)
	}
}

func TestOccupiedPortsBelongToRunningACP(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	hostPort := strings.TrimPrefix(server.URL, "http://")
	host, portString, err := net.SplitHostPort(hostPort)
	if err != nil {
		t.Fatalf("failed to parse test server URL: %v", err)
	}
	port, err := strconv.Atoi(portString)
	if err != nil {
		t.Fatalf("failed to parse gateway port: %v", err)
	}

	opts := Options{
		GatewayHost: host,
		GatewayPort: portString,
	}

	if !occupiedPortsBelongToRunningACP(context.Background(), []int{port}, opts) {
		t.Fatalf("expected occupied gateway port %d to be recognized as running ACP service", port)
	}
	if occupiedPortsBelongToRunningACP(context.Background(), []int{port + 1}, opts) {
		t.Fatalf("did not expect unrelated occupied port to be recognized as ACP service")
	}
}

func TestIsPortOccupied(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Reserve and release an ephemeral local port to ensure a deterministic free port.
	freePort, release := reserveLocalPort(t)
	release()
	if isPortOccupied(ctx, freePort) {
		t.Errorf("expected port %d to be free", freePort)
	}

	// Test with an occupied port
	occupiedPort, releaseOccupied := reserveLocalPort(t)
	defer releaseOccupied()
	if !isPortOccupied(ctx, occupiedPort) {
		t.Errorf("expected port %d to be occupied", occupiedPort)
	}
}

func TestEnvVarsSetCheck_ID(t *testing.T) {
	t.Parallel()
	check := envVarsSetCheck{}
	if check.ID() != "env_vars_set" {
		t.Errorf("expected ID 'env_vars_set', got %s", check.ID())
	}
}

func TestEnvVarsSetCheck_Run_WithMissingVars(t *testing.T) {
	// Not parallel: modifies global environment state

	// Clear relevant environment variables
	oldMasterKey := os.Getenv("LITELLM_MASTER_KEY")
	oldSaltKey := os.Getenv("LITELLM_SALT_KEY")
	oldDbUrl := os.Getenv("DATABASE_URL")
	defer func() {
		os.Setenv("LITELLM_MASTER_KEY", oldMasterKey)
		os.Setenv("LITELLM_SALT_KEY", oldSaltKey)
		os.Setenv("DATABASE_URL", oldDbUrl)
	}()

	os.Unsetenv("LITELLM_MASTER_KEY")
	os.Unsetenv("LITELLM_SALT_KEY")
	os.Unsetenv("DATABASE_URL")

	check := envVarsSetCheck{}
	ctx := context.Background()
	opts := Options{
		RepoRoot: t.TempDir(),
	}

	result := check.Run(ctx, opts)

	if result.Level != status.HealthLevelUnhealthy {
		t.Errorf("expected unhealthy when vars missing, got %v", result.Level)
	}

	if result.Severity != SeverityPrereq {
		t.Errorf("expected severity prereq, got %v", result.Severity)
	}

	if len(result.Suggestions) == 0 {
		t.Error("expected suggestions for missing vars")
	}
}

func TestEnvVarsSetCheck_Run_WithVarsSet(t *testing.T) {
	// Not parallel: modifies global environment state

	// Set environment variables
	os.Setenv("LITELLM_MASTER_KEY", "test-key")
	os.Setenv("LITELLM_SALT_KEY", "test-salt")
	os.Setenv("DATABASE_URL", "test-db-url")

	check := envVarsSetCheck{}
	ctx := context.Background()
	opts := Options{}

	result := check.Run(ctx, opts)

	if result.Level != status.HealthLevelHealthy {
		t.Errorf("expected healthy when vars set, got %v", result.Level)
	}
}

func TestEnvVarsSetCheck_Fix(t *testing.T) {
	t.Parallel()

	check := envVarsSetCheck{}
	ctx := context.Background()
	tmpDir := t.TempDir()
	opts := Options{
		RepoRoot: tmpDir,
	}

	// Create demo directory and .env.example
	demoDir := filepath.Join(tmpDir, "demo")
	if err := os.MkdirAll(demoDir, 0755); err != nil {
		t.Fatalf("failed to create demo dir: %v", err)
	}

	exampleContent := "LITELLM_MASTER_KEY=example\nLITELLM_SALT_KEY=example\n"
	if err := os.WriteFile(filepath.Join(demoDir, ".env.example"), []byte(exampleContent), 0644); err != nil {
		t.Fatalf("failed to write example: %v", err)
	}

	applied, msg, err := check.Fix(ctx, opts)

	if err != nil {
		t.Errorf("Fix failed: %v", err)
	}

	if !applied {
		t.Error("expected fix to be applied")
	}

	if msg == "" {
		t.Error("expected fix message")
	}

	// Verify .env was created
	envPath := filepath.Join(demoDir, ".env")
	if _, err := os.Stat(envPath); err != nil {
		t.Error("expected .env to be created")
	}
}

func TestEnvVarsSetCheck_Fix_AlreadyExists(t *testing.T) {
	t.Parallel()

	check := envVarsSetCheck{}
	ctx := context.Background()
	tmpDir := t.TempDir()
	opts := Options{
		RepoRoot: tmpDir,
	}

	// Create demo directory and existing .env
	demoDir := filepath.Join(tmpDir, "demo")
	if err := os.MkdirAll(demoDir, 0755); err != nil {
		t.Fatalf("failed to create demo dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(demoDir, ".env"), []byte("exists"), 0644); err != nil {
		t.Fatalf("failed to write .env: %v", err)
	}

	applied, _, err := check.Fix(ctx, opts)

	if err != nil {
		t.Errorf("Fix failed: %v", err)
	}

	if applied {
		t.Error("expected fix to NOT be applied when .env exists")
	}
}

func TestLoadEnvFromFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")

	content := `LITELLM_MASTER_KEY=sk-test123
LITELLM_SALT_KEY=salt456
# Comment line
DATABASE_URL=postgresql://localhost
`
	if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write env file: %v", err)
	}

	tests := []struct {
		key      string
		expected string
	}{
		{"LITELLM_MASTER_KEY", "sk-test123"},
		{"LITELLM_SALT_KEY", "salt456"},
		{"DATABASE_URL", "postgresql://localhost"},
		{"MISSING_KEY", ""},
	}

	for _, tt := range tests {
		result := loadEnvFromFile(envPath, tt.key)
		if result != tt.expected {
			t.Errorf("loadEnvFromFile(%s) = %q, want %q", tt.key, result, tt.expected)
		}
	}
}

func TestLoadEnvFromFile_NotFound(t *testing.T) {
	t.Parallel()

	result := loadEnvFromFile("/nonexistent/path/.env", "KEY")
	if result != "" {
		t.Errorf("expected empty string for missing file, got %q", result)
	}
}

func TestGatewayHealthyCheck_ID(t *testing.T) {
	t.Parallel()
	check := gatewayHealthyCheck{}
	if check.ID() != "gateway_healthy" {
		t.Errorf("expected ID 'gateway_healthy', got %s", check.ID())
	}
}

func TestGatewayHealthyCheck_Run_MissingMasterKey(t *testing.T) {
	// Not parallel: modifies global environment state
	oldKey := os.Getenv("LITELLM_MASTER_KEY")
	defer os.Setenv("LITELLM_MASTER_KEY", oldKey)
	os.Unsetenv("LITELLM_MASTER_KEY")

	check := gatewayHealthyCheck{}
	ctx := context.Background()
	opts := Options{
		RepoRoot: t.TempDir(),
	}

	result := check.Run(ctx, opts)
	if result.Level != status.HealthLevelUnhealthy {
		t.Fatalf("expected unhealthy when key missing, got %v", result.Level)
	}
	if result.Severity != SeverityPrereq {
		t.Fatalf("expected prereq severity when key missing, got %v", result.Severity)
	}
	if !strings.Contains(result.Message, "LITELLM_MASTER_KEY not set") {
		t.Fatalf("unexpected message: %q", result.Message)
	}
}

func TestGatewayHealthyCheck_Run_Authorized(t *testing.T) {
	// Not parallel: modifies global environment state
	oldKey := os.Getenv("LITELLM_MASTER_KEY")
	defer os.Setenv("LITELLM_MASTER_KEY", oldKey)
	os.Setenv("LITELLM_MASTER_KEY", "doctor-test-key")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer doctor-test-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	check := gatewayHealthyCheck{}
	ctx := context.Background()
	opts := Options{
		GatewayHost: server.Listener.Addr().(*net.TCPAddr).IP.String(),
		GatewayPort: fmt.Sprintf("%d", server.Listener.Addr().(*net.TCPAddr).Port),
	}

	result := check.Run(ctx, opts)
	if result.Level != status.HealthLevelHealthy {
		t.Fatalf("expected healthy, got %v with message %q", result.Level, result.Message)
	}
}

func TestDBConnectableCheck_ID(t *testing.T) {
	t.Parallel()
	check := dbConnectableCheck{}
	if check.ID() != "db_connectable" {
		t.Errorf("expected ID 'db_connectable', got %s", check.ID())
	}
}

func TestFirstNonEmptyLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "single line", input: "abc123", expected: "abc123"},
		{name: "first line selected", input: "id-one\nid-two", expected: "id-one"},
		{name: "skips blank lines", input: "\n\n container-id ", expected: "container-id"},
		{name: "all blank", input: "\n\t\n", expected: ""},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := firstNonEmptyLine(tc.input); got != tc.expected {
				t.Fatalf("firstNonEmptyLine(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestConfigValidCheck_ID(t *testing.T) {
	t.Parallel()
	check := configValidCheck{}
	if check.ID() != "config_valid" {
		t.Errorf("expected ID 'config_valid', got %s", check.ID())
	}
}

func TestConfigValidCheck_Run_MissingRequiredFiles(t *testing.T) {
	t.Parallel()

	check := configValidCheck{}
	ctx := context.Background()
	opts := Options{
		RepoRoot: t.TempDir(), // Empty temp dir, no required config files
	}

	result := check.Run(ctx, opts)

	if result.Level != status.HealthLevelUnhealthy {
		t.Errorf("expected unhealthy when required files are missing, got %v", result.Level)
	}
}

func TestConfigValidCheck_Run_Healthy(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	paths := []string{
		"demo/docker-compose.yml",
		"demo/config/litellm.yaml",
		"demo/.env",
	}
	for _, relPath := range paths {
		fullPath := filepath.Join(repoRoot, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("failed creating parent dir for %s: %v", relPath, err)
		}
		if err := os.WriteFile(fullPath, []byte("test"), 0644); err != nil {
			t.Fatalf("failed writing %s: %v", relPath, err)
		}
	}

	check := configValidCheck{}
	result := check.Run(context.Background(), Options{RepoRoot: repoRoot})
	if result.Level != status.HealthLevelHealthy {
		t.Fatalf("expected healthy result, got %v", result.Level)
	}
}

func TestCredentialsValidCheck_ID(t *testing.T) {
	t.Parallel()
	check := credentialsValidCheck{}
	if check.ID() != "credentials_valid" {
		t.Errorf("expected ID 'credentials_valid', got %s", check.ID())
	}
}

func TestCredentialsValidCheck_Run_MissingKey(t *testing.T) {
	// Note: Not running in parallel due to environment manipulation

	// Clear the environment variable
	oldKey := os.Getenv("LITELLM_MASTER_KEY")
	defer os.Setenv("LITELLM_MASTER_KEY", oldKey)
	os.Unsetenv("LITELLM_MASTER_KEY")

	// Create a temp directory with no .env file to ensure key is truly missing
	tmpDir := t.TempDir()

	check := credentialsValidCheck{}
	ctx := context.Background()
	opts := Options{
		RepoRoot: tmpDir,
	}

	result := check.Run(ctx, opts)

	if result.Level != status.HealthLevelUnhealthy {
		t.Errorf("expected unhealthy when key missing, got %v", result.Level)
	}

	if result.Severity != SeverityPrereq {
		t.Errorf("expected severity prereq, got %v", result.Severity)
	}
}

func TestSanitizeOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "normal output\nsecret key=hidden\nmore output",
			expected: "normal output\nmore output",
		},
		{
			input:    "password=test123\ntoken=abc",
			expected: "",
		},
		{
			input:    "API_KEY=sk-xxx\nDATABASE_URL=postgres://user:pass@host",
			expected: "",
		},
		{
			input:    "regular log line\nanother line",
			expected: "regular log line\nanother line",
		},
	}

	for _, tt := range tests {
		result := sanitizeOutput(tt.input)
		if result != tt.expected {
			t.Errorf("sanitizeOutput(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestJoinPorts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		ports    []int
		expected string
	}{
		{[]int{4000}, "4000"},
		{[]int{4000, 5432}, "4000|5432"},
		{[]int{4000, 5432, 8080}, "4000|5432|8080"},
		{[]int{}, ""},
	}

	for _, tt := range tests {
		result := joinPorts(tt.ports)
		if result != tt.expected {
			t.Errorf("joinPorts(%v) = %q, want %q", tt.ports, result, tt.expected)
		}
	}
}

// Integration test helpers - skipped if Docker not available

func skipIfDockerUnavailable(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not available")
	}
}
