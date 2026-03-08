// Package collectors provides status collectors backed by typed ACP services.
//
// Purpose:
//
//	Expose a key inventory collector that consumes the shared typed database
//	service instead of direct SQL command execution.
//
// Responsibilities:
//   - Convert typed key counts into status.ComponentStatus.
//   - Preserve operator guidance around missing or expired keys.
//
// Non-scope:
//   - Does not execute collector-local SQL or mutate key state.
//
// Invariants/Assumptions:
//   - Key counts come from the shared typed database service.
//
// Scope:
//   - Virtual key status collection only.
//
// Usage:
//   - Construct with NewKeysCollector(client) and call Collect(ctx).
package collectors

import (
	"context"
	"fmt"

	"github.com/mitchfultz/ai-control-plane/internal/db"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

// KeysCollector checks virtual key counts and status.
type KeysCollector struct {
	reader db.ReadonlyServiceReader
}

// NewKeysCollector creates a new keys collector.
func NewKeysCollector(reader db.ReadonlyServiceReader) KeysCollector {
	return KeysCollector{reader: reader}
}

// Name returns the collector's domain name.
func (c KeysCollector) Name() string {
	return "keys"
}

// Collect gathers virtual key status information.
func (c KeysCollector) Collect(ctx context.Context) status.ComponentStatus {
	summary, err := c.reader.KeySummary(ctx)
	details := status.ComponentDetails{
		TotalKeys:   summary.Total,
		ActiveKeys:  summary.Active,
		ExpiredKeys: summary.Expired,
	}
	if err != nil {
		details.Error = err.Error()
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelWarning,
			Message: "Could not query key count",
			Details: details,
			Suggestions: []string{
				"Table may not exist yet - LiteLLM creates tables on first use",
			},
		}
	}

	if summary.Total == 0 {
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelWarning,
			Message: "No virtual keys configured",
			Details: details,
			Suggestions: []string{
				"Generate a key: acpctl key gen my-key --budget 10.00",
				"Or: make key-gen ALIAS=my-key BUDGET=10.00",
			},
		}
	}

	if summary.Expired > 0 {
		return status.ComponentStatus{
			Name:    c.Name(),
			Level:   status.HealthLevelWarning,
			Message: fmt.Sprintf("%d keys, %d expired", summary.Total, summary.Expired),
			Details: details,
			Suggestions: []string{
				"Review expired keys: acpctl db status",
				"Revoke unused keys: acpctl key revoke <alias>",
			},
		}
	}

	return status.ComponentStatus{
		Name:    c.Name(),
		Level:   status.HealthLevelHealthy,
		Message: fmt.Sprintf("%d active keys", summary.Active),
		Details: details,
	}
}
