# ADR 0005: Ed25519-signed license tokens

**Status:** Accepted (2026-06-08)

## Context

NIAC's license keys were 16-character strings validated offline by a rotor
("Enigma-style") substitution cipher plus a 2-character polynomial self-checksum
(`internal/license/cipher.go`, `validator.go`). The scheme was **symmetric and
self-describing**: the exact algorithm, rotor tables, and `GenerateLicenseKey`
shipped inside every NIAC binary. Anyone with a copy of the public binary could
read the generator and mint unlimited valid Pro keys — the "license" was security
theater. seed and stem shipped the same forgeable scheme.

We need offline validation (clinical/industrial/air-gapped customers, no
phone-home) **and** un-forgeability. Those are reconcilable with asymmetric
signatures: the binary needs only a public key to verify; the secret needed to
mint a key never ships.

This is pre-launch, so there are no issued customer keys to honor — we can
replace the format outright rather than carry a compatibility shim.

## Decision

License keys become **Ed25519-signed tokens**. The format (`signing.go`) is
shared by convention across seed/stem/NIAC and the keygen tool — each repo owns
its own copy (no master module), matching the harmonization rule:

```
MSN1.<base64url(payload)>.<base64url(signature)>
```

- `MSN1` is the scheme + version tag.
- `payload` is canonical JSON: `{v, product, code, serial, tier, maxDevices,
  iat, exp}`. `product` binds a token to one product; `exp=0` means perpetual.
- `signature` is the 64-byte Ed25519 signature over the exact payload bytes.

Verification (`Verifier.Validate`) checks the signature **before** interpreting
any field, then enforces: scheme, payload schema version, `product == "niac"`,
known tier, product code, and expiry. Tier→feature mapping stays **in-binary**,
so a signed token can only grant features this build defines.

- The binary embeds only the base64 Ed25519 **public** key
  (`licensePublicKeyB64`). The private key lives solely in the keygen tool.
- `GenerateLicenseKey` and the rotor cipher are **deleted** from the product.
  Tests mint tokens with an ephemeral key via an exported `NewVerifier`; the
  cross-tool contract is pinned by a production-signed vector (`TestKeygenContract`).

## Consequences

- A NIAC binary can no longer mint a valid license; forging one requires the
  Ed25519 private key. `TestForgeryRejected` / `TestTamperRejected` lock this in.
- Validation stays fully offline — no network, no phone-home — preserving the
  air-gapped story.
- Tokens are longer (~200 chars) than the old 16-char key; they are copy/paste
  artifacts, not hand-typed, and `FormatKey` no longer strips characters (it must
  not — base64url uses `-`/`_`).
- The embedded public key is a **pre-launch** key generated for this change; it
  must be rotated via keygen before GA (regenerate the key + the contract vector
  together). Tracked in the keygen ADR.
- seed and stem adopt the identical format so one keygen serves all three.

## Related issues and PRs

- The Ed25519 license item in the stem/niac remediation plan; the keygen signing
  change; the parallel stem and seed adoptions.
