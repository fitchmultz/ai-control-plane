# Incubating Deployment Assets

This directory holds deployment tracks that are retained in-repo for explicit internal exploration only. They are not part of the supported host-first product surface, not part of the public operator UX, and not part of the default CI or validation contract.

## Status

- `helm/` contains incubating Helm assets.
- `terraform/` contains incubating Terraform assets.

## Rules

- Do not reference these paths from primary docs, public `make` help, or public `acpctl` help.
- Do not add default CI dependencies on these tracks.
- Validate them only through explicit internal workflows.
