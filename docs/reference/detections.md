# Detection Rules Reference

> Generated from `demo/config/detection_rules.yaml`. Do not edit manually.

| Rule ID | Name | Severity | Enabled | Status | Coverage | Expected Signal |
| --- | --- | --- | --- | --- | --- | --- |
| `DR-001` | Non-Approved Model Access | high | yes | validated | decision-grade | Any request model not in approved list within last 24h |
| `DR-002` | Token Usage Spike | medium | yes | example | demo | Keys exceeding static 24h token threshold |
| `DR-003` | High Block/Error Rate | medium | yes | validated | decision-grade | Keys with >10% non-success status over >=10 requests |
| `DR-004` | Budget Exhaustion Warning | low | yes | validated | decision-grade | Keys with less than 20% budget remaining |
| `DR-005` | Rapid Request Rate | medium | yes | example | demo | Any key exceeding 60 requests/minute in last hour |
| `DR-006` | Failed Authentication Attempts | high | yes | validated | decision-grade | Keys with >=5 failed attempts in last 24h |
| `DR-007` | Budget Threshold Alert | medium | yes | validated | decision-grade | Keys with spend at or above 80% of budget |
| `DR-008` | DLP Block Event Detected | high | yes | validated | decision-grade | Requests blocked by guardrail markers in status/response metadata |
| `DR-009` | Repeated PII Submission Attempts | high | yes | example | demo | Same key repeatedly triggering DLP-style failures within one hour |
| `DR-010` | Potential Prompt Injection Attempt | medium | yes | example | demo | Failed requests containing prompt-injection or jailbreak markers in gateway response metadata within the last 24 hours |

