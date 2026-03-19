// Package collectors provides status collectors backed by typed ACP services.
//
// Purpose:
//
//	Expose recent gateway traffic, cost, and error signals from the shared typed
//	database service for operator status and dashboard views.
//
// Responsibilities:
//   - Convert recent request, token, spend, and non-success counts into
//     status.ComponentStatus.
//   - Align elevated error-rate warnings with the documented detection threshold.
//   - Preserve operator guidance when spend-log data is not available yet.
//
// Scope:
//   - Recent gateway traffic status collection only.
//
// Usage:
//   - Construct with NewTrafficCollector(reader) and call Collect(ctx).
//
// Invariants/Assumptions:
//   - Traffic metrics come from the shared typed readonly database service.
//   - Warning threshold matches DR-003: >10% non-success over >=10 requests.
package collectors

import (
	"context"
	"fmt"

	"github.com/mitchfultz/ai-control-plane/internal/db"
	"github.com/mitchfultz/ai-control-plane/internal/status"
)

const trafficErrorRateWarningThreshold = 10.0

// TrafficCollector summarizes recent gateway traffic, cost, and non-success rates.
type TrafficCollector struct {
	reader db.ReadonlyServiceReader
}

// NewTrafficCollector creates a traffic collector.
func NewTrafficCollector(reader db.ReadonlyServiceReader) TrafficCollector {
	return TrafficCollector{reader: reader}
}

// Name returns the collector's domain name.
func (c TrafficCollector) Name() string {
	return "traffic"
}

// Collect gathers recent traffic status information.
func (c TrafficCollector) Collect(ctx context.Context) status.ComponentStatus {
	summary, err := c.reader.TrafficSummary(ctx)
	details := status.ComponentDetails{
		SpendLogsTableExists: summary.SpendLogsTableExists,
		TotalRequests24h:     summary.TotalRequests24h,
		TotalTokens24h:       summary.TotalTokens24h,
		TotalSpend24h:        summary.TotalSpend24h,
		ErrorRequests24h:     summary.ErrorRequests24h,
	}
	if summary.TotalRequests24h > 0 {
		details.ErrorRatePercent24h = float64(summary.ErrorRequests24h) * 100 / float64(summary.TotalRequests24h)
	}
	if err != nil {
		return readonlyQueryWarning(c.Name(), "Could not query recent gateway traffic", details, err)
	}
	if !summary.SpendLogsTableExists {
		return componentStatus(c.Name(), status.HealthLevelHealthy, "No gateway traffic data yet", details,
			"Traffic metrics appear after routed gateway requests are made",
		)
	}
	if summary.TotalRequests24h == 0 {
		return componentStatus(c.Name(), status.HealthLevelHealthy, "No gateway traffic in last 24h", details)
	}
	if summary.TotalRequests24h >= 10 && details.ErrorRatePercent24h > trafficErrorRateWarningThreshold {
		return componentStatus(c.Name(), status.HealthLevelWarning,
			fmt.Sprintf("%d requests in last 24h, %.1f%% non-success", summary.TotalRequests24h, details.ErrorRatePercent24h),
			details,
			"Review recent gateway auth/provider failures",
			"Check DR-003 guidance in docs/security/DETECTION.md",
		)
	}
	message := fmt.Sprintf("%d requests in last 24h", summary.TotalRequests24h)
	if summary.ErrorRequests24h > 0 {
		message = fmt.Sprintf("%d requests in last 24h, %d non-success", summary.TotalRequests24h, summary.ErrorRequests24h)
	}
	return componentStatus(c.Name(), status.HealthLevelHealthy, message, details)
}
