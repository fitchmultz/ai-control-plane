# Hardened LiteLLM runtime image for supply-chain risk reduction.
#
# Purpose:
# - Patch known high-severity Python dependency vulnerabilities.
# - Remove Node/npm toolchain artifacts not required by production runtime.
#
# Responsibilities:
# - Keep upstream LiteLLM runtime behavior compatible for proxy mode.
# - Reduce Trivy CRITICAL/HIGH findings without changing app entrypoint.
#
# Non-scope:
# - Does not modify LiteLLM application source code.
# - Does not pin upstream runtime digest updates automatically.
#
# Invariants/Assumptions:
# - Base image follows docker.litellm.ai/berriai/litellm:main-stable pinned to the
#   latest stable manifest available on 2026-03-06.
# - `litellm --config ...` entrypoint behavior is preserved.
# - npm is not required for runtime proxy execution.
FROM docker.litellm.ai/berriai/litellm:main-stable@sha256:cb7dc175c6e239557c36b5ccc262ac31359fdabbdee61a4afdd9c83525809802

USER root
WORKDIR /app
ENV PIP_DISABLE_PIP_VERSION_CHECK=1 \
    PIP_ROOT_USER_ACTION=ignore

# Apply Wolfi OS security updates for packages reported by Trivy and keep
# Prisma's cached Node runtime linked against required shared libraries.
RUN apk upgrade --no-cache \
    npm \
    python-3.13 \
    python-3.13-base \
    && apk add --no-cache libatomic

# Keep pip aligned with the latest stable release used for image builds.
RUN pip install --no-cache-dir --upgrade pip==26.0.1

# Address Python package CVEs identified by supply-chain gate scans.
RUN pip install --no-cache-dir --upgrade \
    cryptography==46.0.5 \
    python-multipart==0.0.22 \
    pillow==12.1.1

# Prisma's generated Python client bakes cache paths for the build user and an
# older engine layout; rewrite them to the writable runtime cache plus the
# current @prisma/engines location used by the installed CLI.
RUN python -c "from pathlib import Path; p = Path('/usr/lib/python3.13/site-packages/prisma/client.py'); text = p.read_text(); text = text.replace('/root/.cache/prisma-python', '/app/.cache/prisma-python'); text = text.replace('/node_modules/prisma/query-engine-', '/node_modules/@prisma/engines/query-engine-'); p.write_text(text)"

# LiteLLM main-stable currently ships a duplicate model block in the packaged
# litellm_proxy_extras schema; dedupe it so Prisma validation and the
# post-migration sanity check stay truthful.
RUN python - <<'PY'
import pathlib
import re

schema_path = pathlib.Path('/usr/lib/python3.13/site-packages/litellm_proxy_extras/schema.prisma')
lines = schema_path.read_text().splitlines()
deduped_lines = []
seen_models = set()
i = 0
while i < len(lines):
    line = lines[i]
    model_match = re.match(r'^model\s+(\w+)\s+\{$', line)
    if model_match is None:
        deduped_lines.append(line)
        i += 1
        continue

    model_name = model_match.group(1)
    block = [line]
    i += 1
    while i < len(lines):
        block.append(lines[i])
        if lines[i] == '}':
            i += 1
            break
        i += 1

    if model_name in seen_models:
        continue

    seen_models.add(model_name)
    deduped_lines.extend(block)

schema_path.write_text('\n'.join(deduped_lines) + '\n')
PY
RUN cd /usr/lib/python3.13/site-packages/litellm_proxy_extras && \
    DATABASE_URL=postgresql://prisma:prisma@127.0.0.1:5432/prisma prisma validate >/dev/null

# Remove Node ecosystem artifacts unused by LiteLLM proxy runtime.
RUN rm -rf \
    /app/litellm-js \
    /app/ui \
    /app/node_modules \
    /app/package-lock.json \
    /app/package.json \
    /usr/lib/node_modules/npm \
    /usr/lib/python3.13/site-packages/nodejs_wheel/lib/node_modules/npm \
    /usr/local/lib/node_modules/npm

RUN set -eux; \
    addgroup -S -g 10001 litellm; \
    adduser -S -D -u 10001 -G litellm -h /app litellm; \
    python_site_packages="$(python -c 'import site; print(next(path for path in site.getsitepackages() if path.endswith("site-packages")))')"; \
    experimental_dir="${python_site_packages}/litellm/proxy/_experimental"; \
    test -n "${python_site_packages}"; \
    test -d "${experimental_dir}"; \
    mkdir -p \
        /var/log/litellm \
        /app/.cache/prisma \
        /app/.cache/pycache \
        /app/.config \
        /app/.local/share/litellm_proxy_extras \
        /app/.local/state \
        /app/tmp \
        "${experimental_dir}/out"; \
    chown -R 10001:10001 \
        /app \
        /var/log/litellm \
        "${experimental_dir}"

ENV HOME=/app \
    XDG_CACHE_HOME=/app/.cache \
    XDG_CONFIG_HOME=/app/.config \
    XDG_DATA_HOME=/app/.local/share \
    XDG_STATE_HOME=/app/.local/state \
    LITELLM_MIGRATION_DIR=/app/.local/share/litellm_proxy_extras \
    PYTHONPYCACHEPREFIX=/app/.cache/pycache \
    TMPDIR=/app/tmp
USER 10001:10001

# Clean generated baseline diff directories before each startup so LiteLLM does
# not try to re-apply sanity-check migrations on container restart.
ENTRYPOINT ["/bin/sh", "-c", "migration_dir=${LITELLM_MIGRATION_DIR:-/app/.local/share/litellm_proxy_extras}; if [ -d \"$migration_dir/migrations\" ]; then find \"$migration_dir/migrations\" -maxdepth 1 -mindepth 1 -type d -name '*_baseline_diff' -exec rm -rf {} +; fi; exec litellm \"$@\"", "--"]
