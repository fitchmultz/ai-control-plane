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

# Apply Wolfi OS security updates for packages reported by Trivy.
RUN apk upgrade --no-cache \
    npm \
    python-3.13 \
    python-3.13-base

# Keep pip aligned with the latest stable release used for image builds.
RUN pip install --no-cache-dir --upgrade pip==26.0.1

# Address Python package CVEs identified by supply-chain gate scans.
RUN pip install --no-cache-dir --upgrade \
    cryptography==46.0.5 \
    python-multipart==0.0.22 \
    pillow==12.1.1

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

RUN addgroup -S -g 10001 litellm \
    && adduser -S -D -u 10001 -G litellm -h /app litellm \
    && mkdir -p /var/log/litellm \
    && chown -R 10001:10001 /app /var/log/litellm

ENV HOME=/app
USER 10001:10001
