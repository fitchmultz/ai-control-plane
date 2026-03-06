# Browser and Workspace Proof Track

This is the recommended secondary proof track after the host-first gateway baseline. It addresses the enterprise reality that a large share of AI usage happens in browser-based tools and vendor-managed workspaces.

## What This Track Is For

Use this track when the buyer asks questions such as:

- How do we govern browser-based AI use for non-developers?
- How do we attribute LibreChat or managed workspace usage to real users?
- How do we treat direct SaaS usage that does not traverse the gateway?
- What do we need from our workspace, browser, and endpoint teams?

## What The Repository Validates

The repository can validate these claims locally:

- Managed browser chat can be routed through LibreChat into LiteLLM.
- Approved model lists can be pinned and drift-controlled.
- Shared-key routing can still preserve user attribution when trusted server-side identity context is forwarded correctly.
- Normalized evidence and detections can combine routed and non-routed signals into one operational picture.

Canonical sources:

- [tooling/LIBRECHAT.md](tooling/LIBRECHAT.md)
- [security/ENTERPRISE_AUTH_ARCHITECTURE.md](security/ENTERPRISE_AUTH_ARCHITECTURE.md)
- [security/SIEM_INTEGRATION.md](security/SIEM_INTEGRATION.md)

## What The Customer Must Validate

The customer still owns the controls that prevent unmanaged browser usage from escaping governance:

- Browser management and sanctioned-extension policy
- SWG/CASB/firewall policy for AI endpoints
- Vendor workspace admin controls, audit exports, and retention settings
- IdP policy, device posture, and access boundaries
- SOC workflow for alerts triggered by direct or bypass usage

This is why the browser/workspace path is a proof track, not a standalone repo claim.

Reference validation tracker:

- [PILOT_CUSTOMER_VALIDATION_CHECKLIST.md](PILOT_CUSTOMER_VALIDATION_CHECKLIST.md)

## Pilot Acceptance Criteria

A credible browser/workspace pilot should end with all of the following:

- Routed LibreChat usage is attributed to real users in gateway evidence.
- Approved model exposure in the browser UI is restricted to the intended set.
- Customer-owned browser or network controls for direct SaaS usage are documented and tested.
- Direct/bypass activity has a defined detection-and-response path into the SIEM.
- Workspace administrators and security operators agree on who owns enforcement, exports, and incident handling.

## Bypass Strategy

Use this four-lane model when discussing browser and workspace governance:

| Lane | Example | Repo role | Customer role | Honest claim |
|---|---|---|---|---|
| Routed sanctioned browser/chat path | LibreChat through LiteLLM | configure, validate, evidence | approve and operate sanctioned entrypoint | governed routed path |
| Vendor workspace with admin exports | ChatGPT Enterprise, Claude Enterprise | define evidence pattern, normalize, detect | configure workspace, retention, admin policy | governed with vendor-dependent evidence |
| Direct SaaS browser access | unmanaged direct usage | detect and correlate where signals exist | prevent or constrain with SWG/CASB, browser, endpoint | detective unless customer blocks it |
| Unmanaged device or unsanctioned app path | personal browser, rogue plugin, BYO key | limited evidence only | device, network, IAM, procurement controls | not controlled by repo alone |

That last lane is where buyers often want the strongest claim. It is also the lane most dependent on customer-owned controls.

## Demo vs Customer Reality

The local demo proves the governed pattern. It does not prove that the customer has actually closed the bypass path in their environment.

That distinction should be stated plainly in every pilot readout.

## Do Not Say

- "browser AI is fully controlled"
- "workspace governance removes the need for network controls"
- "direct SaaS use is blocked" unless the customer has validated that separately
