# Portainer Notes

Portainer can be used as an optional operator convenience layer on top of the supported host-first Docker deployment, but it does not change the deployment contract.

## Contract

- Use `/etc/ai-control-plane/secrets.env` for host-production runtime configuration.
- Use `demo/.env` for local-only runs.
- Follow the canonical host-first workflow in [DEPLOYMENT.md](../DEPLOYMENT.md).

Portainer does not introduce a separate supported secret-sync or deployment path.
