# Hardened LibreChat runtime image for supply-chain risk reduction.
#
# Purpose:
# - Patch selected vulnerable runtime dependencies while preserving startup.
# - Remove global npm installation to eliminate toolchain CVEs from runtime scan.
#
# Responsibilities:
# - Keep upstream LibreChat server startup behavior compatible.
# - Reduce Trivy CRITICAL/HIGH findings with minimal runtime drift.
#
# Non-scope:
# - Does not rebuild frontend assets.
# - Does not re-resolve entire dependency graph from scratch.
#
# Invariants/Assumptions:
# - Base image remains ghcr.io/danny-avila/librechat:v0.8.5 (latest upstream release as of 2026-04-26).
# - Runtime entry command is `node api/server/index.js`.
# - Mongo/Meili env wiring remains external to image.
FROM ghcr.io/danny-avila/librechat:v0.8.5@sha256:a46254938507971e0d4f7ed3f9d116bd9b118f4810b5b75eb716baf575645068

USER root
WORKDIR /app

# Update vulnerable runtime dependencies while preserving startup behavior.
# LibreChat pins several packages via overrides; bump overrides first, then update.
RUN npm pkg set "overrides.axios=1.13.6" \
    && npm pkg set "overrides.fast-xml-parser=5.5.7" \
    && npm pkg set "overrides.minimatch=10.2.4" \
    && npm pkg set "overrides.@xmldom/xmldom=0.9.10" \
    && npm pkg set "overrides.hono=4.12.5" \
    && npm pkg set "overrides.@hono/node-server=1.19.11" \
    && npm pkg set "overrides.multer=2.1.1" \
    && npm update --omit=dev --no-audit --no-fund axios fast-xml-parser minimatch @xmldom/xmldom hono @hono/node-server

# LibreChat ships multer under the backend workspace path (/app/api/node_modules).
# Replace it in-place with v2.1.1 to satisfy CVE remediation without re-resolving all workspaces.
RUN tarball="$(npm pack --silent multer@2.1.1)" \
    && tmpdir="$(mktemp -d)" \
    && tar -xzf "$tarball" -C "$tmpdir" \
    && rm -rf /app/api/node_modules/multer \
    && mv "$tmpdir/package" /app/api/node_modules/multer \
    && rm -rf "$tmpdir" "$tarball"

# LibreChat's root workspace keeps a vulnerable @xmldom/xmldom copy pinned at 0.8.x.
# Replace the top-level copy in-place to land the 0.9.10 security fixes.
RUN tarball="$(npm pack --silent @xmldom/xmldom@0.9.10)" \
    && tmpdir="$(mktemp -d)" \
    && tar -xzf "$tarball" -C "$tmpdir" \
    && rm -rf /app/node_modules/@xmldom/xmldom \
    && mkdir -p /app/node_modules/@xmldom \
    && mv "$tmpdir/package" /app/node_modules/@xmldom/xmldom \
    && rm -rf "$tmpdir" "$tarball"

# Remove global npm runtime toolchain to drop related CVEs.
RUN rm -rf /usr/local/lib/node_modules/npm

CMD ["node", "api/server/index.js"]
