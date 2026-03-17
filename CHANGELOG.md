# Changelog

All notable changes to this project will be documented in this file.

This repository uses:
- [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) formatting
- semantic versioning from the tracked [`VERSION`](VERSION) file

## [Unreleased]

### Added
- Machine-readable config contract and schema validation for supported config surfaces.
- `examples/` operator reference directory with reusable deployment and pilot artifacts.
- Troubleshooting index, ADR home, and explicit generated-doc drift validation.
- Shared `internal/health` and tracked `internal/rbac` packages.

### Changed
- Release bundle and readiness workflows now default to the tracked root `VERSION` file.
- Release bundles include explicit release metadata and version-source files.
- Root README now presents the validated support boundary, examples, and release discipline more clearly.
- Gateway host, URL, TLS, and secret ergonomics now follow one canonical operator contract across `make`, `acpctl`, onboarding, and operator docs.
- `acpctl onboard` now performs post-write config linting plus actionable verification summaries for local contract issues, gateway reachability, and authorized model access.

### Removed
- Redundant empty placeholder packages: `internal/validate`, `internal/release`, and `internal/key`.

## [0.1.0] - 2026-03-16

### Added
- Initial public baseline for the host-first Docker reference implementation.
- Typed operator workflows for validation, status, readiness evidence, release bundles, and pilot closeout artifacts.
- Public documentation for deployment, governance, demos, and service offerings.
