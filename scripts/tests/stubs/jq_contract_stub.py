#!/usr/bin/env python3
#
# AI Control Plane - jq Contract Stub
#
# Purpose:
#   Emulate the limited jq expressions used by shell contract tests.
#
# Responsibilities:
#   - Support `-e` presence checks for auth-cache schemas.
#   - Render normalized JSON payloads for the copy-helper tests.
#
# Scope:
#   - Test-only stub under scripts/tests/stubs.
#
# Usage:
#   - Install into a temp bin dir as `jq`.
#
# Invariants/Assumptions:
#   - Only the expressions used by chatgpt_auth_cache_copy tests are supported.

import json
import pathlib
import sys

args = sys.argv[1:]
if args and args[0] == "-e":
    expr = args[1]
    path = pathlib.Path(args[2])
    data = json.loads(path.read_text())
    ok = False
    if expr == ".tokens.access_token and .tokens.refresh_token and .tokens.id_token":
        ok = isinstance(data.get("tokens"), dict) and all(data["tokens"].get(k) for k in ("access_token", "refresh_token", "id_token"))
    elif expr == ".access_token and .refresh_token and .id_token":
        ok = all(data.get(k) for k in ("access_token", "refresh_token", "id_token"))
    sys.exit(0 if ok else 1)

expr = args[0]
path = pathlib.Path(args[1])
data = json.loads(path.read_text())
if expr.startswith("{"):
    if "tokens." in expr:
        tokens = data["tokens"]
        output = {
            "access_token": tokens["access_token"],
            "refresh_token": tokens["refresh_token"],
            "id_token": tokens["id_token"],
            "account_id": tokens.get("account_id"),
        }
    else:
        output = {
            "access_token": data["access_token"],
            "refresh_token": data["refresh_token"],
            "id_token": data["id_token"],
            "account_id": data.get("account_id"),
            "expires_at": data.get("expires_at"),
        }
    sys.stdout.write(json.dumps(output))
    sys.exit(0)

sys.exit(1)
