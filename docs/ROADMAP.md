# NIAC Roadmap

NIAC is in pre-1.0 stabilization. Product features are frozen until every exit
criterion below is complete and verified from a release-built artifact.

## Pre-1.0 Exit Criteria

- [ ] Validate the routed CyberScope scenario end to end, including DHCP,
  routed discovery, SNMP walks, and Link-Live observation.
- [x] Make injected interface faults observable through IF-MIB and EtherLike-MIB
  counters while preserving monotonic counter behavior.
- [x] Remove stale roadmap, compatibility, licensing, and deployment claims from
  active documentation and close superseded tracking issues.
- [ ] Pass lint, formatting, unit, integration, browser, security, package,
  install, and deployment validation for the release candidate.

No new product capability enters the pre-1.0 line until these criteria pass.

## v1 Product Boundary

NIAC simulates configurable network devices and protocol behavior for lab,
monitoring, discovery, and troubleshooting workflows. The supported surface
includes the CLI, embedded web UI, API, topology and scenario configuration,
packet capture analysis and replay, and the protocol implementations shipped in
the current binary.

NIAC ships as one unrestricted binary. There is no runtime tier, no activation,
and no phone-home. Resource ceilings are technical safety limits rather than
entitlements: 1,000 devices for one configuration, plus daemon-wide budgets
bounding concurrent sessions and total devices.

## Release Process

Release Please owns version selection and creates the release PR. Merging that
PR creates the tag; the release workflow then builds, signs, attests, and
publishes the platform artifacts. See [Distribution](DISTRIBUTION.md) for the
artifact matrix and validation contract.

After v1.0.0, roadmap work will be selected through the project feature-scope
and marketability gates. It is intentionally not precommitted here.
