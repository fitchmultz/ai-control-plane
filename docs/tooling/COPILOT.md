# GitHub Copilot Configuration for AI Control Plane

## Overview

GitHub Copilot is an AI-powered coding assistant developed by GitHub and OpenAI. Unlike other AI coding tools in this project (Cursor, OpenCode, Codex CLI, Claude Code), **GitHub Copilot does NOT support custom OpenAI-compatible base URLs**. This fundamental architectural difference requires a different integration approach.

### Decision Record (February 2026)

- **Decision:** Include VS Code/Copilot guidance now as a supported supplemental path.
- **Rationale:** Operators asked for practical rollout guidance even though Copilot cannot be fully gateway-enforced.
- **Boundary:** Copilot support in this repo is **proxy/telemetry visibility guidance**, not full LiteLLM inline enforcement.
- **Primary governed developer path remains:** Cursor, Codex, Claude Code, and OpenCode through LiteLLM.

### Integration Approach: HTTP Proxy Mode

The AI Control Plane integrates with GitHub Copilot by acting as an **HTTP forward proxy**. This provides network-level visibility into Copilot traffic while acknowledging the limitations imposed by Copilot's direct GitHub API communication.

### What This Integration Provides

| Capability | Status | Notes |
|------------|--------|-------|
| Network Visibility | ✓ Available | Traffic passes through gateway; connection logging |
| Connection Logging | ✓ Available | Who, when, bytes transferred |
| Request/Response Inspection | ✗ Not Available | TLS-encrypted content cannot be inspected |
| Content Filtering | ✗ Not Available | Cannot inspect or modify request content |
| Budget Enforcement | ✗ Not Available | Copilot uses GitHub billing, not LiteLLM budgets |
| Audit Trail | Partial | Connection metadata only (no prompt/response content) |

### Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Client Machine                               │
│                                                                     │
│  ┌─────────────────┐     HTTP Proxy Settings      ┌──────────────┐ │
│  │   VS Code /     │ ────────────────────────────▶ │  System or   │ │
│  │   JetBrains /   │    http://GATEWAY_HOST:3128  │  IDE Proxy   │ │
│  │   Visual Studio │                              │  Settings    │ │
│  └─────────────────┘                              └──────────────┘ │
└────────────────────────────────┬────────────────────────────────────┘
                                 │
                                 │ Forward Proxy Request
                                 ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      AI Control Plane Gateway                       │
│                                                                     │
│  ┌───────────────────────────────────────────────────────────────┐ │
│  │  Forward Proxy Service (Port 3128)                            │ │
│  │  • Accepts CONNECT requests for TLS tunneling                 │ │
│  │  • Logs connection metadata (source IP, destination, bytes)   │ │
│  │  • Forwards encrypted traffic to api.github.com               │ │
│  └───────────────────────────────────────────────────────────────┘ │
│                                                                     │
│  ┌───────────────────────────────────────────────────────────────┐ │
│  │  OTEL Collector (Port 4317)                                   │ │
│  │  • Optional telemetry reception from IDE extensions           │ │
│  │  • Provides additional visibility when proxy not feasible     │ │
│  └───────────────────────────────────────────────────────────────┘ │ │
└────────────────────────────────┬────────────────────────────────────┘
                                 │ Encrypted TLS
                                 │ (Cannot inspect content)
                                 ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      GitHub Copilot API                             │
│                      api.github.com/copilot/*                       │
│                                                                     │
│  • Copilot completions API                                          │
│ • Copilot chat API                                                  │
│  • GitHub billing and authentication                                │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Configuration Modes](#configuration-modes)
3. [VS Code Configuration](#vs-code-configuration)
4. [JetBrains IDE Configuration](#jetbrains-ide-configuration)
5. [Visual Studio Configuration](#visual-studio-configuration)
6. [OTEL Telemetry Mode](#otel-telemetry-mode)
7. [Verification](#verification)
8. [Troubleshooting](#troubleshooting)
9. [Enterprise Deployment Considerations](#enterprise-deployment-considerations)
10. [Related Documentation](#related-documentation)

---

## Prerequisites

### Required Components

| Component | Purpose |
|-----------|---------|
| GitHub Copilot subscription | Active Copilot subscription (Individual, Business, or Enterprise) |
| Supported IDE | VS Code, JetBrains IDE, or Visual Studio |
| AI Control Plane Gateway | Running gateway with proxy capability |

### Quick Setup with Onboarding Script

The easiest way to get Copilot configuration guidance is using the onboarding script:

```bash
# Proxy mode (recommended for visibility)
make onboard TOOL=copilot MODE=proxy

# With verification
make onboard TOOL=copilot MODE=proxy VERIFY=1

# For remote Docker host
make onboard TOOL=copilot MODE=proxy HOST=GATEWAY_HOST

# OTEL telemetry mode (when proxy is not feasible)
make onboard TOOL=copilot MODE=otel
```

### Manual Configuration

If you prefer to configure manually, you'll need:

1. Gateway host address (e.g., `192.168.1.122` or `127.0.0.1`)
2. Proxy port (default: `3128` if gateway proxy is configured)
3. OTEL endpoint (for telemetry mode): `http://GATEWAY_HOST:4317`

---

## Configuration Modes

### Mode A: HTTP Proxy (Recommended)

In proxy mode, the AI Control Plane acts as a forward proxy for all Copilot traffic. This provides connection-level visibility without requiring changes to Copilot's authentication or API endpoints.

**Benefits:**
- Network-level logging of Copilot connections
- Egress control enforcement point
- No changes to Copilot authentication required

**Limitations:**
- Cannot inspect encrypted request/response content
- No per-request budget controls
- Requires proxy configuration in IDE or system settings

### Mode B: OTEL Telemetry (Bypass Scenario)

When proxy configuration is not feasible, OTEL telemetry mode provides visibility through telemetry export from IDE extensions that support it.

**Benefits:**
- No network topology changes required
- Can capture higher-level metrics (if IDE extension supports OTEL)

**Limitations:**
- Most Copilot telemetry is internal to GitHub
- IDE extension must explicitly support OTEL export
- Limited to metrics that Copilot chooses to expose

**Note:** GitHub Copilot's primary telemetry is sent directly to GitHub and cannot be redirected. OTEL mode only works if your IDE or an extension specifically supports exporting telemetry to a custom endpoint.

---

## VS Code Configuration

### Method 1: VS Code Settings (Application-Level Proxy)

1. Open VS Code Settings (Cmd/Ctrl + ,)
2. Search for "proxy"
3. Configure the following settings:

| Setting | Value |
|---------|-------|
| `http.proxy` | `http://GATEWAY_HOST:3128` |
| `http.proxySupport` | `override` |

### Method 2: System Proxy (OS-Level)

Configure your operating system's proxy settings to route all traffic through the gateway:

**macOS:**
```bash
networksetup -setwebproxy "Wi-Fi" GATEWAY_HOST 3128
networksetup -setsecurewebproxy "Wi-Fi" GATEWAY_HOST 3128
```

**Windows (PowerShell Admin):**
```powershell
netsh winhttp set proxy GATEWAY_HOST:3128
```

**Linux (GNOME):**
```bash
gsettings set org.gnome.system.proxy mode 'manual'
gsettings set org.gnome.system.proxy.http host 'GATEWAY_HOST'
gsettings set org.gnome.system.proxy.http port 3128
```

### Method 3: Environment Variables

Set environment variables before launching VS Code:

```bash
export HTTP_PROXY="http://GATEWAY_HOST:3128"
export HTTPS_PROXY="http://GATEWAY_HOST:3128"
code
```

### Verification

After configuration:

1. Open VS Code Developer Tools (Help → Toggle Developer Tools)
2. Open the Network tab
3. Trigger a Copilot completion (type in a code file)
4. Check that requests are routed through the proxy

---

## JetBrains IDE Configuration

### IntelliJ IDEA, PyCharm, WebStorm, etc.

1. Open **Settings** (File → Settings or IntelliJ IDEA → Preferences)
2. Navigate to **Appearance & Behavior → System Settings → HTTP Proxy**
3. Select **Manual proxy configuration**
4. Configure:

| Setting | Value |
|---------|-------|
| HTTP Proxy | `GATEWAY_HOST` |
| Port | `3128` |

5. Click **Check connection** to verify (test URL: `https://api.github.com`)
6. Click **OK** to save

### Per-Project Configuration

If you only want to proxy Copilot traffic for specific projects:

1. Open **Settings** → **Appearance & Behavior** → **System Settings** → **HTTP Proxy**
2. Select **No proxy** for the global setting
3. Use the IDE's VM options for specific proxy routing (advanced)

### GitHub Copilot Plugin Settings

The GitHub Copilot plugin for JetBrains respects the IDE's proxy settings. No additional configuration is required within the Copilot plugin itself.

---

## Visual Studio Configuration

### System Proxy Settings

Visual Studio uses the Windows system proxy settings:

1. Open **Windows Settings** → **Network & Internet** → **Proxy**
2. Under **Manual proxy setup**, configure:

| Setting | Value |
|---------|-------|
| Address | `GATEWAY_HOST` |
| Port | `3128` |

3. Click **Save**

### Visual Studio devenv.exe.config

For more granular control, you can configure proxy settings in Visual Studio's configuration file:

1. Locate `devenv.exe.config` in your Visual Studio installation directory
   (typically `C:\Program Files\Microsoft Visual Studio\2022\Enterprise\Common7\IDE\`)

2. Add or modify the `system.net` section:

```xml
<system.net>
  <defaultProxy>
    <proxy proxyaddress="http://GATEWAY_HOST:3128" bypassonlocal="true" />
  </defaultProxy>
</system.net>
```

3. Restart Visual Studio

---

## OTEL Telemetry Mode

### When to Use OTEL Mode

OTEL telemetry mode is appropriate when:

- You cannot configure HTTP proxy (corporate restrictions)
- Proxy configuration causes Copilot performance issues
- You only need high-level metrics, not connection logging
- Your IDE or an extension supports custom OTEL endpoints

### Configuration

Set the following environment variables before launching your IDE:

```bash
export OTEL_EXPORTER_OTLP_ENDPOINT="http://GATEWAY_HOST:4317"
export OTEL_EXPORTER_OTLP_PROTOCOL="grpc"
export OTEL_SERVICE_NAME="copilot-vscode"  # or copilot-jetbrains, etc.
export OTEL_RESOURCE_ATTRIBUTES="service.version=1.0.0,deployment.environment=production"
```

### Limitations

**Important:** GitHub Copilot does NOT natively support custom OTEL endpoints. OTEL mode will only provide telemetry if:

1. Your IDE has a plugin that supports OTEL export
2. You've installed a Copilot analytics extension that supports custom endpoints
3. You're using a corporate-managed IDE with pre-configured telemetry

For most users, **HTTP Proxy mode is the only effective integration path**.

---

## Verification

### Step 1: Verify Proxy Connectivity

Test that the proxy is reachable:

```bash
# Test proxy connectivity
curl -x http://GATEWAY_HOST:3128 -I https://api.github.com

# Expected: HTTP/1.1 200 OK (or 301/302 redirect)
```

### Step 2: Verify Gateway Logs

After configuring your IDE and triggering a Copilot completion:

```bash
# Check gateway logs for proxy connections
make logs

# Or check specific proxy service logs if available
docker compose logs --tail=50 proxy
```

Look for:
- Connection entries to `api.github.com`
- TLS tunnel establishment (`CONNECT api.github.com:443`)
- Source IP and timestamp information

### Step 3: Verify Copilot Functionality

Ensure Copilot continues to work correctly:

1. Open a code file in your IDE
2. Type a comment like `// function to reverse a string`
3. Wait for Copilot suggestion (gray text)
4. Press Tab to accept

If suggestions appear, the proxy is correctly forwarding traffic.

### Step 4: Check Connection Logging

If your gateway provides connection logs:

```bash
# Example: Query proxy access logs (implementation-specific)
docker compose logs proxy | grep "api.github.com"
```

Expected output should show:
- Timestamp
- Source IP (your client machine)
- Destination (`api.github.com:443`)
- Bytes transferred (if available)

---

## Troubleshooting

### Copilot Stops Working After Proxy Configuration

**Symptom:** Copilot was working, but after configuring proxy, it stops providing suggestions.

**Diagnosis:**

1. Test proxy connectivity:
```bash
curl -x http://GATEWAY_HOST:3128 https://api.github.com/copilot/v1/chat/completions -v
```

2. Check if proxy requires authentication (most don't for forward proxy)

3. Verify firewall rules allow gateway to reach `api.github.com:443`

**Solutions:**

- Ensure gateway proxy service is running: `make ps`
- Check gateway can reach GitHub: `docker exec <gateway> curl -I https://api.github.com`
- If using a corporate proxy, ensure it supports CONNECT method for TLS tunneling

### Proxy Configuration Not Taking Effect

**Symptom:** VS Code/JetBrains shows no proxy traffic, Copilot connects directly.

**Diagnosis:**

Check current proxy settings:

```bash
# VS Code: Check settings.json
cat ~/.config/Code/User/settings.json | grep -i proxy

# System: Check environment variables
env | grep -i proxy
```

**Solutions:**

- Fully restart IDE after changing proxy settings
- Check for conflicting proxy settings (system vs IDE vs environment)
- On macOS, ensure proxy is set for the correct network interface
- Try launching IDE from terminal with proxy env vars set

### Certificate Errors

**Symptom:** TLS/SSL certificate errors when Copilot tries to connect.

**Cause:** Some proxy configurations intercept TLS (MITM), which breaks Copilot's certificate pinning.

**Solutions:**

- Ensure proxy uses `CONNECT` tunneling (not TLS interception)
- If your enterprise requires TLS inspection, Copilot may not function correctly
- Consider OTEL mode as an alternative (with reduced visibility)

### No Logs Appearing in Gateway

**Symptom:** Copilot works, but gateway shows no connection logs.

**Diagnosis:**

1. Verify proxy is actually being used (check IDE network settings)
2. Check if Copilot is using cached authentication
3. Test with a fresh IDE window/restart

**Solutions:**

- Clear Copilot cache (varies by IDE)
- Sign out and back into Copilot
- Check if IDE has multiple network paths (bypassing proxy)

### Performance Degradation

**Symptom:** Copilot suggestions are slower through the proxy.

**Causes:**

- Additional network hop through gateway
- Proxy logging overhead
- Geographic distance between client and gateway

**Mitigation:**

- Ensure gateway has good connectivity to GitHub
- Consider running gateway closer to clients (same network segment)
- Reduce logging verbosity if configurable

---

## Enterprise Deployment Considerations

### Governance and Compliance

Unlike other AI tools in this project, GitHub Copilot has **limited gateway integration** due to its direct GitHub API dependency. Enterprises should consider:

| Control | Implementation | Effectiveness |
|---------|---------------|---------------|
| **Network Visibility** | HTTP Proxy | Partial - connection metadata only |
| **Egress Filtering** | Firewall rules blocking direct GitHub access | Requires proxy for all GitHub access |
| **Audit Trail** | Proxy logs + GitHub Audit Log | GitHub Audit Log provides more detail |
| **Data Residency** | Copilot Enterprise with data residency | GitHub-side configuration |
| **Content Policy** | GitHub Copilot Enterprise policies | Managed in GitHub Admin settings |

### GitHub Copilot Enterprise Features

For comprehensive governance, consider GitHub Copilot Enterprise which provides:

- **Audit logging** via GitHub Audit Log API
- **Policy enforcement** (e.g., block suggestions matching public code)
- **Data residency** options
- **Organization-wide configuration**

The AI Control Plane proxy integration is **complementary** to these GitHub-native controls, not a replacement.

### Integration Architecture

For enterprise deployments, the recommended architecture is:

```
┌─────────────────────────────────────────────────────────────────────┐
│                      Enterprise Network                             │
│                                                                     │
│  ┌──────────────┐      ┌──────────────┐      ┌──────────────┐      │
│  │   Developer  │─────▶│   AI Control │─────▶│   Internet   │      │
│  │   Workstation│      │   Plane GW   │      │   (GitHub)   │      │
│  └──────────────┘      └──────────────┘      └──────────────┘      │
│                              │                                      │
│                              ▼                                      │
│                       ┌──────────────┐                              │
│                       │   SIEM/Logs  │                              │
│                       └──────────────┘                              │
└─────────────────────────────────────────────────────────────────────┘
```

Key points:
- Block direct HTTPS to `api.github.com` at the corporate firewall
- Force all GitHub API traffic through AI Control Plane gateway
- Gateway acts as the single egress point for audit logging
- Combine with GitHub Copilot Enterprise for policy enforcement

### Monitoring and Alerting

Set up monitoring for:

1. **Unusual connection patterns** (high volume, off-hours)
2. **Proxy bypass attempts** (direct GitHub connections)
3. **Failed authentication** (expired Copilot licenses)

Example detection rule (conceptual):

```yaml
# Alert on direct GitHub connections bypassing proxy
detection:
  selection:
    destination:
      - api.github.com
      - copilot-proxy.githubusercontent.com
    source|not:
      - ai-control-plane-gateway  # Only gateway should connect
  condition: selection
```

---

## Related Documentation

| Document | Purpose |
|----------|---------|
| [CURSOR.md](./CURSOR.md) | Cursor IDE with full gateway routing |
| [OPENCODE.md](./OPENCODE.md) | OpenCode with gateway and bypass modes |
| [CODEX.md](./CODEX.md) | Codex CLI with multiple authentication modes |
| [DEPLOYMENT.md](../DEPLOYMENT.md) | Network setup and deployment modes |
| [GitHub Copilot Documentation](https://docs.github.com/copilot) | Official GitHub Copilot docs |
| [GitHub Copilot Network Settings](https://docs.github.com/copilot/configuring-github-copilot/configuring-network-settings-for-github-copilot) | Official network configuration guide |

---

## Additional Resources

- **GitHub Copilot Enterprise:** <https://github.com/enterprise/copilot>
- **GitHub Audit Log API:** <https://docs.github.com/rest/enterprise-admin/audit-log>
- **VS Code Copilot Settings:** <https://code.visualstudio.com/docs/copilot/reference/copilot-settings>
- **JetBrains Copilot Plugin:** <https://plugins.jetbrains.com/plugin/17718-github-copilot>

---

## Summary

GitHub Copilot integration with the AI Control Plane is **different** from other AI tools due to Copilot's direct GitHub API dependency:

1. **No Custom Base URL**: Copilot cannot be routed through LiteLLM like Cursor/OpenCode
2. **Proxy Mode Only**: HTTP forward proxy provides connection-level visibility
3. **Limited Enforcement**: Cannot inspect content or enforce budgets (GitHub billing)
4. **Enterprise Complement**: Use alongside GitHub Copilot Enterprise for comprehensive governance

For full gateway enforcement with content inspection and budget controls, consider **Cursor** or **Claude Code** as alternatives.
