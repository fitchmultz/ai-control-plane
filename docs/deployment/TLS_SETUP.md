# TLS/HTTPS Setup Guide

This guide explains how to enable TLS/HTTPS for the AI Control Plane demo environment using a Caddy reverse proxy.

## Table of Contents

1. [Overview](#1-overview)
2. [Why TLS Matters](#2-why-tls-matters)
3. [Quick Start (Local Demo)](#3-quick-start-local-demo)
4. [Configuration Modes](#4-configuration-modes)
5. [Security Considerations](#5-security-considerations)
6. [OAuth Token Safety](#6-oauth-token-safety)
7. [Client Configuration](#7-client-configuration)
8. [Troubleshooting](#8-troubleshooting)
9. [Production Deployment](#9-production-deployment)
10. [Certificate Management](#10-certificate-management)

---

## 1. Overview

The AI Control Plane supports **optional** TLS/HTTPS through Caddy, a modern reverse proxy with automatic certificate management.

### Default Behavior (HTTP)

By default, the gateway runs on HTTP only:
- LiteLLM Gateway: `http://127.0.0.1:4000`
- Suitable for local development and testing
- No additional configuration required

### Optional TLS Mode

When enabled, Caddy provides:
- Automatic HTTPS with certificate management
- Security headers (HSTS, X-Frame-Options, etc.)
- OAuth token protection in transit
- Support for both self-signed (local) and Let's Encrypt (production) certificates

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Without TLS (Default)                     │
├─────────────────────────────────────────────────────────────┤
│  Client (Claude Code, Codex, etc.)                          │
│       │                                                      │
│       └──> http://localhost:4000 ──> LiteLLM Gateway        │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                      With TLS (Optional)                     │
├─────────────────────────────────────────────────────────────┤
│  Client (Claude Code, Codex, etc.)                          │
│       │                                                      │
│       └──> https://localhost ──> Caddy (port 443)           │
│                                  │                          │
│                                  └──> LiteLLM (port 4000)   │
│                                      (localhost only)        │
└─────────────────────────────────────────────────────────────┘
```

---

## 2. Why TLS Matters

### Security Benefits

1. **OAuth Token Protection**: When using subscription mode (e.g., Claude Code with OAuth login), OAuth tokens are forwarded to upstream providers. TLS encrypts these tokens in transit.

2. **Data Integrity**: Prevents man-in-the-middle attacks that could modify API requests or responses.

3. **Compliance**: Many enterprise security policies require TLS for any production deployment.

### When TLS is Required

| Scenario | TLS Required | Reason |
|----------|--------------|--------|
| Local development on single machine | No | Traffic doesn't leave the host |
| Remote Docker host on trusted network | Recommended | Tokens traverse network |
| Production deployment | Yes | Security best practice |
| Public internet exposure | Yes | Required for security |

---

## 3. Quick Start (Local Demo)

### Prerequisites

- Docker and Docker Compose installed
- Base HTTP mode working (`make up` succeeds)
- Ports 80 and 443 available (or modify docker-compose.tls.yml)

### Step 1: Enable TLS Mode

Start services with TLS using the Makefile target:

```bash
make up-tls
```

This command:
- Starts Caddy reverse proxy on ports 80 and 443
- Configures LiteLLM to run behind a proxy
- Generates self-signed certificates automatically

### Step 2: Verify HTTPS is Working

```bash
# Test the HTTPS endpoint (ignoring self-signed cert warnings)
curl -k https://localhost/health

# Expected output:
# Caddy Reverse Proxy - Healthy
```

### Step 3: Trust Self-Signed Certificate (Optional)

For a better user experience, trust the self-signed certificate:

#### macOS

```bash
# Export the certificate from Caddy
docker exec ai-control-plane-caddy \
  cat /data/caddy/pki/authorities/local/root.crt > /tmp/caddy-root.crt

# Trust the certificate (requires sudo)
sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain /tmp/caddy-root.crt
```

#### Linux (Ubuntu/Debian)

```bash
# Export the certificate
docker exec ai-control-plane-caddy \
  cat /data/caddy/certs/local.crt > /tmp/caddy-local.crt

# Install to system trust store
sudo cp /tmp/caddy-local.crt /usr/local/share/ca-certificates/caddy-local.crt
sudo update-ca-certificates
```

#### Windows

```bash
# Export the certificate
docker exec ai-control-plane-caddy \
  cat /data/caddy/certs/local.crt > c:\temp\caddy-local.crt

# Import via Certificate Manager (certmgr.msc)
# Place in "Trusted Root Certification Authorities"
```

### Step 4: Update Client Configuration

After trusting the certificate, update your AI tool configuration:

```bash
# Claude Code configuration
export ANTHROPIC_BASE_URL="https://localhost"

# Verify connection
claude-code --version
```

See [Client Configuration](#7-client-configuration) for detailed tool setup.

---

## 4. Configuration Modes

### Mode 1: Self-Signed Certificates (Default)

**Use Case:** Local development, demo environments, internal testing

**Configuration:**
- Uses Caddy's internal CA for automatic certificate generation
- No external dependencies
- Certificates are trusted only by your machine (unless explicitly trusted)

**Environment Variables (.env):**
```bash
CADDY_ACME_CA=internal
LITELLM_PUBLIC_URL=https://localhost
```

**Caddyfile:** `demo/config/caddy/Caddyfile.dev`

### Mode 2: Let's Encrypt Certificates

**Use Case:** Production deployments, public-facing gateways

**Configuration:**
- Automatic certificate provisioning via Let's Encrypt ACME
- Requires valid domain name with DNS configured
- Requires public HTTP(S) access on port 80 (for ACME challenge)

**Prerequisites:**
- Domain name (e.g., `gateway.example.com`) pointing to your server
- Ports 80 and 443 accessible from the internet
- Valid email address for certificate expiration notices

**Environment Variables (.env):**
```bash
CADDY_ACME_CA=letsencrypt
CADDY_DOMAIN=gateway.example.com
CADDY_EMAIL=admin@example.com
LITELLM_PUBLIC_URL=https://gateway.example.com
```

**Caddyfile:** `demo/config/caddy/Caddyfile.prod` uses environment variables (`{$CADDY_DOMAIN}`, `{$CADDY_EMAIL}`) and does not require editing.

---

## 5. Security Considerations

### OAuth Token Safety

**CRITICAL**: When `forward_client_headers_to_llm_api: true` is set in `litellm.yaml`, OAuth tokens are forwarded to upstream providers.

**Caddy's Default Behavior (Safe):**
- Caddy does NOT log HTTP headers by default
- The Authorization header containing OAuth tokens is NOT logged
- This is the intended behavior - DO NOT add `log_headers` directive

**Verification:**
```bash
# Check Caddy logs for Authorization headers (should be empty)
docker compose logs caddy | grep -i authorization

# Expected: No matches (Authorization headers are not logged)
```

### Security Headers

Caddy automatically adds security headers:

| Header | Purpose | Value |
|--------|---------|-------|
| Strict-Transport-Security | Enforce HTTPS | max-age=31536000 |
| X-Content-Type-Options | Prevent MIME sniffing | nosniff |
| X-Frame-Options | Prevent clickjacking | DENY |
| X-XSS-Protection | XSS protection | 1; mode=block |

### What NOT to Change

**DO NOT** modify Caddyfile to add:
- `log_headers` directive (would expose OAuth tokens)
- Custom logging that includes headers
- Any directive that logs the Authorization header

**DO NOT** disable:
- TLS (keep HTTPS only)
- Security headers
- Caddy's default logging behavior

---

## 6. OAuth Token Safety

### Understanding the Risk

When using subscription mode (e.g., Claude Code with OAuth login):
1. Client sends OAuth token to gateway
2. Gateway forwards token to upstream provider (Anthropic, OpenAI, etc.)
3. Provider validates token and processes request

**Risk**: If logs capture the Authorization header, OAuth tokens are exposed in log files.

### Caddy Configuration (Safe by Default)

The provided Caddyfile configuration is safe:

```caddyfile
log {
    output file /var/log/caddy/access.log
    format json
    # NO log_headers directive - headers are not logged
}
```

### Verification Commands

**Check for token leakage in Caddy logs:**
```bash
# Should return no matches
docker compose logs caddy | grep -i "authorization"

# Should return no matches
docker compose logs caddy | grep -i "bearer"
```

**Check for token leakage in LiteLLM logs:**
```bash
# Should return no matches (unless explicitly configured)
docker compose logs litellm | grep -i "authorization"
```

**Automated secrets audit:**
```bash
# Run the tracked-file audit
make secrets-audit

# Difference from tls-verify:
# - tls-verify: checks live container logs only
# - secrets-audit: applies docs/policy/SECRET_SCAN_POLICY.json to tracked repository files
```

### What to Do If Tokens Are Found in Logs

1. **Immediate Action**: Stop services and secure log files
2. **Run Secrets Audit**: Use `make secrets-audit` to check tracked repository content before sharing or committing it
3. **Rotate Tokens**: Revoke compromised OAuth tokens through provider portal
4. **Fix Configuration**: Remove any `log_headers` or similar directives
5. **Clean Logs**: Remove or redact log files containing tokens
6. **Verify**: Run verification commands to confirm no future leakage

### Secrets Audit vs tls-verify

| Command | Scope | Use Case |
|---------|-------|----------|
| `make tls-verify` | Live container logs (Caddy, LiteLLM) | Runtime verification |
| `make secrets-audit` | Tracked repository files matched by `docs/policy/SECRET_SCAN_POLICY.json` | Pre-commit and pre-sharing repository validation |

**When to use each:**
- Use `make tls-verify` during development to ensure OAuth tokens aren't being logged
- Use `make secrets-audit` before committing or sharing tracked repository content
- Both are included in `make lint` and `make ci` for automated checking

---

## 7. Client Configuration

### Using the Onboarding Script (Recommended)

The easiest way to configure clients for TLS is using the onboarding script with the `--tls` flag:

```bash
# Claude Code with TLS
make onboard TOOL=claude MODE=api-key TLS=1

# Codex with TLS
make onboard TOOL=codex MODE=api-key TLS=1

# OpenCode with TLS
make onboard TOOL=opencode MODE=gateway TLS=1

# For production with custom domain
make onboard TOOL=claude MODE=api-key HOST=gateway.example.com TLS=1
```

### Manual Configuration

If you prefer to configure manually:

#### Claude Code

**Without TLS (default):**
```bash
export ANTHROPIC_BASE_URL="http://localhost:4000"
export ANTHROPIC_API_KEY="sk-litellm-..."
```

**With TLS (local, self-signed):**
```bash
export ANTHROPIC_BASE_URL="https://localhost"
export ANTHROPIC_API_KEY="sk-litellm-..."
# Or use NODE_EXTRA_CA_CERTS if certificate is not trusted
export NODE_EXTRA_CA_CERTS=/path/to/caddy-root.crt
```

**With TLS (production, Let's Encrypt):**
```bash
export ANTHROPIC_BASE_URL="https://gateway.example.com"
export ANTHROPIC_API_KEY="sk-litellm-..."
```

#### Codex CLI

**Without TLS:**
```bash
export OPENAI_BASE_URL="http://localhost:4000"
export OPENAI_API_KEY="sk-litellm-..."
```

**With TLS:**
```bash
export OPENAI_BASE_URL="https://localhost"
export OPENAI_API_KEY="sk-litellm-..."
```

#### OpenCode

Update `~/.opencode/config.json`:

```json
{
  "apiBaseUrl": "https://localhost",
  "apiKey": "sk-litellm-..."
}
```

### Cursor

Update Cursor settings to use HTTPS endpoint.

---

## 8. Troubleshooting

### Issue: Certificate Trust Warnings

**Symptom**: Browser or client shows certificate warning

**Self-signed certificates (expected):**
- This is normal for local development
- Trust the certificate manually (see Quick Start)
- Or use `curl -k` to ignore warnings

**Let's Encrypt certificates:**
- Verify domain DNS is correct
- Check port 80 is accessible from internet
- Check Caddy logs for ACME errors: `docker compose logs caddy`

### Issue: Port 80 or 443 Already in Use

**Symptom**: Caddy container fails to start

**Solution 1**: Stop conflicting service:
```bash
# Check what's using the ports
sudo lsof -i :80
sudo lsof -i :443
```

**Solution 2**: Modify ports in `docker-compose.tls.yml`:
```yaml
ports:
  - "8080:80"   # Use alternative HTTP port
  - "8443:443"  # Use alternative HTTPS port
```

Then update client configuration to use `https://localhost:8443`

### Issue: OAuth Tokens in Logs

**Symptom**: Authorization headers found in logs

**Immediate Action**:
1. Stop services: `make down-tls`
2. Check Caddyfile for `log_headers` directive (remove it)
3. Check LiteLLM configuration for excessive logging
4. Rotate compromised OAuth tokens
5. Remove or redact log files

**Verification**:
```bash
# Should return no matches after fix
docker compose logs caddy | grep -i authorization
```

### Issue: ACME Challenge Fails (Let's Encrypt)

**Symptom**: Caddy logs show ACME challenge timeout

**Diagnosis**:
```bash
# Check Caddy logs
docker compose logs caddy | grep -i acme

# Check port 80 accessibility
curl http://your-domain.com/.well-known/acme-challenge/test
```

**Solutions**:
1. Verify DNS A record points to correct IP
2. Verify port 80 is open and accessible from internet
3. Check firewall rules: `sudo ufw status`
4. Use Let's Encrypt staging for testing:
   ```caddyfile
   acme_ca https://acme-staging-v02.api.letsencrypt.org/directory
   ```

### Issue: Health Check Fails

**Diagnosis**:
```bash
# Check Caddy is running
docker compose ps caddy

# Check Caddy configuration is valid
docker exec ai-control-plane-caddy caddy validate --config /etc/caddy/Caddyfile

# Test health endpoint directly
curl -k https://localhost/health
```

**Solution**: Ensure `LITELLM_PUBLIC_URL` is set correctly in `.env`

---

## 9. Production Deployment

### Pre-Deployment Checklist

- [ ] Domain name configured with DNS A record
- [ ] Ports 80 and 443 accessible from internet
- [ ] Firewall rules allow HTTP/HTTPS traffic
- [ ] Caddyfile.prod updated with correct domain
- [ ] Let's Encrypt email address configured
- [ ] `CADDY_ACME_CA=letsencrypt` in `.env`
- [ ] `LITELLM_PUBLIC_URL` set to production URL
- [ ] Self-signed Caddyfile.dev removed or disabled
- [ ] OAuth token safety verified (no header logging)
- [ ] Backup strategy configured for Caddy data volume

### Production Caddyfile

The production Caddyfile (`demo/config/caddy/Caddyfile.prod`) uses environment variables
and does not require editing:

```caddyfile
{
    email {$CADDY_EMAIL}
}

{$CADDY_DOMAIN} {
    reverse_proxy litellm:4000
    ...
}
```

Simply set `CADDY_DOMAIN` and `CADDY_EMAIL` in your `.env` file.

### Update Docker Compose Override

Modify `demo/docker-compose.tls.yml` to use production Caddyfile:

```yaml
volumes:
  - ./config/caddy/Caddyfile.prod:/etc/caddy/Caddyfile:ro
```

### Start Production Services

```bash
# Set production environment
export CADDY_ACME_CA=letsencrypt
export CADDY_DOMAIN=gateway.example.com
export CADDY_EMAIL=admin@example.com
export LITELLM_PUBLIC_URL=https://gateway.example.com

# Start services
make up-tls
```

### Validating TLS Configuration

After enabling TLS, run the production smoke tests to validate the deployment:

```bash
# Validate local TLS deployment
make prod-smoke-local-tls

# Or validate the active runtime configuration directly
make prod-smoke
```

The smoke tests verify:
- Gateway health is reachable over the configured TLS endpoint
- Authorized `/v1/models` access works with `LITELLM_MASTER_KEY`
- Database/readiness requirements are healthy

For production handoff procedures, see the [Production Handoff Runbook](./PRODUCTION_HANDOFF_RUNBOOK.md).

---

## 10. Certificate Management

### Certificate Storage

Caddy stores certificates in Docker named volumes:
- `caddy_data`: Certificates and private keys
- `caddy_config`: Caddy configuration state

**Location on host** (for inspection/backup):
```bash
# Find volume location
docker volume inspect ai_control_plane_caddy_data

# Inspect certificates
docker run --rm -v ai_control_plane_caddy_data:/data \
  alpine ls -la /data/caddy/certs/
```

### Backup Certificates

```bash
# Backup Caddy data volume
docker run --rm -v ai_control_plane_caddy_data:/data \
  -v $(pwd)/backups:/backup \
  alpine tar czf /backup/caddy-data-backup.tar.gz -C /data .
```

### Certificate Renewal

**Let's Encrypt**: Automatic renewal happens in background (typically 30 days before expiry)

**Self-signed**: No renewal needed, but regeneration is possible:
```bash
# Remove existing certificates
docker run --rm -v ai_control_plane_caddy_data:/data \
  alpine rm -rf /data/caddy/pki

# Restart Caddy to regenerate
docker compose restart caddy
```

### Certificate Expiration Monitoring

**Check Let's Encrypt certificate expiry:**
```bash
echo | openssl s_client -connect gateway.example.com:443 2>/dev/null | \
  openssl x509 -noout -dates
```

**Monitor Caddy logs for renewal events:**
```bash
docker compose logs caddy | grep -i renew
```

---

## TLS on Kubernetes

For Kubernetes deployments, TLS is handled at the Ingress layer instead of Caddy.

### Overview

When deploying on Kubernetes:
- **Caddy is NOT used** - Ingress controller handles TLS termination
- **cert-manager recommended** - For automatic Let's Encrypt certificates
- **Pre-existing secrets supported** - For custom certificates

### Quick Start

See [KUBERNETES_HELM.md](./KUBERNETES_HELM.md) for complete Kubernetes TLS setup. Key configuration:

```yaml
ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
  hosts:
    - host: ai-control-plane.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: ai-control-plane-tls
      hosts:
        - ai-control-plane.example.com
```

### OAuth Token Safety on Kubernetes

**IMPORTANT**: Ensure your Ingress controller does NOT log Authorization headers:

```yaml
# nginx ingress - don't log headers
nginx.ingress.kubernetes.io/configuration-snippet: |
  # Authorization headers are NOT logged by default
  # Verify with: kubectl logs -n ingress-nginx deployment/ingress-nginx-controller | grep -i authorization
```

Most modern Ingress controllers (nginx, traefik, Ambassador) do NOT log headers by default. Always verify after deployment.

---

## Additional Resources

- [Caddy Documentation](https://caddyserver.com/docs/)
- [Caddy Automatic HTTPS](https://caddyserver.com/docs/automatic-https)
- [Let's Encrypt](https://letsencrypt.org/)
- [LiteLLM Behind Proxy](https://docs.litellm.ai/docs/proxy/configs)
- [Kubernetes TLS Setup](./KUBERNETES_HELM.md)

---

**Security Note**: Always verify that OAuth tokens are NOT being logged. Run verification commands after any configuration changes.
