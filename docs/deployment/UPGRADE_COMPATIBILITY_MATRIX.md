# Upgrade Compatibility Matrix

| From | To | Supported | Path rule | Config migration | DB migration | Rollback |
| --- | --- | --- | --- | --- | --- | --- |
| pre-framework releases | 0.1.0 | No | Fresh install + restore only | Manual or fresh-secrets cutover | Manual restore only | Manual rollback only |
| any release | next release | Only when an explicit edge exists in the typed upgrade catalog | Adjacent-only unless every intermediate edge is declared | Edge-specific | Edge-specific | Restore upgrade snapshots from the previous checkout |
| any release | non-adjacent release | Only when every intermediate edge exists and is executed in order | Multi-hop explicit chain required | Edge-specific | Edge-specific | Restore upgrade snapshots from the previous checkout |

## Current Explicit Edges

No in-place edges are shipped in `0.1.0`.

Future releases must add explicit rows here when they add typed release edges.

## Release Discipline

Every shipped release that claims upgrade support must update:

- `internal/upgrade` release-edge catalog
- this matrix
- `RELEASE_NOTES.md`
- `CHANGELOG.md`
