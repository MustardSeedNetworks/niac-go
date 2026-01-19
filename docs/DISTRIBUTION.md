# Distribution & Licensing Roadmap

**Product:** NIAC
**Model:** TBD (Open Source or Commercial)
**Status:** Development - NOT FOR PUBLIC DISTRIBUTION

---

## Current State (Development)

All distribution channels are **locked down**:

| Channel | Status | Notes |
|---------|--------|-------|
| Container registry | DISABLED | No `container-push` target |
| Public downloads | DISABLED | No public artifacts |
| Package repos | DISABLED | .deb/.rpm stay local |

### Local Development Only

```bash
make container   # Builds locally only
make deb         # Creates dist/niac_*.deb (local)
make rpm         # Creates dist/niac_*.rpm (local)
```

---

## Distribution Decision Pending

### Option A: Open Source

**License:** Apache 2.0 or MIT

**Pros:**
- Community contributions
- Wider adoption
- Ecosystem building
- Complements commercial Seed/Stem

**Cons:**
- No direct revenue
- Support burden
- Competitors can use

**If Open Source:**
- [ ] Choose license (Apache 2.0 recommended)
- [ ] Clean up any proprietary code
- [ ] Add CONTRIBUTING.md
- [ ] Set up public GitHub repo
- [ ] Enable GitHub Container Registry (public)

### Option B: Commercial

**License:** Proprietary

**Pros:**
- Revenue stream
- Control over distribution
- Premium support model

**Cons:**
- Smaller user base
- More support overhead
- Licensing complexity

**If Commercial:**
- [ ] Implement license validation
- [ ] Set up private registry
- [ ] Customer portal integration

---

## Relationship to Seed/Stem

NIAC could be strategically open-sourced to:
1. Build community around network simulation
2. Drive adoption of commercial Seed/Stem
3. Create ecosystem lock-in
4. Attract contributors

```
┌─────────────────────────────────────────────────┐
│  NIAC (Open Source)                             │
│  └── Simulates network devices                  │
│                                                 │
│  Seed (Commercial)                              │
│  └── Diagnoses real networks                    │
│  └── Can test against NIAC simulations          │
│                                                 │
│  Stem (Commercial)                              │
│  └── Performance tests real networks            │
│  └── Can benchmark against NIAC simulations     │
└─────────────────────────────────────────────────┘
```

---

## Deployment Channels (Future)

### If Open Source:
```bash
# Public registry
docker pull ghcr.io/krisarmstrong/niac:latest

# Or package managers
apt install niac
dnf install niac
```

### If Commercial:
```bash
# Private registry with auth
docker pull registry.mustardseednetworks.com/niac:latest
```

---

## Pre-Release Checklist

**Decision Required:**
- [ ] **Decide: Open Source or Commercial?**

**If Open Source:**
- [ ] License file added (Apache 2.0 / MIT)
- [ ] CONTRIBUTING.md
- [ ] Code audit for proprietary content
- [ ] Public repo enabled
- [ ] CI/CD for public releases

**If Commercial:**
- [ ] License validation implemented
- [ ] Private registry configured
- [ ] Customer portal integration

---

## Version Strategy

**Single source of truth:** Git tags

```bash
git tag v1.0.0          # Creates version
make build              # Embeds version via ldflags
./bin/niac --version    # Shows v1.0.0
```

- `package.json` version is `0.0.0` (ignored, real version from API)
- Container tags match git tags
- All artifacts include version in filename

---

*Last updated: 2025-01-19*
*Status: Development lockdown - awaiting distribution decision*
