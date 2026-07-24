# CI/CD Integration

Use a published GitHub Release artifact when NIAC is a dependency of another
pipeline. Do not rebuild an arbitrary branch and call it a released NIAC
binary.

## Acquire and verify

1. Select an explicit NIAC release tag.
2. Download the archive or native package for the runner architecture.
3. Download `checksums.txt` and the asset’s `.cosign.bundle`.
4. Verify the checksum and keyless signature.
5. Verify the release provenance covers the asset.
6. Record the tag and asset digest with the test result.

Asset names include the NIAC version, platform, and architecture. Enumerate the
selected release rather than hard-coding a filename from an older release:

```bash
gh release view "$NIAC_VERSION" \
  --repo MustardSeedNetworks/niac-go \
  --json assets \
  --jq '.assets[].name'
```

The complete release and integrity contract is in
[DISTRIBUTION.md](DISTRIBUTION.md).

## Runner requirements

Linux runners need libpcap and permission to capture/inject packets. Prefer a
dedicated self-hosted runner attached only to an isolated test network.
Container-hosted shared runners generally cannot provide a truthful physical
network acceptance result.

For a daemon-driven test:

```bash
export NIAC_API_TOKEN="$(openssl rand -base64 32)"
niac daemon --listen 127.0.0.1:8445 >niac.log 2>&1 &
NIAC_PID=$!

curl -sk --fail https://127.0.0.1:8445/__version

# Run the pipeline's NIAC scenario here.

kill "$NIAC_PID"
wait "$NIAC_PID"
```

Use a trap in the real pipeline so the daemon and temporary configuration are
cleaned up on failure. Never print the bearer token or place it in a URL.

## Acceptance levels

| Level | Suitable runner | Evidence |
| --- | --- | --- |
| Config validation | Hosted or self-hosted | `niac validate`, schema, expected diagnostics |
| API/browser integration | Hosted or self-hosted | HTTPS daemon, authenticated journey, logs |
| Packet behavior | Privileged isolated runner | Capture, expected response frames, no leakage |
| Release acceptance | Deployment hosts and lab | Package install, `/__version`, browser and observer evidence |

Engine-only browser tests and loopback packet tests do not replace actual
Chrome/Edge/Safari or Link-Live/CyberScope acceptance.

## Repository release flow

NIAC’s own CI runs formatting, lint, unit/race, browser, security, schema, and
build gates on pull requests. Release Please selects the version and opens the
release pull request. Merging it creates the tag; the release workflow builds,
signs, attests, and publishes artifacts.

GitHub Actions is the only canonical release build environment. A local
`make build` is required development evidence but is not a publishable release
artifact.
