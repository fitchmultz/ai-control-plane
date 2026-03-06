# 5–10 Minute Demo Script

## Goal

Demonstrate that the control plane is installable, governed, testable, and release-ready without external provider keys.

## Steps

1. **Setup + deterministic PR gate**

```bash
make install-ci
make ci-pr
```

Expect: all checks pass with deterministic output.

2. **Full gate with runtime**

```bash
make ci
```

Expect: offline runtime boots in CI slot, gateway becomes ready, detection validation passes, teardown runs.

3. **Manual runtime proof**

```bash
ACP_SLOT=ci-runtime make up-offline
ACP_SLOT=ci-runtime make ci-runtime-checks
ACP_SLOT=ci-runtime make down-offline-clean
```

Expect: health/models checks pass and teardown removes CI-slot volumes.

4. **Release artifact confidence**

```bash
make release-bundle
make release-bundle-verify
```

Expect: bundle + checksums produced, verification succeeds.

## Troubleshooting quick hits

- Gateway readiness delay: check `docker compose -f demo/docker-compose.offline.yml logs litellm`.
- Hygiene failure: remove tracked local-only artifacts and rerun `make public-hygiene-check`.
- Supply-chain failure: inspect policy windows/digests in `demo/config/supply_chain_vulnerability_policy.json`.
