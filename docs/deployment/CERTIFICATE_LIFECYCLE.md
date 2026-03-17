# Certificate Lifecycle Runbook

## Supported surface

The supported host-first TLS path uses Caddy-managed certificates stored in Docker named volumes:
- `ai_control_plane_caddy_data_<slot>`
- `ai_control_plane_caddy_config_<slot>`

Use typed ACP workflows for day-to-day operations:

```bash
make cert-status
./scripts/acpctl.sh cert list
./scripts/acpctl.sh cert inspect --domain gateway.example.com
./scripts/acpctl.sh cert check --threshold-days 30
./scripts/acpctl.sh cert renew --domain gateway.example.com
sudo make cert-renew-install SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env
```

## Health thresholds

- Warning: 30 days
- Critical: 7 days

`acpctl cert check` validates both:
- the stored Caddy-managed certificate metadata
- the live served TLS certificate on the configured HTTPS endpoint

Warning and unhealthy certificate states exit non-zero so they can be used in operator gates.

## Manual renewal

ACP manual renewal preserves rollback artifacts under:

```text
demo/logs/cert-renewals/
```

ACP performs:
1. snapshot the stored certificate directory
2. remove the stored certificate directory
3. restart Caddy
4. verify a replacement certificate appears
5. restore the snapshot automatically if verification fails

## Automated renewal

The supported host path installs:
- `ai-control-plane-cert-renewal.service`
- `ai-control-plane-cert-renewal.timer`

The timer runs daily by default and only forces renewal when the certificate is within the configured threshold.

Host-first Ansible convergence installs and manages this timer automatically when the `tls` overlay is enabled. Manual host installs can use:

```bash
sudo make cert-renew-install SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env
systemctl status ai-control-plane-cert-renewal.timer
```

## Development certificates

Development TLS uses Caddy's internal CA. The certificate may be valid but untrusted until the Caddy root CA is installed on the client host.

## Production certificates

Production TLS uses Caddy-managed ACME issuance for the configured hostname. The supported operator workflow is to:
1. ensure DNS and public ingress are correct
2. run `make up-production` or the host-first `host apply` path with the `tls` overlay
3. validate `./scripts/acpctl.sh cert check --threshold-days 30`
4. confirm `systemctl status ai-control-plane-cert-renewal.timer` remains healthy

## Recovery guidance

If certificate renewal fails repeatedly:
1. inspect stored state with `./scripts/acpctl.sh cert list`
2. inspect the specific hostname with `./scripts/acpctl.sh cert inspect --domain <host>`
3. review Caddy logs
4. confirm DNS, ACME reachability, and ports 80/443
5. use the saved rollback artifact under `demo/logs/cert-renewals/`
