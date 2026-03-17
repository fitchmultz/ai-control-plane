# TLS Ingress Example

Use this example when external client access must terminate through the supported TLS ingress overlay.

## Commands

```bash
make validate-config-production SECRETS_ENV_FILE=/etc/ai-control-plane/secrets.env
make up-tls
make tls-health
```

## References

- `demo/docker-compose.tls.yml`
- `demo/config/caddy/Caddyfile.prod`
- `docs/deployment/TLS_SETUP.md`
