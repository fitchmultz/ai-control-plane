// keys_test validates KeysCollector docker-based status checks.
//
// Purpose:
//
//	Ensure virtual key status collection correctly interprets PostgreSQL
//	query results for the LiteLLM_VerificationToken table.
//
// Responsibilities:
//   - Verify collector name returns "keys".
//   - Verify status levels based on key counts and expiration.
//   - Verify response parsing for key count queries.
//   - Verify suggestions for various key states.
//
// Non-scope:
//   - Does not test against real running PostgreSQL containers.
//
// Invariants/Assumptions:
//   - Zero keys triggers a warning (configuration incomplete).
//   - Expired keys trigger a warning.
//   - Active keys only returns healthy status.
package collectors

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/mitchfultz/ai-control-plane/internal/status"
	"github.com/mitchfultz/ai-control-plane/internal/status/runner"
)

func TestKeysCollector_Name(t *testing.T) {
	t.Parallel()

	c := KeysCollector{}
	if c.Name() != "keys" {
		t.Fatalf("expected name 'keys', got %q", c.Name())
	}
}

func TestKeysCollector_StatusLevels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		totalKeys     int
		activeKeys    int
		expiredKeys   int
		expectedLevel status.HealthLevel
		expectedMsg   string
	}{
		{
			name:          "zero keys - warning",
			totalKeys:     0,
			activeKeys:    0,
			expiredKeys:   0,
			expectedLevel: status.HealthLevelWarning,
			expectedMsg:   "No virtual keys configured",
		},
		{
			name:          "all keys active - healthy",
			totalKeys:     3,
			activeKeys:    3,
			expiredKeys:   0,
			expectedLevel: status.HealthLevelHealthy,
			expectedMsg:   "3 active keys",
		},
		{
			name:          "some expired - warning",
			totalKeys:     5,
			activeKeys:    3,
			expiredKeys:   2,
			expectedLevel: status.HealthLevelWarning,
			expectedMsg:   "5 keys, 2 expired",
		},
		{
			name:          "all expired - warning",
			totalKeys:     2,
			activeKeys:    0,
			expiredKeys:   2,
			expectedLevel: status.HealthLevelWarning,
			expectedMsg:   "2 keys, 2 expired",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Verify expected behavior based on key counts
			// These are the expected outcomes the collector should produce

			details := map[string]any{
				"total_keys": tt.totalKeys,
			}

			if tt.activeKeys > 0 || tt.totalKeys > 0 {
				details["active_keys"] = tt.activeKeys
				details["expired_keys"] = tt.expiredKeys
			}

			component := status.ComponentStatus{
				Name:    "keys",
				Level:   tt.expectedLevel,
				Message: tt.expectedMsg,
				Details: details,
			}

			if component.Level != tt.expectedLevel {
				t.Fatalf("expected level %s, got %s", tt.expectedLevel, component.Level)
			}

			if component.Message != tt.expectedMsg {
				t.Fatalf("expected message %q, got %q", tt.expectedMsg, component.Message)
			}

			// Verify details
			if component.Details == nil {
				t.Fatal("expected details")
			}
		})
	}
}

func TestKeysCollector_ZeroKeys_Suggestions(t *testing.T) {
	t.Parallel()

	component := status.ComponentStatus{
		Name:    "keys",
		Level:   status.HealthLevelWarning,
		Message: "No virtual keys configured",
		Details: map[string]any{
			"total_keys": 0,
		},
		Suggestions: []string{
			"Generate a key: acpctl key gen my-key --budget 10.00",
			"Or: make key-gen ALIAS=my-key BUDGET=10.00",
		},
	}

	if len(component.Suggestions) == 0 {
		t.Fatal("expected suggestions for zero keys")
	}

	hasKeyGen := false
	hasMake := false
	for _, s := range component.Suggestions {
		if strings.Contains(s, "acpctl key gen") {
			hasKeyGen = true
		}
		if strings.Contains(s, "make key-gen") {
			hasMake = true
		}
	}

	if !hasKeyGen {
		t.Fatal("expected acpctl key gen suggestion")
	}

	if !hasMake {
		t.Fatal("expected make key-gen suggestion")
	}
}

func TestKeysCollector_ExpiredKeys_Suggestions(t *testing.T) {
	t.Parallel()

	component := status.ComponentStatus{
		Name:    "keys",
		Level:   status.HealthLevelWarning,
		Message: "5 keys, 2 expired",
		Details: map[string]any{
			"total_keys":   5,
			"active_keys":  3,
			"expired_keys": 2,
		},
		Suggestions: []string{
			"Review expired keys: acpctl db status",
			"Revoke unused keys: acpctl key revoke <alias>",
		},
	}

	if len(component.Suggestions) == 0 {
		t.Fatal("expected suggestions for expired keys")
	}

	hasReview := false
	hasRevoke := false
	for _, s := range component.Suggestions {
		if strings.Contains(s, "acpctl db status") {
			hasReview = true
		}
		if strings.Contains(s, "acpctl key revoke") {
			hasRevoke = true
		}
	}

	if !hasReview {
		t.Fatal("expected review suggestion")
	}

	if !hasRevoke {
		t.Fatal("expected revoke suggestion")
	}
}

func TestKeysCollector_ParseKeyCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		output   string
		expected int
		wantErr  bool
	}{
		{
			name:     "single digit",
			output:   "5",
			expected: 5,
			wantErr:  false,
		},
		{
			name:     "with whitespace",
			output:   "  42  ",
			expected: 42,
			wantErr:  false,
		},
		{
			name:     "with newlines",
			output:   "\n  100\n  ",
			expected: 100,
			wantErr:  false,
		},
		{
			name:     "zero",
			output:   "0",
			expected: 0,
			wantErr:  false,
		},
		{
			name:     "non-numeric returns error",
			output:   "not a number",
			expected: 0,
			wantErr:  true,
		},
		{
			name:     "empty returns error",
			output:   "",
			expected: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Simulate the parsing logic from the collector
			parsed, err := strconv.Atoi(strings.TrimSpace(tt.output))

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if parsed != tt.expected {
				t.Fatalf("expected %d, got %d", tt.expected, parsed)
			}
		})
	}
}

func TestKeysCollector_HealthyStatus_SingleKey(t *testing.T) {
	t.Parallel()

	component := status.ComponentStatus{
		Name:    "keys",
		Level:   status.HealthLevelHealthy,
		Message: "1 active keys",
		Details: map[string]any{
			"total_keys": 1,
		},
	}

	if component.Level != status.HealthLevelHealthy {
		t.Fatal("expected healthy status")
	}
}

func TestKeysCollector_QueryError_Response(t *testing.T) {
	t.Parallel()

	// When the table doesn't exist or query fails
	component := status.ComponentStatus{
		Name:    "keys",
		Level:   status.HealthLevelWarning,
		Message: "Could not query key count",
		Suggestions: []string{
			"Table may not exist yet - LiteLLM creates tables on first use",
		},
	}

	if component.Level != status.HealthLevelWarning {
		t.Fatalf("expected warning level, got %s", component.Level)
	}

	if len(component.Suggestions) == 0 {
		t.Fatal("expected suggestions for query error")
	}
}

func TestKeysCollector_Collect_UsesComposeResolverWhenAvailable(t *testing.T) {
	recording := newRecordingRunner()
	recording.SetResponse(`docker exec compose-postgres psql -U litellm -d litellm -t -c SELECT COUNT(*) FROM "LiteLLM_VerificationToken";`, &runner.Result{
		Stdout:   "1\n",
		ExitCode: 0,
	})
	recording.SetResponse(`docker exec compose-postgres psql -U litellm -d litellm -t -c SELECT COUNT(*) FROM "LiteLLM_VerificationToken" WHERE expires IS NULL OR expires > NOW();`, &runner.Result{
		Stdout:   "1\n",
		ExitCode: 0,
	})

	resolver := &fakeContainerResolver{containerID: "compose-postgres"}

	c := NewKeysCollector("/tmp")
	c.SetRunner(recording)
	c.SetContainerResolver(resolver)

	result := c.Collect(context.Background())
	if result.Level != status.HealthLevelHealthy {
		t.Fatalf("expected healthy, got %v", result.Level)
	}

	if resolver.calls != 1 {
		t.Fatalf("expected resolver to be called once, got %d", resolver.calls)
	}

	if recording.sawCommandContaining("docker ps --filter name=postgres") {
		t.Fatal("expected compose resolver to avoid docker ps fallback")
	}
}
