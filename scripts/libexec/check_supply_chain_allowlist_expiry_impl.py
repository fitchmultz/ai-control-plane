#!/usr/bin/env python3
# check_supply_chain_allowlist_expiry_impl.py - Supply-chain allowlist expiry guard
#
# Purpose:
#   Validate allowlist expiry windows in the supply-chain policy file.
#
# Responsibilities:
#   - Parse policy JSON and validate expires_on dates
#   - Warn for entries nearing expiry
#   - Fail when entries are expired or inside fail window
#
# Scope:
#   - Policy metadata validation only (does not perform vulnerability scanning)
#
# Exit codes:
#   0 = pass (no failing entries)
#   1 = domain failure (expired/invalid/fail-window entries found)
#   2 = usage/prerequisite failure

from __future__ import annotations

import argparse
import datetime as dt
import json
from pathlib import Path
import sys


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        prog="check_supply_chain_allowlist_expiry_impl.py",
        description="Validate supply-chain allowlist expiry windows.",
        epilog=(
            "Examples:\n"
            "  python3 scripts/libexec/check_supply_chain_allowlist_expiry_impl.py "
            "--policy demo/config/supply_chain_vulnerability_policy.json --warn-days 45 --fail-days 14\n"
            "  python3 scripts/libexec/check_supply_chain_allowlist_expiry_impl.py --policy demo/config/supply_chain_vulnerability_policy.json"
        ),
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument(
        "--policy",
        required=True,
        help="Path to supply-chain vulnerability policy JSON",
    )
    parser.add_argument(
        "--warn-days",
        type=int,
        default=45,
        help="Warn when an allowlist entry expires in fewer than this many days (default: 45)",
    )
    parser.add_argument(
        "--fail-days",
        type=int,
        default=14,
        help="Fail when an allowlist entry expires in fewer than this many days (default: 14)",
    )
    return parser.parse_args()


def main() -> int:
    args = parse_args()

    if args.warn_days < 0 or args.fail_days < 0:
        print("Error: --warn-days and --fail-days must be non-negative", file=sys.stderr)
        return 2

    policy_path = Path(args.policy)
    if not policy_path.is_file():
        print(f"Error: policy file not found: {policy_path}", file=sys.stderr)
        return 2

    try:
        policy = json.loads(policy_path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as err:
        print(f"Error: failed to read policy JSON: {err}", file=sys.stderr)
        return 2

    allowlist = policy.get("allowlist", [])
    if not isinstance(allowlist, list):
        print("Error: policy.allowlist must be an array", file=sys.stderr)
        return 1

    today = dt.date.today()
    warn_hits: list[tuple[str, int | str, str, str]] = []
    fail_hits: list[tuple[str, int | str, str, str]] = []

    for entry in allowlist:
        if not isinstance(entry, dict):
            fail_hits.append(("UNKNOWN", "invalid-entry", "n/a", "n/a"))
            continue

        identifier = f"{entry.get('id', 'UNKNOWN')}:{entry.get('package', 'unknown')}"
        ticket = str(entry.get("ticket", "n/a"))
        expires = entry.get("expires_on")

        if not isinstance(expires, str) or not expires:
            fail_hits.append((identifier, "missing", "n/a", ticket))
            continue

        try:
            expiry_date = dt.date.fromisoformat(expires)
        except ValueError:
            fail_hits.append((identifier, "invalid-date", expires, ticket))
            continue

        days_remaining = (expiry_date - today).days
        row = (identifier, days_remaining, expires, ticket)
        if days_remaining < args.fail_days:
            fail_hits.append(row)
        elif days_remaining < args.warn_days:
            warn_hits.append(row)

    if warn_hits:
        print("⚠ Allowlist entries nearing expiry:")
        for identifier, days_remaining, expires, ticket in warn_hits:
            print(
                f"  - {identifier}: {days_remaining} day(s) remaining "
                f"(expires {expires}, ticket {ticket})"
            )

    if fail_hits:
        print("✗ Allowlist expiry check failed:", file=sys.stderr)
        for identifier, days_remaining, expires, ticket in fail_hits:
            print(
                f"  - {identifier}: {days_remaining} day(s) remaining "
                f"(expires {expires}, ticket {ticket})",
                file=sys.stderr,
            )
        return 1

    print("✓ Allowlist expiry check passed")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
