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
# - Base image remains docker.litellm.ai/berriai/litellm:v1.81.3-stable.
# - `litellm --config ...` entrypoint behavior is preserved.
# - npm is not required for runtime proxy execution.
FROM docker.litellm.ai/berriai/litellm:v1.81.3-stable@sha256:3d80908e35230a0dfb58051ae4e2847371b2a327b44e105e79b8c4f67a65596c

USER root
WORKDIR /app

# Apply Wolfi OS security updates for packages reported by Trivy.
RUN apk upgrade --no-cache \
    npm \
    python-3.13 \
    python-3.13-base

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
