# Basic Deployment Example

Use this example when you want the narrowest supported starting point.

## Uses

- `demo/docker-compose.yml`
- `demo/config/litellm.yaml`
- `docs/DEPLOYMENT.md`

## Commands

```bash
make install
make validate-config
make up
make health
make prod-smoke
```

## Notes

- This is the base supported runtime: LiteLLM + PostgreSQL.
- Optional overlays are not part of this example.
