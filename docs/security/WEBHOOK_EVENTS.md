# AI Control Plane - Webhook Events Reference

## Overview

The AI Control Plane can push real-time event notifications to external systems via webhooks. This enables integration with ticketing systems, chat platforms, SIEM tools, and custom automation without polling.

This document is the **outbound webhook event reference**. For the supported host-first **inbound** normalization path that takes vendor export/webhook payloads from file or stdin and converts them into ACP evidence artifacts, use `acpctl evidence ingest` and see [../evidence/VENDOR_EVIDENCE_INGEST.md](../evidence/VENDOR_EVIDENCE_INGEST.md).

## Quick Start

### Enable Webhooks

```bash
# Set the master switch
export WEBHOOKS_ENABLED=true

# Or set in demo/.env
echo "WEBHOOKS_ENABLED=true" >> demo/.env
```

### Configure an Endpoint

Edit `demo/config/webhooks.yaml`:

```yaml
webhooks:
  enabled: true
  endpoints:
    - name: "my-webhook"
      enabled: true
      url: "${WEBHOOK_URL:-https://example.com/webhook}"
      secret_env: "WEBHOOK_SECRET"
      events:
        - "key.*"
        - "approval.*"
```

### Test Webhook Delivery

```bash
# Validate detection pipeline (policy.violation / detection.triggered signals)
make validate-detections

# Trigger key lifecycle events
make key-gen ALIAS=webhook-test BUDGET=1.00
make key-revoke ALIAS=webhook-test

# Inspect receiver logs to confirm delivery
```

## Configuration

Webhooks are configured in `demo/config/webhooks.yaml`.

### Master Switch

Enable or disable all webhooks globally:

```yaml
webhooks:
  enabled: true  # Set via WEBHOOKS_ENABLED env var
```

### Global Settings

Configure retry behavior:

```yaml
webhooks:
  settings:
    retry_count: 3              # Number of delivery attempts
    retry_delay_seconds: 1      # Initial delay between retries
    retry_backoff_factor: 2     # Exponential backoff multiplier
    timeout_seconds: 10         # HTTP timeout per attempt
    signature_algorithm: "sha256"
```

### Endpoint Configuration

Each endpoint has:

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Unique identifier for the endpoint |
| `enabled` | Yes | Whether this endpoint is active |
| `url` | Yes | Webhook URL (supports env var expansion) |
| `secret_env` | No | Environment variable name containing HMAC secret |
| `events` | Yes | Array of event patterns to subscribe to |
| `format` | No | Output format: `generic` (default), `slack`, `pagerduty` |
| `headers` | No | Additional HTTP headers |

### Event Pattern Matching

Events support glob-style pattern matching:

| Pattern | Matches |
|---------|---------|
| `key.created` | Exact match only |
| `key.*` | All key events (key.created, key.revoked, key.expired) |
| `approval.*` | All approval events |
| `*` | All events |

## Event Types

| Event Type | Description | Trigger |
|------------|-------------|---------|
| `key.created` | New API key generated | `make key-gen` or `make key-rotate` |
| `key.revoked` | API key revoked | `make key-revoke` or `make key-rotate REVOKE_OLD=1` |
| `key.expired` | Key budget exhausted | Detection rule DR-004 |
| `approval.requested` | New approval request | Approval workflow integration event (no direct Make target) |
| `approval.approved` | Request approved | Approval workflow integration event (no direct Make target) |
| `approval.rejected` | Request rejected | Approval workflow integration event (no direct Make target) |
| `approval.escalated` | Request SLA breached | SLA monitor |
| `budget.threshold` | Budget threshold crossed | Detection rule DR-007 |
| `policy.violation` | Policy violation detected | Detection rules |
| `detection.triggered` | Detection rule triggered | `make detection` |

## Payload Schema

All webhooks use a standardized envelope:

```json
{
  "event_id": "evt_abc123def456",
  "event_type": "key.revoked",
  "timestamp": "2026-02-16T04:49:26Z",
  "idempotency_key": "key.revoked:my-key:2026-02-16T04:49:26Z",
  "source": "ai-control-plane",
  "version": "1.0",
  "payload": {
    // Event-specific data
  }
}
```

### Idempotency

The `idempotency_key` field enables receivers to deduplicate deliveries:

- Format: `{event_type}:{primary_identifier}:{timestamp}`
- Sent as `X-Idempotency-Key` HTTP header
- Important for retry scenarios

### Event-Specific Payloads

#### key.created

```json
{
  "alias": "my-key",
  "budget": 10.00,
  "role": "developer",
  "models": ["openai-gpt5.2", "claude-haiku-4-5"]
}
```

#### key.revoked

```json
{
  "alias": "my-key",
  "timestamp": "2026-02-16T04:49:26Z"
}
```

#### approval.requested

```json
{
  "request_id": "req-20260216-abc123",
  "alias": "requested-key",
  "budget": "50.00",
  "requester": "user@example.com",
  "role": "developer"
}
```

#### approval.approved

```json
{
  "request_id": "req-20260216-abc123",
  "approved_by": "admin@example.com"
}
```

#### approval.rejected

```json
{
  "request_id": "req-20260216-abc123",
  "rejected_by": "admin@example.com",
  "reason": "Budget exceeds department limits"
}
```

#### approval.escalated

```json
{
  "request_id": "req-20260216-abc123",
  "reason": "SLA_breached"
}
```

## Signature Verification

Webhooks include an HMAC-SHA256 signature in the `X-ACP-Signature` header.

### Header Format

```
X-ACP-Signature: sha256=<hex_digest>
```

### Verification Examples

#### Python

```python
import hmac
import hashlib

def verify_signature(payload: bytes, signature: str, secret: str) -> bool:
    """Verify webhook HMAC signature."""
    expected = hmac.new(
        secret.encode(),
        payload,
        hashlib.sha256
    ).hexdigest()
    return signature == f"sha256={expected}"

# Usage
payload = request.get_data()
signature = request.headers.get("X-ACP-Signature", "")

if verify_signature(payload, signature, "your-secret"):
    # Process webhook
    data = request.get_json()
else:
    # Reject invalid signature
    return "Invalid signature", 401
```

#### Node.js

```javascript
const crypto = require('crypto');

function verifySignature(payload, signature, secret) {
    const expected = crypto
        .createHmac('sha256', secret)
        .update(payload)
        .digest('hex');
    return signature === `sha256=${expected}`;
}

// Express.js example
app.post('/webhook', (req, res) => {
    const payload = JSON.stringify(req.body);
    const signature = req.headers['x-acp-signature'];

    if (!verifySignature(payload, signature, process.env.WEBHOOK_SECRET)) {
        return res.status(401).send('Invalid signature');
    }

    // Process webhook
    console.log('Event:', req.body.event_type);
    res.status(200).send('OK');
});
```

#### Go

```go
import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "strings"
)

func VerifySignature(payload []byte, signature string, secret string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(payload)
    expected := hex.EncodeToString(mac.Sum(nil))
    return signature == "sha256="+expected
}
```

## Integration Examples

### Slack Integration

Configure a Slack webhook:

```yaml
endpoints:
  - name: "slack-alerts"
    enabled: true
    url: "${SLACK_WEBHOOK_URL}"
    secret_env: "SLACK_WEBHOOK_SECRET"
    events:
      - "approval.escalated"
      - "key.revoked"
      - "detection.triggered"
    format: "slack"
    slack_channel: "#ai-alerts"
```

The `slack` format automatically formats messages with appropriate colors and emojis.

### PagerDuty Integration

```yaml
endpoints:
  - name: "pagerduty"
    enabled: true
    url: "${PAGERDUTY_EVENTS_URL:-https://events.pagerduty.com/v2/enqueue}"
    secret_env: "PAGERDUTY_ROUTING_KEY"
    events:
      - "approval.escalated"
      - "key.revoked"
    format: "pagerduty"
```

### Generic SIEM Integration

```yaml
endpoints:
  - name: "siem"
    enabled: true
    url: "${SIEM_WEBHOOK_URL}"
    secret_env: "SIEM_WEBHOOK_SECRET"
    events:
      - "key.*"
      - "policy.*"
      - "detection.triggered"
    headers:
      Authorization: "Bearer ${SIEM_API_TOKEN}"
```

### Custom Webhook Receiver

```bash
# Simple Python receiver
python3 -c '
from http.server import HTTPServer, BaseHTTPRequestHandler
import json

class WebhookHandler(BaseHTTPRequestHandler):
    def do_POST(self):
        length = int(self.headers["Content-Length"])
        body = self.rfile.read(length)
        data = json.loads(body)
        print(f"Event: {data[\"event_type\"]}")
        print(f"Payload: {json.dumps(data[\"payload\"], indent=2)}")
        self.send_response(200)
        self.end_response()

HTTPServer(("0.0.0.0", 8080), WebhookHandler).serve_forever()
'
```

## Testing

### Manual Testing

```bash
# Trigger detection-related events
make validate-detections

# Trigger key lifecycle events
make key-gen ALIAS=webhook-test BUDGET=1.00
make key-revoke ALIAS=webhook-test

# Verify endpoint connectivity from host
curl -i "$WEBHOOK_URL"
```

### Integration Testing

```bash
# Start a test receiver (requires Python)
python3 -m http.server 8080 &

# Configure webhooks
export WEBHOOKS_ENABLED=true
export WEBHOOK_URL=http://localhost:8080/webhook

# Generate a key (triggers webhook)
make key-gen ALIAS=webhook-test BUDGET=1.00

# Revoke the key (triggers webhook)
make key-revoke ALIAS=<alias>
```

### Online Testing

Use [webhook.site](https://webhook.site) for quick testing:

1. Get a unique URL from webhook.site
2. Configure in `webhooks.yaml`:

```yaml
endpoints:
  - name: "test"
    url: "https://webhook.site/YOUR-UNIQUE-ID"
    events:
      - "*"
```

3. Trigger events and view them on webhook.site

## Best Practices

### Security

1. **Always verify signatures** - Prevents spoofing and replay attacks
2. **Use HTTPS endpoints** - Encrypts data in transit
3. **Rotate secrets regularly** - Reduces impact of leaked credentials
4. **Never log secrets** - Check logs before sharing

### Reliability

1. **Return 200 quickly** - Process asynchronously to avoid timeouts
2. **Handle duplicates** - Use `idempotency_key` for deduplication
3. **Implement retries** - Webhooks retry up to 3 times with backoff
4. **Log all events** - Keep an audit trail for debugging

### Performance

1. **Filter events** - Subscribe only to needed event types
2. **Batch processing** - Queue events and process in batches
3. **Async handling** - Don't block HTTP responses

## Troubleshooting

### Webhooks Not Sending

1. Check master switch: `echo $WEBHOOKS_ENABLED`
2. Verify endpoint is enabled in YAML
3. Check event subscription patterns
4. Test with `--dry-run` flag

### Signature Verification Failing

1. Ensure secret is correctly set
2. Verify you're signing the raw payload body
3. Check for encoding issues (use raw bytes)

### Timeouts

1. Increase `timeout_seconds` in settings
2. Process events asynchronously
3. Return 200 immediately and process in background

## Related Documentation

- [SIEM Integration Guide](SIEM_INTEGRATION.md) - SIEM integration methods
- [Detection Rules](DETECTION.md) - Detection rule configuration
- [Deployment Guide](../DEPLOYMENT.md) - Production deployment
