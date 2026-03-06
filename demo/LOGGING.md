# Logging Guide

This document describes how logging is configured in the AI Control Plane demo environment and how to work with logs effectively.

## Overview

The demo environment uses Docker's built-in log rotation to prevent indefinite log growth and manage disk usage. Logs are produced by two primary services:

- **LiteLLM Gateway**: Application logs for API requests, responses, and system events
- **PostgreSQL**: Database logs for connections, queries, and system messages

## Log Rotation Configuration

### Docker Log Driver Settings

Both services use Docker's `json-file` log driver with automatic rotation configured in `docker-compose.yml`:

```yaml
logging:
  driver: "json-file"
  options:
    max-size: "10m"
    max-file: "3"
```

**Configuration details:**
- **max-size: "10m"** - Each log file is capped at 10MB
- **max-file: "3"** - Docker keeps 3 rotated log files per service
- **Total retention per service**: ~30MB (10MB × 3 files)

### Retention Policy

- **Application logs**: ~30 days (assuming ~1MB/day typical log volume)
- **Total disk usage**: ~60MB maximum (30MB per service × 2 services)

Docker automatically rotates logs when a file reaches 10MB:
- Current log file: `<container-id>-json.log`
- Rotated files: `<container-id>-json.log.1`, `<container-id>-json.log.2`

When rotation occurs, the oldest log file is removed.

## Log Locations

### Container Logs (Docker JSON Logs)

Docker captures stdout/stderr from containers and stores them as JSON files:

```bash
# Docker's default location (Linux/WSL2)
~/.local/docker/containers/<container-id>/<container-id>-json.log
```

These logs are managed by Docker's log rotation configuration and do not require manual cleanup.

### Mounted Volume Logs

LiteLLM writes additional logs to a mounted volume:

```bash
demo/logs/           # Mounted to /var/log/litellm in container
```

The `logs/` directory is gitignored and contains any files written by LiteLLM to its configured log directory.

### Database Audit Logs

PostgreSQL audit logs are stored in the database and can be queried directly. See [DATABASE.md](../docs/DATABASE.md) for details.

## Viewing Logs

### Follow All Logs

```bash
# Using Make
make logs

# Direct docker-compose command
cd demo && docker compose logs -f
```

### View Specific Service Logs

```bash
# LiteLLM gateway logs
docker compose logs -f litellm

# PostgreSQL logs
docker compose logs -f postgres

# Last 100 lines of LiteLLM logs
docker compose logs --tail=100 litellm
```

### View Logs Since Specific Time

```bash
# Logs from last 5 minutes
docker compose logs --since 5m litellm

# Logs since a specific timestamp
docker compose logs --since "2024-01-15T10:00:00" litellm
```

### Direct File Access

For mounted volume logs:

```bash
# List log files in demo/logs/
ls -la demo/logs/

# View a specific log file
less demo/logs/<filename>
```

## Cleanup Commands

### Remove Mounted Log Files

```bash
# Log-files-only cleanup (preserves Docker volumes)
rm -rf demo/logs/*
```

**Note**: `make clean` is a full destructive cleanup (services + volumes + logs). Use manual `rm -rf demo/logs/*` when you only want to clear local log files.

### Full Cleanup (Including Volumes)

```bash
# Stops services and removes all volumes (including database data)
# In interactive shells: prompts for confirmation (Continue? [y/N])
make clean

# For scripts/automation - skip confirmation
make clean-force

# Equivalent manual command
cd demo && docker compose -f docker-compose.yml -f docker-compose.tls.yml down -v
```

**Warning**: `make clean` removes all Docker volumes, including the PostgreSQL database (`pgdata`) and TLS certificates (`ai_control_plane_caddy_data`, `ai_control_plane_caddy_config`). Always backup before running this command:

```bash
make db-backup
make clean  # prompts for confirmation; for scripts use: make clean-force
```

## Log Rotation Verification

### Verify Configuration

```bash
# Check that log rotation is configured
docker compose config | grep -A 5 logging
```

Expected output:
```yaml
logging:
  driver: json-file
  options:
    max-file: "3"
    max-size: 10m
```

### Check Current Log Sizes

```bash
# Check Docker log file sizes
du -sh ~/.local/docker/containers/*/*-json.log

# Check mounted log directory
du -sh demo/logs/
```

### Monitor Log Growth

```bash
# Watch log file sizes in real-time
watch -n 5 'du -sh ~/.local/docker/containers/*/*-json.log'

# Check log line counts
docker compose logs litellm | wc -l
docker compose logs postgres | wc -l
```

## Security Considerations

### Runtime Secret Redaction

**Default Protection**: Demo scenario scripts now automatically redact secrets in output.

By default, all scenario scripts mask sensitive values:
- Key previews (for config/export hints) are shown as `sk-abc123def...` (first 12 chars only)
- Full-key diagnostic lines are rendered as `sk-[REDACTED]`
- Bearer tokens are replaced with `[REDACTED]`
- Export commands show previews, not full values

**Example - Default (Safe) Output:**
```bash
$ ./scenario_1_api_path.sh --verbose
Step 2: Generate Test Virtual Key
  ✓ Key created: virtual-key-abc123... (alias: scenario-1-test-key)
  Full key: virtual-key-[REDACTED]

Step 3: Configuration
  export OPENAI_API_KEY="virtual-key-abc123..."
```

**Debug Mode (Explicit Opt-In):**
If you need to see full secrets for debugging:
```bash
# Use --reveal-secrets flag
./scenario_1_api_path.sh --verbose --reveal-secrets

# Or set environment variable
DEMO_REVEAL_SECRETS=1 ./scenario_1_api_path.sh --verbose
```

**⚠️ Warning when using --reveal-secrets:**
- Output will contain full API keys and tokens
- Only use in controlled debugging sessions
- Never share or store revealed output in tickets/transcripts
- Always redact before sharing: `make secrets-audit`

### OAuth Tokens in Logs

**Security Notice**: OAuth tokens may appear in logs when using subscription mode.

This is particularly relevant when:
- Testing OAuth-based authentication flows
- Debugging subscription request/response cycles
- Sharing logs for troubleshooting

**Best practices:**
- Review logs carefully before sharing
- Use `make clean` after debugging sensitive operations (this is safe; it only removes mounted log files)
- Consider redacting tokens when copying log excerpts
- Use `--reveal-secrets` only when necessary and never share that output

### Secrets Audit

The AI Control Plane includes an automated secrets audit that scans logs, backups, and build contexts for leaked credentials:

```bash
# Run secrets audit manually
make secrets-audit

# Run audit tests
make secrets-audit

# Run all script tests
make script-tests
```

**What the audit scans:**
- Mounted log directories (`demo/logs/` including gateway, compliance, normalized, and otel subdirectories)
- Backup files (`demo/backups/`)
- Docker build contexts (respecting `.dockerignore` exclusions)

**Leak patterns detected:**
- `Authorization: Bearer <token>` headers
- Standalone `Bearer <token>` patterns
- OpenAI-style API keys (`sk-<alphanumeric>`)
- LiteLLM API key headers (`x-litellm-api-key`)
- JWT-like tokens (three-part base64url format)

**Fail-fast behavior:**
- The audit exits with code 1 on first confirmed leak (unless `--report-all` is used)
- Raw secrets are never printed in output
- Output shows file path, line number, and pattern label only

**CI/CD integration:**
- `make lint` includes the secrets audit
- `make ci` will fail if leaks are detected
- Audit runs automatically during CI gate

**Remediation:**
If leaks are detected:
1. Identify the source file and line number from audit output
2. Redact or remove the sensitive data
3. If the leak is in a fixture or test file, ensure it's intentional and documented
4. Re-run `make secrets-audit` to verify the fix
5. If tokens were exposed, rotate them immediately through the provider console

### Log File Permissions

Log files are owned by root (Docker daemon) but readable by the user running docker commands. Ensure appropriate file permissions on the host:

```bash
# Ensure demo/logs/ has appropriate permissions
chmod 750 demo/logs/
```

## Troubleshooting

### Logs Not Rotating

If logs appear to grow beyond the configured size limit:

1. **Verify Docker daemon is running**
   ```bash
   docker info
   ```

2. **Check log driver configuration**
   ```bash
   docker inspect <container-id> | grep -A 10 LogConfig
   ```

3. **Restart services to apply new configuration**
   ```bash
   make down
   make up
   ```

### Excessive Log Volume

If logs are consuming more disk space than expected:

1. **Check for log files outside Docker's management**
   ```bash
   find demo/logs/ -type f -size +10M
   ```

2. **Manually remove large log files**
   ```bash
   make clean
   ```

3. **Consider reducing max-size in docker-compose.yml** (e.g., `"5m"` instead of `"10m"`)

### No Logs Appearing

If logs appear empty or missing:

1. **Verify services are running**
   ```bash
   make ps
   ```

2. **Check service health**
   ```bash
   make health
   ```

3. **Verify log directory exists and is writable**
   ```bash
   ls -la demo/logs/
   ```

## Advanced Configuration

### Adjusting Retention Policy

To change log retention limits, edit `demo/docker-compose.yml`:

```yaml
logging:
  driver: "json-file"
  options:
    max-size: "20m"    # Increase max file size
    max-file: "5"      # Keep more rotated files
```

After modifying, restart services:
```bash
make down
make up
```

### Alternative Log Drivers

For production deployments, consider alternative log drivers:

- **syslog**: Centralized logging with syslog servers
- **journald**: Integration with systemd journal
- **fluentd**: Log aggregation and forwarding
- **splunk**: Enterprise log management

Example syslog configuration:
```yaml
logging:
  driver: "syslog"
  options:
    syslog-address: "tcp://192.168.0.42:514"
```

### External Log Aggregation

For large-scale deployments, consider integrating with external log management systems:

- **ELK Stack** (Elasticsearch, Logstash, Kibana)
- **Grafana Loki**
- **CloudWatch Logs** (AWS)
- **Azure Monitor** (Azure)
- **Google Cloud Logging** (GCP)

These systems provide advanced features like log searching, alerting, and visualization.

## References

- [Docker Logging Documentation](https://docs.docker.com/config/containers/logging/)
- [LiteLLM Configuration](https://docs.litellm.ai/)
- [PostgreSQL Logging](https://www.postgresql.org/docs/current/runtime-config-logging.html)
- [DATABASE.md](../docs/DATABASE.md) - Database-specific logging and audit trails
