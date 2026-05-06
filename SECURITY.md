# Security Policy

## Supported versions

Only the latest minor release on `main` receives security fixes. Older
tags are kept on the repo for reference but are not patched.

| Version          | Supported           |
| ---------------- | ------------------- |
| Latest (`main`)  | :white_check_mark:  |
| Anything older   | :x:                 |

## Reporting a vulnerability

**Please do not open a public issue for a security vulnerability.**

Use one of these private channels instead:

1. **GitHub Security Advisories (preferred):**
   <https://github.com/krisarmstrong/niac-go/security/advisories/new>.
   This creates a private advisory visible only to repository maintainers
   and you, with a built-in audit trail and CVE coordination workflow.
2. **Email:** `kris.armstrong@netally.com` with subject `[NIAC SECURITY]`.

Include in your report:

- A description of the vulnerability and the affected component(s).
- Steps to reproduce, ideally with a minimal proof-of-concept.
- The version / commit you tested against.
- The potential impact (e.g. unauthenticated RCE, info disclosure, DoS).
- A suggested fix or mitigation, if you have one.

## What to expect

- **Acknowledgment** within 2 business days.
- **Triage** with a severity assessment within 7 business days.
- **Fix or mitigation** released within 30 days for high/critical
  severity, longer for low severity. We coordinate disclosure timing
  with you for high-impact issues.
- **Credit** in the resulting security advisory, if you'd like it
  (researcher names attributed in the GHSA / release notes).

## Scope

In scope:

- Code in this repository (Go backend, embedded React UI, CI workflows,
  release pipeline).
- Built artifacts published as part of a tagged GitHub release
  (verifiable via the included `cosign` signatures and SBOM).

Out of scope:

- Vulnerabilities in third-party dependencies — please report those
  upstream. We track them via Dependabot and `govulncheck` and patch on
  the next release.
- Self-inflicted misconfigurations (e.g. running `niac daemon --token=""`
  on a public network — the daemon explicitly warns against this).

## Hardening notes for operators

- Always run `niac daemon --token <secret>` when exposed to the network.
- Set `--webhook-allowed-host` to lock the alert webhook destination to
  a known host (the default still rejects raw private/loopback IPs but
  an explicit allowlist is the canonical SSRF defense).
- Verify release artifacts with `cosign verify-blob` against
  `<file>.cosign.bundle`. Each release also ships a CycloneDX SBOM per
  archive — the verification recipe is in the GitHub release notes.

Thank you for helping keep this project secure.
