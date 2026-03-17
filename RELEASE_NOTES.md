# Release Notes Convention

This file defines how to write operator-facing release notes for AI Control Plane.

## Source of Truth

For every release:
1. Update [`VERSION`](VERSION)
2. Add the release entry to [`CHANGELOG.md`](CHANGELOG.md)
3. Publish release notes using the template below

## Release Notes Template

# AI Control Plane X.Y.Z

## Highlights
- What materially changed
- Why an operator or buyer should care

## Operator Impact
- Required actions
- Optional actions
- No-op / safe upgrade notes

## Validation
- `make ci`
- `make validate-config`
- Any release-specific verification commands

## Docs Updated
- README
- deployment docs
- reference docs
- examples

## Known Limits
- Explicitly state what is still not validated or still incubating
