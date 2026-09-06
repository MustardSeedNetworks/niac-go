# Changelog

All notable changes to NIAC will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.95.10](https://github.com/MustardSeedNetworks/niac-go/compare/v0.95.9...v0.95.10) (2026-09-06)


### Bug Fixes

* **acceptance:** treat voip-phone and ap as leaf nodes in the comparator ([#1848](https://github.com/MustardSeedNetworks/niac-go/issues/1848)) ([105ee1e](https://github.com/MustardSeedNetworks/niac-go/commit/105ee1ea2fb7da3571e0c1dcb1339a2e2dbf0b1f))

## [0.95.9](https://github.com/MustardSeedNetworks/niac-go/compare/v0.95.8...v0.95.9) (2026-09-06)


### Bug Fixes

* **netbios:** send routed NBSTAT replies from the gateway MAC ([#1843](https://github.com/MustardSeedNetworks/niac-go/issues/1843)) ([9fcfdd8](https://github.com/MustardSeedNetworks/niac-go/commit/9fcfdd8415d2378a89e5ff2d0625634783f43e2d))


### Continuous Integration

* install the pinned golangci-lint when the local version differs ([#1839](https://github.com/MustardSeedNetworks/niac-go/issues/1839)) ([c1d1d7a](https://github.com/MustardSeedNetworks/niac-go/commit/c1d1d7a4ec141c693d4af06b8f7187ec8328ead5))

## [0.95.8](https://github.com/MustardSeedNetworks/niac-go/compare/v0.95.7...v0.95.8) (2026-09-06)


### Bug Fixes

* **deps:** update dependency react-hook-form to v7.87.0 ([#1838](https://github.com/MustardSeedNetworks/niac-go/issues/1838)) ([67e5929](https://github.com/MustardSeedNetworks/niac-go/commit/67e592993788b13d8101bc9b35cafadd8622aeca))

## [0.95.7](https://github.com/MustardSeedNetworks/niac-go/compare/v0.95.6...v0.95.7) (2026-09-06)


### Bug Fixes

* **ci:** do not arm auto-merge on a release PR that no longer describes main ([#1831](https://github.com/MustardSeedNetworks/niac-go/issues/1831)) ([26702d8](https://github.com/MustardSeedNetworks/niac-go/commit/26702d8085c6f107cbb33cf0b2b67a24a996f77d))


### Miscellaneous

* **deps:** update pre-commit hook gitleaks/gitleaks to v8.30.1 ([#1832](https://github.com/MustardSeedNetworks/niac-go/issues/1832)) ([650042b](https://github.com/MustardSeedNetworks/niac-go/commit/650042ba29bdcef5af78f22e0d7db88a32655f20))
* **deps:** update pre-commit hook igorshubovych/markdownlint-cli to v0.49.1 ([#1833](https://github.com/MustardSeedNetworks/niac-go/issues/1833)) ([18a0b9c](https://github.com/MustardSeedNetworks/niac-go/commit/18a0b9cabdb28c28e26e52aaf10e34c4e558c13d))
* **deps:** update pre-commit hook koalaman/shellcheck-precommit to v0.11.0 ([#1834](https://github.com/MustardSeedNetworks/niac-go/issues/1834)) ([5e42771](https://github.com/MustardSeedNetworks/niac-go/commit/5e42771201ed3c2b19a4b2a77385d46197f68faf))

## [0.95.6](https://github.com/MustardSeedNetworks/niac-go/compare/v0.95.5...v0.95.6) (2026-09-05)


### Bug Fixes

* **ui:** show fetch failures instead of rendering an empty page ([#1829](https://github.com/MustardSeedNetworks/niac-go/issues/1829)) ([9bc7292](https://github.com/MustardSeedNetworks/niac-go/commit/9bc72921d0c9b8176a63a16bcf579d915d84622f))

## [0.95.5](https://github.com/MustardSeedNetworks/niac-go/compare/v0.95.4...v0.95.5) (2026-09-05)


### Features

* **lldp:** advertise LLDP-MED so endpoints identify themselves ([#1824](https://github.com/MustardSeedNetworks/niac-go/issues/1824)) ([22d321d](https://github.com/MustardSeedNetworks/niac-go/commit/22d321d2342a92340a09a25c9e2ef170e22eb4dd))

## [0.95.4](https://github.com/MustardSeedNetworks/niac-go/compare/v0.95.3...v0.95.4) (2026-09-05)


### Bug Fixes

* **capture:** decode packets once, so all three surfaces agree ([#1819](https://github.com/MustardSeedNetworks/niac-go/issues/1819)) ([c20eea2](https://github.com/MustardSeedNetworks/niac-go/commit/c20eea2163cb8d6e9dce421acba4431c06cee409))
* **sanitize:** map subnets, not addresses, and stop damaging vendor strings ([#1821](https://github.com/MustardSeedNetworks/niac-go/issues/1821)) ([49ee339](https://github.com/MustardSeedNetworks/niac-go/commit/49ee339e1cb8f0952e53c6fe78ffcaf067d95bfb))

## [0.95.3](https://github.com/MustardSeedNetworks/niac-go/compare/v0.95.2...v0.95.3) (2026-09-05)


### Miscellaneous

* **deps:** lock file maintenance ([#1814](https://github.com/MustardSeedNetworks/niac-go/issues/1814)) ([7ab46fb](https://github.com/MustardSeedNetworks/niac-go/commit/7ab46fbb8da6b8ec6d41ee4fdf935f94b144adfb))

## [0.95.2](https://github.com/MustardSeedNetworks/niac-go/compare/v0.95.1...v0.95.2) (2026-09-05)


### Miscellaneous

* **deps:** lock file maintenance ([#1812](https://github.com/MustardSeedNetworks/niac-go/issues/1812)) ([7b9260a](https://github.com/MustardSeedNetworks/niac-go/commit/7b9260adce02d061258dc3db76b00e9cdda50a50))

## [0.95.1](https://github.com/MustardSeedNetworks/niac-go/compare/v0.95.0...v0.95.1) (2026-09-05)


> Backfilled by hand. These commits were in the tag but missing from this
> entry: the release PR auto-merged before release-please regenerated it
> (niac#1817). The version bump is left as released.


### ⚠ BREAKING CHANGES

* **cli:** delete the legacy runtime and `niac run` (P1c-3) ([#1806](https://github.com/MustardSeedNetworks/niac-go/issues/1806)). `daemon --once` is the one foreground runtime. This should have bumped a minor under `bump-minor-pre-major`; it shipped as a patch and is left as released.

### Bug Fixes

* **protocols:** route diagnostics through internal/logging, not os.Stdout ([#1810](https://github.com/MustardSeedNetworks/niac-go/issues/1810)) ([b5b421a](https://github.com/MustardSeedNetworks/niac-go/commit/b5b421ad6b20ed6b5916c751612fcbdf2093e510))


### Code Refactoring

* **cli:** delete the legacy runtime and `niac run` (P1c-3) ([#1806](https://github.com/MustardSeedNetworks/niac-go/issues/1806)) ([a0703ee](https://github.com/MustardSeedNetworks/niac-go/commit/a0703eeb37cbff88206975cf08dde8688cac7f8f))


### Continuous Integration

* restore the release-PR auto-merge step, with a repository for gh ([#1804](https://github.com/MustardSeedNetworks/niac-go/issues/1804)) ([3c353c8](https://github.com/MustardSeedNetworks/niac-go/commit/3c353c89104688d281b2354570d806d7e32fdefe))

## [0.95.0](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.92...v0.95.0) (2026-09-05)


### ⚠ BREAKING CHANGES

* **cli:** remove the interactive TUI (P1c-6) ([#1802](https://github.com/MustardSeedNetworks/niac-go/issues/1802))

### Features

* **capture:** export a session's retained frames as pcapng (P1c-10) ([#1799](https://github.com/MustardSeedNetworks/niac-go/issues/1799)) ([4a094f4](https://github.com/MustardSeedNetworks/niac-go/commit/4a094f4d33a08004a5eff72b81ede2f14a341d36))


### Code Refactoring

* **cli:** remove the interactive TUI (P1c-6) ([#1802](https://github.com/MustardSeedNetworks/niac-go/issues/1802)) ([3e9aaa1](https://github.com/MustardSeedNetworks/niac-go/commit/3e9aaa11b3f885f93e9cbc5cd72a7ee9dde84f46))


### Documentation

* generate the CLI reference from the cobra command tree (P1c-9) ([#1794](https://github.com/MustardSeedNetworks/niac-go/issues/1794)) ([32a9e93](https://github.com/MustardSeedNetworks/niac-go/commit/32a9e9373500cbf3399413ae87c355d23ab06f10))

## [0.94.92](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.91...v0.94.92) (2026-09-04)


> Backfilled by hand. These commits were in the tag but missing from this
> entry: the release PR auto-merged before release-please regenerated it
> (niac#1817).


### Features

* **cli:** add daemon --once, a foreground single-shot runtime ([#1764](https://github.com/MustardSeedNetworks/niac-go/issues/1764)) ([849dc41](https://github.com/MustardSeedNetworks/niac-go/commit/849dc411)) — backfilled
* **content:** let the content bundle upgrade (P1c-5) ([#1775](https://github.com/MustardSeedNetworks/niac-go/issues/1775)) ([17f5b96](https://github.com/MustardSeedNetworks/niac-go/commit/17f5b96b8c023bdd77f4d68a51e79c4e8fdac56b))


### Bug Fixes

* **deps:** update dependency react-router to v8.3.1 ([#1779](https://github.com/MustardSeedNetworks/niac-go/issues/1779)) ([2efaaef](https://github.com/MustardSeedNetworks/niac-go/commit/2efaaeff4c3aa269f87c4c143dbf1cc84ca04717))
* **library:** scope draft read/write locks per draft name ([#1797](https://github.com/MustardSeedNetworks/niac-go/issues/1797)) ([30aa3a6](https://github.com/MustardSeedNetworks/niac-go/commit/30aa3a6e9db350af5694be13463d06c11f472225)), closes [#1795](https://github.com/MustardSeedNetworks/niac-go/issues/1795)
* **sanitize:** remove real addresses the sanitizer could not reach, and gate them ([#1770](https://github.com/MustardSeedNetworks/niac-go/issues/1770)) ([120aef8](https://github.com/MustardSeedNetworks/niac-go/commit/120aef845507a34e2cce989c860cb570064b94eb)), closes [#1769](https://github.com/MustardSeedNetworks/niac-go/issues/1769)


### Documentation

* clear the markdown debt that fails the docs gate on any touched file ([#1768](https://github.com/MustardSeedNetworks/niac-go/issues/1768)) ([75e3c51](https://github.com/MustardSeedNetworks/niac-go/commit/75e3c515c2df394f6e75b9a9a6513cdf8f72934f)), closes [#1767](https://github.com/MustardSeedNetworks/niac-go/issues/1767)


### Tests

* **auth:** pin the fleet SIEM contract for auth.forbidden ([#1761](https://github.com/MustardSeedNetworks/niac-go/issues/1761)) ([0ef2555](https://github.com/MustardSeedNetworks/niac-go/commit/0ef2555dcd788e28fb0b00e92566bf19b87d73ee))
* **capture:** count the refill goroutine by name, not the whole process ([#1782](https://github.com/MustardSeedNetworks/niac-go/issues/1782)) ([8d7c199](https://github.com/MustardSeedNetworks/niac-go/commit/8d7c199e6a3139359020eb3183f3cf4222005a52))
* **e2e:** author one network three ways and replay it on the wire (P1b-6) ([#1758](https://github.com/MustardSeedNetworks/niac-go/issues/1758)) ([2ef5717](https://github.com/MustardSeedNetworks/niac-go/commit/2ef5717e0ae286dee08f7d09e74fd50aaa57563e))
* **e2e:** wait for the device dialog's autofocus before filling it ([#1796](https://github.com/MustardSeedNetworks/niac-go/issues/1796)) ([1d2d578](https://github.com/MustardSeedNetworks/niac-go/commit/1d2d578347d7a93b1a078c7e73ee77bbf64621bb)), closes [#1773](https://github.com/MustardSeedNetworks/niac-go/issues/1773)


### Continuous Integration

* arm auto-merge on the release PR ([#1777](https://github.com/MustardSeedNetworks/niac-go/issues/1777)) ([6ee75a5](https://github.com/MustardSeedNetworks/niac-go/commit/6ee75a59c52bb0e39b52c0134a8021332d3995af)), closes [#1776](https://github.com/MustardSeedNetworks/niac-go/issues/1776)
* collapse the file-size gate to one enforced limit ([#1787](https://github.com/MustardSeedNetworks/niac-go/issues/1787)) ([5f9602f](https://github.com/MustardSeedNetworks/niac-go/commit/5f9602f1d406281f4f907b199897287210f5a71c)), closes [#1784](https://github.com/MustardSeedNetworks/niac-go/issues/1784)
* decide npm audit outcomes on the JSON verdict, not on prose ([#1791](https://github.com/MustardSeedNetworks/niac-go/issues/1791)) ([b9fd29d](https://github.com/MustardSeedNetworks/niac-go/commit/b9fd29dbb33a25a13798e103a8b08ea2a9ba5762)), closes [#1790](https://github.com/MustardSeedNetworks/niac-go/issues/1790)
* drop the five funlen exclusions that exclude nothing ([#1786](https://github.com/MustardSeedNetworks/niac-go/issues/1786)) ([83f67f9](https://github.com/MustardSeedNetworks/niac-go/commit/83f67f9b634677a9ee212ec662f8a1b7d9d0f13a)), closes [#1785](https://github.com/MustardSeedNetworks/niac-go/issues/1785)
* lint the Windows tree with GOOS=windows, gating new findings only ([#1789](https://github.com/MustardSeedNetworks/niac-go/issues/1789)) ([ce4b82b](https://github.com/MustardSeedNetworks/niac-go/commit/ce4b82bd30fbc2710f4fdc86edbe60113201dcbf)), closes [#1788](https://github.com/MustardSeedNetworks/niac-go/issues/1788)
* retry npm audit when the registry, not the tree, is the problem ([#1772](https://github.com/MustardSeedNetworks/niac-go/issues/1772)) ([291e793](https://github.com/MustardSeedNetworks/niac-go/commit/291e793d43a791fb2951e3626b9a4d8045e1be33)), closes [#1771](https://github.com/MustardSeedNetworks/niac-go/issues/1771)


### Miscellaneous

* **deps:** lock file maintenance ([#1762](https://github.com/MustardSeedNetworks/niac-go/issues/1762)) ([07052d6](https://github.com/MustardSeedNetworks/niac-go/commit/07052d6c20e85db628d5abccb71b9bc986adb86a))
* **deps:** lock file maintenance ([#1780](https://github.com/MustardSeedNetworks/niac-go/issues/1780)) ([6f89b74](https://github.com/MustardSeedNetworks/niac-go/commit/6f89b74df15be0b1a4ed1f1f7e9c16a3a7b83022))
* **deps:** lock file maintenance ([#1792](https://github.com/MustardSeedNetworks/niac-go/issues/1792)) ([18918f6](https://github.com/MustardSeedNetworks/niac-go/commit/18918f6f621fc1e141f1b532da54327e7db7f63b))

## [0.94.91](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.90...v0.94.91) (2026-09-04)


> Backfilled by hand. These commits were in the tag but missing from this
> entry: the release PR auto-merged before release-please regenerated it
> (niac#1817).


### Miscellaneous

* assert the port fallback stepped, not which port it landed on ([#1766](https://github.com/MustardSeedNetworks/niac-go/issues/1766)) ([0902539](https://github.com/MustardSeedNetworks/niac-go/commit/0902539d)) — backfilled
* **deps:** update dependency @biomejs/biome to v2.5.11 ([#1720](https://github.com/MustardSeedNetworks/niac-go/issues/1720)) ([ef2e8e5](https://github.com/MustardSeedNetworks/niac-go/commit/ef2e8e5b)) — backfilled


### Features

* **storage:** bound the run history ([#1752](https://github.com/MustardSeedNetworks/niac-go/issues/1752)) ([13dd6bc](https://github.com/MustardSeedNetworks/niac-go/commit/13dd6bc0fa25ae7739390da38da5c014836f60b0)), closes [#1751](https://github.com/MustardSeedNetworks/niac-go/issues/1751)
* **ui:** generate the device editor's forms from the config schema (P1b-2, 1 of 2) ([#1721](https://github.com/MustardSeedNetworks/niac-go/issues/1721)) ([0ba6e16](https://github.com/MustardSeedNetworks/niac-go/commit/0ba6e16383d8cd7b763ad677ee545ba868dbed7c))
* **ui:** the device editor authors the whole schema (P1b-2, 2 of 2) ([#1725](https://github.com/MustardSeedNetworks/niac-go/issues/1725)) ([b0dd0a7](https://github.com/MustardSeedNetworks/niac-go/commit/b0dd0a7dac5b83a014fda07047d768f6ababe983))
* **ui:** the wizard authors a complete network from empty (P1b-3, 2 of 2) ([#1741](https://github.com/MustardSeedNetworks/niac-go/issues/1741)) ([500d75a](https://github.com/MustardSeedNetworks/niac-go/commit/500d75a7594696b7f45b05067e47b3039cc08e9d))


### Bug Fixes

* **api:** step over a reserved port instead of stopping at it ([#1686](https://github.com/MustardSeedNetworks/niac-go/issues/1686)) ([a4045bd](https://github.com/MustardSeedNetworks/niac-go/commit/a4045bd63c5aa1ce382331021681a0f2b086a292)), closes [#1682](https://github.com/MustardSeedNetworks/niac-go/issues/1682)
* **api:** stop one threshold crossing alerting every five seconds ([#1748](https://github.com/MustardSeedNetworks/niac-go/issues/1748)) ([790c394](https://github.com/MustardSeedNetworks/niac-go/commit/790c394865c31fa9efad41f28966913e32274d5e)), closes [#1747](https://github.com/MustardSeedNetworks/niac-go/issues/1747)
* **api:** stop the interfaces resource reporting the selected session (P1-8) ([#1730](https://github.com/MustardSeedNetworks/niac-go/issues/1730)) ([dd47b40](https://github.com/MustardSeedNetworks/niac-go/commit/dd47b40ef3277b9bad1bc6e38265a1ae22ac3117))
* **biome:** bring the config schema back to the installed version ([#1717](https://github.com/MustardSeedNetworks/niac-go/issues/1717)) ([f520b82](https://github.com/MustardSeedNetworks/niac-go/commit/f520b8290d227f69b606fb2fd3c6bb099d6f9f69)), closes [#1716](https://github.com/MustardSeedNetworks/niac-go/issues/1716)
* **cli:** make the command list, version and unknown commands work ([#1746](https://github.com/MustardSeedNetworks/niac-go/issues/1746)) ([b476ee0](https://github.com/MustardSeedNetworks/niac-go/commit/b476ee075ee3292c1fd71a6e42f308b2332f1048)), closes [#1745](https://github.com/MustardSeedNetworks/niac-go/issues/1745) [#1712](https://github.com/MustardSeedNetworks/niac-go/issues/1712)
* **daemon:** honour NIAC_CONFIGS_DIR and stop a session by its own path (P1-16) ([#1728](https://github.com/MustardSeedNetworks/niac-go/issues/1728)) ([55396df](https://github.com/MustardSeedNetworks/niac-go/commit/55396dfd96e735f8ed7fd328fbd06c3b21583461))
* **deps:** golang.org/x/crypto v0.56.0 for GO-2026-6354/6355 ([#1695](https://github.com/MustardSeedNetworks/niac-go/issues/1695)) ([c3f2a1e](https://github.com/MustardSeedNetworks/niac-go/commit/c3f2a1eebe1a69ae284a84e73e1fb2b45c355655))
* **mibzip:** drop the bitwise-OR case and its exhaustive suppression ([#1699](https://github.com/MustardSeedNetworks/niac-go/issues/1699)) ([5add5fe](https://github.com/MustardSeedNetworks/niac-go/commit/5add5fead3e626b16a62b4f64518636ec797206a))
* **ui:** accept every editable device field, and remove the traffic section that never persisted ([#1707](https://github.com/MustardSeedNetworks/niac-go/issues/1707)) ([a66c109](https://github.com/MustardSeedNetworks/niac-go/commit/a66c109f8263da09249abcb6cff5af0e97b7c24a))
* **ui:** the wizard stops leaking drafts and authoring warned-about links ([#1738](https://github.com/MustardSeedNetworks/niac-go/issues/1738)) ([6169a9a](https://github.com/MustardSeedNetworks/niac-go/commit/6169a9a9b97f917ff81ad76209445a7a09178f6d)), closes [#1737](https://github.com/MustardSeedNetworks/niac-go/issues/1737)
* **validate:** run one validator on every authoring surface ([#1711](https://github.com/MustardSeedNetworks/niac-go/issues/1711)) ([57791fc](https://github.com/MustardSeedNetworks/niac-go/commit/57791fc4b1656d19ba3176430ea3b96c78b74c33)), closes [#1710](https://github.com/MustardSeedNetworks/niac-go/issues/1710)
* **version:** stop double-stamping dirty, and correct stale docs ([#1700](https://github.com/MustardSeedNetworks/niac-go/issues/1700)) ([ca6ec84](https://github.com/MustardSeedNetworks/niac-go/commit/ca6ec84a7395a7b3c9b0fd9a29660f2135bfca63))


### Code Refactoring

* delete the dead IPC socket client ([#1756](https://github.com/MustardSeedNetworks/niac-go/issues/1756)) ([b0a5749](https://github.com/MustardSeedNetworks/niac-go/commit/b0a5749604903921e37ef9b4b6b50afc7b71cc4a)), closes [#1754](https://github.com/MustardSeedNetworks/niac-go/issues/1754)
* **niac:** return errors instead of exiting, and cover the IPC client ([#1702](https://github.com/MustardSeedNetworks/niac-go/issues/1702)) ([c6ab8f3](https://github.com/MustardSeedNetworks/niac-go/commit/c6ab8f3dbe06862832a5df5bc4e6e2131a33997f))


### Documentation

* **api:** generate openapi.yaml from the capability registry (P1-7) ([#1724](https://github.com/MustardSeedNetworks/niac-go/issues/1724)) ([8b3202d](https://github.com/MustardSeedNetworks/niac-go/commit/8b3202dfe4c26486ab4b9efa5a7610606d43c830))
* **config:** describe every schema field and ship an authoring guide ([#1736](https://github.com/MustardSeedNetworks/niac-go/issues/1736)) ([f74df56](https://github.com/MustardSeedNetworks/niac-go/commit/f74df56028bcadd9b6187c95586b39ef7a2105a5))


### Tests

* **capture:** assert behavior in the ten no-op tests ([#1697](https://github.com/MustardSeedNetworks/niac-go/issues/1697)) ([ea24bd6](https://github.com/MustardSeedNetworks/niac-go/commit/ea24bd6a5bee25ad471fe2812d4f6428335177a6))
* **scenario:** run the authoring epic's own kill criterion ([#1688](https://github.com/MustardSeedNetworks/niac-go/issues/1688)) ([687467f](https://github.com/MustardSeedNetworks/niac-go/commit/687467f6029b0523769a4cc6a80bc925cd6a124c))
* **wiretest:** assert the simulation on a real wire ([#1690](https://github.com/MustardSeedNetworks/niac-go/issues/1690)) ([93a47d4](https://github.com/MustardSeedNetworks/niac-go/commit/93a47d4460e58074e38bb246d3af1a0863a58954))


### Continuous Integration

* drop the quality job's no-op TODO check ([#1750](https://github.com/MustardSeedNetworks/niac-go/issues/1750)) ([c93f9f0](https://github.com/MustardSeedNetworks/niac-go/commit/c93f9f0a08fc2c8c86de8f52ae0facf74132f5c7)), closes [#1749](https://github.com/MustardSeedNetworks/niac-go/issues/1749)
* give each main commit its own concurrency group ([#1732](https://github.com/MustardSeedNetworks/niac-go/issues/1732)) ([b44ea74](https://github.com/MustardSeedNetworks/niac-go/commit/b44ea74de1af8b4b12041a173d17b383c0bf1419)), closes [#1731](https://github.com/MustardSeedNetworks/niac-go/issues/1731)
* route-consumer and authoring-parity gates ([#1709](https://github.com/MustardSeedNetworks/niac-go/issues/1709)) ([b4a77c9](https://github.com/MustardSeedNetworks/niac-go/commit/b4a77c9fdfebe3bff8374bcb8cc2d44f26b72096)), closes [#1708](https://github.com/MustardSeedNetworks/niac-go/issues/1708)
* wire the shared dependency-review gate into CI ([#1743](https://github.com/MustardSeedNetworks/niac-go/issues/1743)) ([79bac05](https://github.com/MustardSeedNetworks/niac-go/commit/79bac0557cb46bb3579aa240749227480cccd9d8)), closes [#1742](https://github.com/MustardSeedNetworks/niac-go/issues/1742)
* **wire:** run the wire suite from a lab timer, not a self-hosted runner ([#1692](https://github.com/MustardSeedNetworks/niac-go/issues/1692)) ([d755fdb](https://github.com/MustardSeedNetworks/niac-go/commit/d755fdbbda0cea6de351c3af3c745448f3bfe195))


### Miscellaneous

* **api:** close out the route and stream plumbing items ([#1705](https://github.com/MustardSeedNetworks/niac-go/issues/1705)) ([d0eb538](https://github.com/MustardSeedNetworks/niac-go/commit/d0eb53834b92c3de98e0366a6b7fd0bdd717536a))
* **ci:** ratchet the coverage floor to 68 ([#1718](https://github.com/MustardSeedNetworks/niac-go/issues/1718)) ([f060048](https://github.com/MustardSeedNetworks/niac-go/commit/f060048efbcc4e8159668925b2d9901d240734e9))
* **deps:** lock file maintenance ([#1713](https://github.com/MustardSeedNetworks/niac-go/issues/1713)) ([b86fc10](https://github.com/MustardSeedNetworks/niac-go/commit/b86fc10ced9c0772fae318b5fed62f7fee7a10c3))
* **deps:** lock file maintenance ([#1726](https://github.com/MustardSeedNetworks/niac-go/issues/1726)) ([dc2d542](https://github.com/MustardSeedNetworks/niac-go/commit/dc2d5425e01131c9f6461efb580cc7c41841b7e6))
* **deps:** lock file maintenance ([#1734](https://github.com/MustardSeedNetworks/niac-go/issues/1734)) ([b43bf21](https://github.com/MustardSeedNetworks/niac-go/commit/b43bf21d2912f5b0bbde7961acecc92c9612484f))
* **deps:** lock file maintenance ([#1744](https://github.com/MustardSeedNetworks/niac-go/issues/1744)) ([14c0a60](https://github.com/MustardSeedNetworks/niac-go/commit/14c0a6075b34672c9397f64f0216e41b202795d2))
* **deps:** lock file maintenance ([#1755](https://github.com/MustardSeedNetworks/niac-go/issues/1755)) ([1e48bdc](https://github.com/MustardSeedNetworks/niac-go/commit/1e48bdc237334f720969644b8ca02530b0adf5d4))
* **deps:** take the two stale indirect Go modules to latest ([#1715](https://github.com/MustardSeedNetworks/niac-go/issues/1715)) ([c01e584](https://github.com/MustardSeedNetworks/niac-go/commit/c01e584cd6edee83d522e76bfae1455c2b1f368f)), closes [#1714](https://github.com/MustardSeedNetworks/niac-go/issues/1714)
* **deps:** update dependency @biomejs/biome to v2.5.11 ([#1733](https://github.com/MustardSeedNetworks/niac-go/issues/1733)) ([4a710de](https://github.com/MustardSeedNetworks/niac-go/commit/4a710deae8a74255ac6ea7b69e99a67f1a7d1e78))
* **deps:** update dependency @vitejs/plugin-react to v6.1.1 ([#1759](https://github.com/MustardSeedNetworks/niac-go/issues/1759)) ([1de6ec4](https://github.com/MustardSeedNetworks/niac-go/commit/1de6ec4b14ce73040785ea23cbdcadfcb0965fdf))

## [0.94.90](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.89...v0.94.90) (2026-09-02)


### Miscellaneous

* **deps:** lock file maintenance ([#1684](https://github.com/MustardSeedNetworks/niac-go/issues/1684)) ([9b7e366](https://github.com/MustardSeedNetworks/niac-go/commit/9b7e3663b7ed799c90b604a0178d12576e7a2dff))

## [0.94.89](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.88...v0.94.89) (2026-09-02)


### Bug Fixes

* **i18n:** narrow the preserve patterns so dead keys are visible ([#1677](https://github.com/MustardSeedNetworks/niac-go/issues/1677)) ([3e35487](https://github.com/MustardSeedNetworks/niac-go/commit/3e354877af7cdb6dfee8f5687a1c6bf040ba6f49)), closes [#1343](https://github.com/MustardSeedNetworks/niac-go/issues/1343)
* **test:** stop the port-fallback test depending on which port it gets ([#1683](https://github.com/MustardSeedNetworks/niac-go/issues/1683)) ([1839df8](https://github.com/MustardSeedNetworks/niac-go/commit/1839df8b6be658c3fc12a51eafd1364872c43d3e))


### Tests

* **e2e:** cover phone and tablet viewports ([#1680](https://github.com/MustardSeedNetworks/niac-go/issues/1680)) ([6e50348](https://github.com/MustardSeedNetworks/niac-go/commit/6e50348535cfec01c0d9f03e496f15ab21050feb)), closes [#1320](https://github.com/MustardSeedNetworks/niac-go/issues/1320)
* **i18n:** fail the test that renders an unresolved key ([#1678](https://github.com/MustardSeedNetworks/niac-go/issues/1678)) ([0e51344](https://github.com/MustardSeedNetworks/niac-go/commit/0e51344cfc0aca4ebd3c532d06365229a14124ce)), closes [#1669](https://github.com/MustardSeedNetworks/niac-go/issues/1669)
* **protocols:** decode EDP and FDP advertisements end to end ([#1675](https://github.com/MustardSeedNetworks/niac-go/issues/1675)) ([04c1e2a](https://github.com/MustardSeedNetworks/niac-go/commit/04c1e2a69723351b295bb3b34872511a1b71da19)), closes [#1329](https://github.com/MustardSeedNetworks/niac-go/issues/1329)
* **storybook:** run the stories, and let a11y violations fail the build ([#1679](https://github.com/MustardSeedNetworks/niac-go/issues/1679)) ([153a245](https://github.com/MustardSeedNetworks/niac-go/commit/153a2458b1f098b82df4545afacfd0c84377691a)), closes [#1321](https://github.com/MustardSeedNetworks/niac-go/issues/1321) [#1668](https://github.com/MustardSeedNetworks/niac-go/issues/1668)


### Continuous Integration

* compile the windows build target ([#1676](https://github.com/MustardSeedNetworks/niac-go/issues/1676)) ([b9f328a](https://github.com/MustardSeedNetworks/niac-go/commit/b9f328ad95496ad7f6a257d4c19612a141418439)), closes [#1523](https://github.com/MustardSeedNetworks/niac-go/issues/1523)
* pick up the i18n gate that runs extraction ([#1681](https://github.com/MustardSeedNetworks/niac-go/issues/1681)) ([1782b8a](https://github.com/MustardSeedNetworks/niac-go/commit/1782b8a01110f6b1af723f3cabf1e17ae0e4753c)), closes [#1342](https://github.com/MustardSeedNetworks/niac-go/issues/1342)


### Miscellaneous

* four small corrections ([#1673](https://github.com/MustardSeedNetworks/niac-go/issues/1673)) ([bf99eb2](https://github.com/MustardSeedNetworks/niac-go/commit/bf99eb2c2e74b09907aa1c4339a32295d635c472)), closes [#1337](https://github.com/MustardSeedNetworks/niac-go/issues/1337) [#1341](https://github.com/MustardSeedNetworks/niac-go/issues/1341) [#1344](https://github.com/MustardSeedNetworks/niac-go/issues/1344) [#1346](https://github.com/MustardSeedNetworks/niac-go/issues/1346)

## [0.94.88](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.87...v0.94.88) (2026-09-01)


### Miscellaneous

* **deps:** lock file maintenance ([#1671](https://github.com/MustardSeedNetworks/niac-go/issues/1671)) ([d44429d](https://github.com/MustardSeedNetworks/niac-go/commit/d44429df809f74817caeeaf7c0d05a61529cae19))

## [0.94.87](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.86...v0.94.87) (2026-09-01)


### Bug Fixes

* **ci:** make the issue-title gate clear itself and accept the house style ([#1667](https://github.com/MustardSeedNetworks/niac-go/issues/1667)) ([7d6edd9](https://github.com/MustardSeedNetworks/niac-go/commit/7d6edd91ed4eddf0a3e9da27ab01cc10e6cdd915))
* **config:** scope address uniqueness to a segment ([#1665](https://github.com/MustardSeedNetworks/niac-go/issues/1665)) ([4aa5d87](https://github.com/MustardSeedNetworks/niac-go/commit/4aa5d87e8eae31a494f7e5b049ef2dd0f56dd996))
* **device-editor:** serialise the YAML preview instead of concatenating it ([#1660](https://github.com/MustardSeedNetworks/niac-go/issues/1660)) ([9157c4e](https://github.com/MustardSeedNetworks/niac-go/commit/9157c4e6bcb7f1b243770d72c42c21bd24e470d6)), closes [#1607](https://github.com/MustardSeedNetworks/niac-go/issues/1607)
* **device-yaml:** emit the daemon's schema from one serialiser ([#1663](https://github.com/MustardSeedNetworks/niac-go/issues/1663)) ([8a31793](https://github.com/MustardSeedNetworks/niac-go/commit/8a31793931245e4d247180eaf858573e96f626fb)), closes [#1661](https://github.com/MustardSeedNetworks/niac-go/issues/1661)
* **events:** strip one leading space from SSE data fields, not all whitespace ([#1659](https://github.com/MustardSeedNetworks/niac-go/issues/1659)) ([7e8bd4a](https://github.com/MustardSeedNetworks/niac-go/commit/7e8bd4ac5d19cfa7d05a722d881b6596c0d2b8e6)), closes [#1606](https://github.com/MustardSeedNetworks/niac-go/issues/1606)
* **pcap:** show the raw timestamp when a packet time will not parse ([#1657](https://github.com/MustardSeedNetworks/niac-go/issues/1657)) ([1f1cdd1](https://github.com/MustardSeedNetworks/niac-go/commit/1f1cdd1135382e3e8a2e12dbe641cb7c9d986380)), closes [#1604](https://github.com/MustardSeedNetworks/niac-go/issues/1604)
* **topology:** drop the dangling separator when a link speed will not format ([#1658](https://github.com/MustardSeedNetworks/niac-go/issues/1658)) ([751f72b](https://github.com/MustardSeedNetworks/niac-go/commit/751f72b98191992818aadc2018f4cea6b26fe630)), closes [#1605](https://github.com/MustardSeedNetworks/niac-go/issues/1605)
* **topology:** stop parsing a VLAN id as the link speed ([#1656](https://github.com/MustardSeedNetworks/niac-go/issues/1656)) ([9778771](https://github.com/MustardSeedNetworks/niac-go/commit/9778771ed87af8c25b36bf2b424248e715be7e5b)), closes [#1603](https://github.com/MustardSeedNetworks/niac-go/issues/1603)


### Continuous Integration

* never cancel a CI run on main ([#1655](https://github.com/MustardSeedNetworks/niac-go/issues/1655)) ([d656a2d](https://github.com/MustardSeedNetworks/niac-go/commit/d656a2d5037916bd8f28c608c63c1fc2e139b7fb))
* ride out a mirror outage on the release path ([#1666](https://github.com/MustardSeedNetworks/niac-go/issues/1666)) ([2b9ba4e](https://github.com/MustardSeedNetworks/niac-go/commit/2b9ba4e0513d3e12e50d6f16502f407cffa04805)), closes [#1364](https://github.com/MustardSeedNetworks/niac-go/issues/1364)


### Miscellaneous

* **deps:** lock file maintenance ([#1670](https://github.com/MustardSeedNetworks/niac-go/issues/1670)) ([0477741](https://github.com/MustardSeedNetworks/niac-go/commit/04777417e8377df9f1a50a6c000613e046dd4e90))

## [0.94.86](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.85...v0.94.86) (2026-08-31)


### Bug Fixes

* **daemon,config:** validate at the path-building sinks, not only at the handler ([#1627](https://github.com/MustardSeedNetworks/niac-go/issues/1627)) ([6566fae](https://github.com/MustardSeedNetworks/niac-go/commit/6566fae6ae742706d87e48836646cb531bc88aa5)), closes [#1626](https://github.com/MustardSeedNetworks/niac-go/issues/1626)


### Documentation

* reconcile CI.md with the pipeline it describes ([#1651](https://github.com/MustardSeedNetworks/niac-go/issues/1651)) ([11dd32f](https://github.com/MustardSeedNetworks/niac-go/commit/11dd32fde7cc1d08a3c9917215d0cd156566fc30))


### Tests

* **capture:** assert the frames Engine emits via a named handle seam ([#1623](https://github.com/MustardSeedNetworks/niac-go/issues/1623)) ([0fb87f9](https://github.com/MustardSeedNetworks/niac-go/commit/0fb87f94d1cc39bafc3d2163f09df2844b3ba410)), closes [#1622](https://github.com/MustardSeedNetworks/niac-go/issues/1622)
* **capture:** assert the limiter's owned worker instead of a constant ([#1609](https://github.com/MustardSeedNetworks/niac-go/issues/1609)) ([4772bea](https://github.com/MustardSeedNetworks/niac-go/commit/4772bea3a492eb262eace72ca1f037b39ce6a836)), closes [#1608](https://github.com/MustardSeedNetworks/niac-go/issues/1608)
* **capture:** cover the raw-socket boundary in a job that fails closed ([#1625](https://github.com/MustardSeedNetworks/niac-go/issues/1625)) ([7cb5de2](https://github.com/MustardSeedNetworks/niac-go/commit/7cb5de2bf4627065dc2c30748ee8dbd9e6307705)), closes [#1624](https://github.com/MustardSeedNetworks/niac-go/issues/1624)
* **capture:** drive PCAP replay through an injected sender instead of a raw socket ([#1621](https://github.com/MustardSeedNetworks/niac-go/issues/1621)) ([2a79800](https://github.com/MustardSeedNetworks/niac-go/commit/2a798002cb1f5e3ee9d8e4fac8665a37cc417fe9)), closes [#1620](https://github.com/MustardSeedNetworks/niac-go/issues/1620)
* **config:** stop the fuzz helpers swallowing the panics they exist to catch ([#1613](https://github.com/MustardSeedNetworks/niac-go/issues/1613)) ([5e9b457](https://github.com/MustardSeedNetworks/niac-go/commit/5e9b4579a3f0cd34eb7630214466a02abe18564a)), closes [#1612](https://github.com/MustardSeedNetworks/niac-go/issues/1612)
* **e2e:** give the topology contract tests their own topology ([#1616](https://github.com/MustardSeedNetworks/niac-go/issues/1616)) ([6961dc4](https://github.com/MustardSeedNetworks/niac-go/commit/6961dc47f3ebfbc1b4cc14b500925f0be286d3a3)), closes [#1615](https://github.com/MustardSeedNetworks/niac-go/issues/1615)
* **e2e:** run the Firefox engine the docs already promise ([#1638](https://github.com/MustardSeedNetworks/niac-go/issues/1638)) ([cffde13](https://github.com/MustardSeedNetworks/niac-go/commit/cffde1324144899670a2d018cee52adc4c260306)), closes [#1637](https://github.com/MustardSeedNetworks/niac-go/issues/1637)
* **e2e:** synchronise the wizard tab switch on the draft response ([#1647](https://github.com/MustardSeedNetworks/niac-go/issues/1647)) ([a2951ce](https://github.com/MustardSeedNetworks/niac-go/commit/a2951cefc11a105c7b9812dc0b39c802a27440a6))
* turn off Node's unused webstorage global instead of warning per file ([#1611](https://github.com/MustardSeedNetworks/niac-go/issues/1611)) ([5db52d2](https://github.com/MustardSeedNetworks/niac-go/commit/5db52d237790774393dc5b8c3650e3eeb57a5412)), closes [#1610](https://github.com/MustardSeedNetworks/niac-go/issues/1610)


### Continuous Integration

* fail the build on open High CodeQL alerts ([#1630](https://github.com/MustardSeedNetworks/niac-go/issues/1630)) ([5855775](https://github.com/MustardSeedNetworks/niac-go/commit/585577511a457826d6c47dc31181f2d292ed8edb)), closes [#1629](https://github.com/MustardSeedNetworks/niac-go/issues/1629)
* make release builds reproducible ([#1653](https://github.com/MustardSeedNetworks/niac-go/issues/1653)) ([36d8683](https://github.com/MustardSeedNetworks/niac-go/commit/36d86836b2c4349e2b39e5cb05225c25c7dec266))
* pin depcheck in the dead-code job ([#1636](https://github.com/MustardSeedNetworks/niac-go/issues/1636)) ([d9d272d](https://github.com/MustardSeedNetworks/niac-go/commit/d9d272d206d4fe59391eed7aa47b2f031f6416fe)), closes [#1635](https://github.com/MustardSeedNetworks/niac-go/issues/1635)
* run Build and E2E on backend-only changes ([#1632](https://github.com/MustardSeedNetworks/niac-go/issues/1632)) ([3c99789](https://github.com/MustardSeedNetworks/niac-go/commit/3c9978971cb7744014dd85504257c1ff2253e375)), closes [#1631](https://github.com/MustardSeedNetworks/niac-go/issues/1631)
* stop checkout persisting credentials into the workspace ([#1634](https://github.com/MustardSeedNetworks/niac-go/issues/1634)) ([723e753](https://github.com/MustardSeedNetworks/niac-go/commit/723e7536b0adc52d60d2b247d63a63e33ea53cd7)), closes [#1633](https://github.com/MustardSeedNetworks/niac-go/issues/1633)


### Miscellaneous

* **deps:** lock file maintenance ([#1617](https://github.com/MustardSeedNetworks/niac-go/issues/1617)) ([e6f1c43](https://github.com/MustardSeedNetworks/niac-go/commit/e6f1c437b72fe3086d8d2c3b21275a0267ca4745))
* **deps:** lock file maintenance ([#1618](https://github.com/MustardSeedNetworks/niac-go/issues/1618)) ([0680673](https://github.com/MustardSeedNetworks/niac-go/commit/0680673de18556fac69cc330faa90f4f1299db13))
* **deps:** lock file maintenance ([#1639](https://github.com/MustardSeedNetworks/niac-go/issues/1639)) ([b3585f6](https://github.com/MustardSeedNetworks/niac-go/commit/b3585f63df05792dc0f50813bf0ee18fc697c727))
* **deps:** lock file maintenance ([#1643](https://github.com/MustardSeedNetworks/niac-go/issues/1643)) ([d279cd0](https://github.com/MustardSeedNetworks/niac-go/commit/d279cd0b4c366ae5899c3961599e72b9012cbfa8))
* **deps:** lock file maintenance ([#1645](https://github.com/MustardSeedNetworks/niac-go/issues/1645)) ([a77523f](https://github.com/MustardSeedNetworks/niac-go/commit/a77523f1547d4224e6bc903826e395fbefa981e9))
* **deps:** update dependency @biomejs/biome to v2.5.11 ([#1644](https://github.com/MustardSeedNetworks/niac-go/issues/1644)) ([120b539](https://github.com/MustardSeedNetworks/niac-go/commit/120b539c2a3260ef7b03202a00a2e766283337a8))
* **deps:** update dependency @testing-library/react to v16.3.3 ([#1640](https://github.com/MustardSeedNetworks/niac-go/issues/1640)) ([bc868f2](https://github.com/MustardSeedNetworks/niac-go/commit/bc868f295c3f8bb4758daf5e939052fec09325d2))
* **deps:** update dependency @types/node to v26.4.0 ([#1619](https://github.com/MustardSeedNetworks/niac-go/issues/1619)) ([35c0c5d](https://github.com/MustardSeedNetworks/niac-go/commit/35c0c5df53d9b8527bed3856ce47539b9fcfa2a7))
* **deps:** update dependency lint-staged to v17.4.1 ([#1628](https://github.com/MustardSeedNetworks/niac-go/issues/1628)) ([9fffe2d](https://github.com/MustardSeedNetworks/niac-go/commit/9fffe2dc9244f573c25294c6037d3f6abed497f5))
* **npm:** soak package releases for seven days before resolving them ([#1642](https://github.com/MustardSeedNetworks/niac-go/issues/1642)) ([e7901da](https://github.com/MustardSeedNetworks/niac-go/commit/e7901dacc4796180258ac9c0181a8814d439c739))
* stop tracking the linklive-acceptance binary ([#1649](https://github.com/MustardSeedNetworks/niac-go/issues/1649)) ([636a183](https://github.com/MustardSeedNetworks/niac-go/commit/636a183151f136be7cf7c4aa44b1bdb2729fd60b))

## [0.94.85](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.84...v0.94.85) (2026-08-29)


### Tests

* **ui:** restore branch coverage toward the anti-regression floors ([#1602](https://github.com/MustardSeedNetworks/niac-go/issues/1602)) ([378cb16](https://github.com/MustardSeedNetworks/niac-go/commit/378cb16a96fa6c389d1475820faced51c4326c83)), closes [#1595](https://github.com/MustardSeedNetworks/niac-go/issues/1595)


### Continuous Integration

* drop the Codecov upload, keep the local coverage gate ([#1600](https://github.com/MustardSeedNetworks/niac-go/issues/1600)) ([f1e2257](https://github.com/MustardSeedNetworks/niac-go/commit/f1e2257ad4ed4f63f6ed9bdd5f6cdb76d8fb25e6)), closes [#1599](https://github.com/MustardSeedNetworks/niac-go/issues/1599)

## [0.94.84](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.83...v0.94.84) (2026-08-28)


### Continuous Integration

* pass codecov the input name it actually accepts ([#1596](https://github.com/MustardSeedNetworks/niac-go/issues/1596)) ([3e5fca0](https://github.com/MustardSeedNetworks/niac-go/commit/3e5fca0c6a3b40d66c4ddffcf2ea0bb0afb19267)), closes [#1595](https://github.com/MustardSeedNetworks/niac-go/issues/1595)

## [0.94.83](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.82...v0.94.83) (2026-08-28)


### Bug Fixes

* **deps:** update dependency @xyflow/react to v12.11.5 ([#1591](https://github.com/MustardSeedNetworks/niac-go/issues/1591)) ([3384ea6](https://github.com/MustardSeedNetworks/niac-go/commit/3384ea6e178d5ff8594565cdaea07a709632b8f0))


### Miscellaneous

* **deps:** lock file maintenance ([#1592](https://github.com/MustardSeedNetworks/niac-go/issues/1592)) ([eeb82a8](https://github.com/MustardSeedNetworks/niac-go/commit/eeb82a84db780c5111ffe920ff2b462b59f57114))
* **deps:** update dependency i18next-cli to v1.71.3 ([#1590](https://github.com/MustardSeedNetworks/niac-go/issues/1590)) ([ba33573](https://github.com/MustardSeedNetworks/niac-go/commit/ba33573e111b98ead64d218e9255e1aff5195e87))

## [0.94.82](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.81...v0.94.82) (2026-08-28)


### Miscellaneous

* **deps:** lock file maintenance ([#1585](https://github.com/MustardSeedNetworks/niac-go/issues/1585)) ([f4e28f9](https://github.com/MustardSeedNetworks/niac-go/commit/f4e28f9f6012260f5d84ad5841e83bc6d6fa3dcb))
* **deps:** lock file maintenance ([#1588](https://github.com/MustardSeedNetworks/niac-go/issues/1588)) ([b39c81b](https://github.com/MustardSeedNetworks/niac-go/commit/b39c81b18ca6844f9ed78fd23f991b6136e426e8))
* **deps:** lock file maintenance ([#1589](https://github.com/MustardSeedNetworks/niac-go/issues/1589)) ([d608a34](https://github.com/MustardSeedNetworks/niac-go/commit/d608a34eeb027801b2e6421306282592f2d4bf67))
* **deps:** update dependency i18next-cli to v1.71.2 ([#1587](https://github.com/MustardSeedNetworks/niac-go/issues/1587)) ([b505ca4](https://github.com/MustardSeedNetworks/niac-go/commit/b505ca418866fa3e3edc0508cf1d88a956ca3b19))

## [0.94.81](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.80...v0.94.81) (2026-08-28)


### Bug Fixes

* **e2e:** stop behavior-timeline deadlocking on a failed route assertion ([#1582](https://github.com/MustardSeedNetworks/niac-go/issues/1582)) ([df06475](https://github.com/MustardSeedNetworks/niac-go/commit/df06475c55fe208beeca82a40c101e24db5882da)), closes [#1581](https://github.com/MustardSeedNetworks/niac-go/issues/1581)


### Continuous Integration

* use the App Client ID instead of the deprecated app-id input ([#1577](https://github.com/MustardSeedNetworks/niac-go/issues/1577)) ([10ce530](https://github.com/MustardSeedNetworks/niac-go/commit/10ce53094eff90b7f3f2eb5dd8f63201d32a56e5))

## [0.94.80](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.79...v0.94.80) (2026-08-28)


### Miscellaneous

* **deps:** lock file maintenance ([#1572](https://github.com/MustardSeedNetworks/niac-go/issues/1572)) ([be4ac2a](https://github.com/MustardSeedNetworks/niac-go/commit/be4ac2a72ca9aee61bbaf01e20b2981e27bcdd28))
* **deps:** lock file maintenance ([#1579](https://github.com/MustardSeedNetworks/niac-go/issues/1579)) ([f468260](https://github.com/MustardSeedNetworks/niac-go/commit/f468260eab867df2ad943001e03fb70f8fed7b52))
* **deps:** update dependency @types/node to v26.3.0 ([#1571](https://github.com/MustardSeedNetworks/niac-go/issues/1571)) ([74e1d07](https://github.com/MustardSeedNetworks/niac-go/commit/74e1d07d6497977b6551d85907645afa5702c8bc))
* **deps:** update dependency i18next-cli to v1.71.1 ([#1570](https://github.com/MustardSeedNetworks/niac-go/issues/1570)) ([b529735](https://github.com/MustardSeedNetworks/niac-go/commit/b5297359adb219aed37e54fc46ce57f1261f6246))
* **deps:** update github actions ([#1576](https://github.com/MustardSeedNetworks/niac-go/issues/1576)) ([3f489be](https://github.com/MustardSeedNetworks/niac-go/commit/3f489bef3a56337e38bc44b080692e50e2ec1884))
* **deps:** update module github.com/golangci/golangci-lint/v2/cmd/golangci-lint to v2.13.2 ([#1575](https://github.com/MustardSeedNetworks/niac-go/issues/1575)) ([dba90ca](https://github.com/MustardSeedNetworks/niac-go/commit/dba90caf34534cd2fa6f100519b952c185d602b5))

## [0.94.79](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.78...v0.94.79) (2026-08-27)


### Continuous Integration

* adopt shared-workflow v1.8.0 ([#1563](https://github.com/MustardSeedNetworks/niac-go/issues/1563)) ([39c85f4](https://github.com/MustardSeedNetworks/niac-go/commit/39c85f45a354f0ae32ce7a5fd486bf0db0d29134))
* make .nvmrc the only source for the Node version ([#1565](https://github.com/MustardSeedNetworks/niac-go/issues/1565)) ([a36fc2e](https://github.com/MustardSeedNetworks/niac-go/commit/a36fc2eba4607f6d4238f0bcdff759d0231217fa))
* make gate implementations trigger the gates they implement ([#1560](https://github.com/MustardSeedNetworks/niac-go/issues/1560)) ([c5f34b9](https://github.com/MustardSeedNetworks/niac-go/commit/c5f34b91d8b545141e2c79ecc2e0be222e7ca843))
* refuse to release a tag whose commit never passed CI ([#1567](https://github.com/MustardSeedNetworks/niac-go/issues/1567)) ([3692051](https://github.com/MustardSeedNetworks/niac-go/commit/3692051780c12d6faf8d71ffe36cede4d9bb58a4))
* save the Go cache after compiling, cancel superseded runs, pin the macOS target ([#1569](https://github.com/MustardSeedNetworks/niac-go/issues/1569)) ([f410ff3](https://github.com/MustardSeedNetworks/niac-go/commit/f410ff34756ef21fa6e8787e330668cb9935d85e))


### Miscellaneous

* **deps:** lock file maintenance ([#1558](https://github.com/MustardSeedNetworks/niac-go/issues/1558)) ([8d5f7c9](https://github.com/MustardSeedNetworks/niac-go/commit/8d5f7c9adb4461ab4965e32ce9f77dd57a9ed0b4))

## [0.94.78](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.77...v0.94.78) (2026-08-27)


### Bug Fixes

* **deps:** update dependency lucide-react to v1.34.0 ([#1556](https://github.com/MustardSeedNetworks/niac-go/issues/1556)) ([285f1be](https://github.com/MustardSeedNetworks/niac-go/commit/285f1be3ca84cf21b9de6abeb25cc0969416681e))

## [0.94.77](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.76...v0.94.77) (2026-08-27)


### Continuous Integration

* **dead-code:** make one step actually gate, on reachability ([#1548](https://github.com/MustardSeedNetworks/niac-go/issues/1548)) ([90fb757](https://github.com/MustardSeedNetworks/niac-go/commit/90fb757310306fd9b58a47918ff0dfccad074b1a))


### Miscellaneous

* **deps:** lock file maintenance ([#1546](https://github.com/MustardSeedNetworks/niac-go/issues/1546)) ([3c17960](https://github.com/MustardSeedNetworks/niac-go/commit/3c17960362ce70a9f5537f959c1ac07fe71107c2))
* **deps:** lock file maintenance ([#1553](https://github.com/MustardSeedNetworks/niac-go/issues/1553)) ([3a1da9e](https://github.com/MustardSeedNetworks/niac-go/commit/3a1da9e8967eeaf3969a787908d957ad6882191f))
* **deps:** lock file maintenance ([#1554](https://github.com/MustardSeedNetworks/niac-go/issues/1554)) ([42fd8af](https://github.com/MustardSeedNetworks/niac-go/commit/42fd8af76c998c4ed154f522a35fc923664e24ba))
* **deps:** lock file maintenance ([#1555](https://github.com/MustardSeedNetworks/niac-go/issues/1555)) ([3585b45](https://github.com/MustardSeedNetworks/niac-go/commit/3585b4564bf4ee6df0ce56ae3cfd541ea9ddca5c))
* **deps:** update dependency @types/react-dom to v19.2.5 ([#1551](https://github.com/MustardSeedNetworks/niac-go/issues/1551)) ([d760ac9](https://github.com/MustardSeedNetworks/niac-go/commit/d760ac9c38dea1180d2c0374ad3dd2e3282c4d0e))
* **deps:** update module github.com/securego/gosec/v2/cmd/gosec to v2.29.0 ([#1545](https://github.com/MustardSeedNetworks/niac-go/issues/1545)) ([f04284a](https://github.com/MustardSeedNetworks/niac-go/commit/f04284a1c56e682fda17302d4289f1d155812b0f))
* **deps:** update node.js to v26.8.0 ([#1549](https://github.com/MustardSeedNetworks/niac-go/issues/1549)) ([c1594e1](https://github.com/MustardSeedNetworks/niac-go/commit/c1594e129ac5eb4b0f9c20f9d37d0fdfa599c2d7))
* **deps:** update node.js to v26.8.1 ([#1552](https://github.com/MustardSeedNetworks/niac-go/issues/1552)) ([faff9fe](https://github.com/MustardSeedNetworks/niac-go/commit/faff9fef4bc147e63118f3126f6e81c16fc3d873))

## [0.94.76](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.75...v0.94.76) (2026-08-26)


### Bug Fixes

* **api:** detect an in-use port by the Winsock errno on Windows ([#1543](https://github.com/MustardSeedNetworks/niac-go/issues/1543)) ([548b58e](https://github.com/MustardSeedNetworks/niac-go/commit/548b58e611edf57d349bc810520a4dce550937eb))

## [0.94.75](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.74...v0.94.75) (2026-08-26)


### Bug Fixes

* **ci:** request the workflows scope for the release-please token ([#1541](https://github.com/MustardSeedNetworks/niac-go/issues/1541)) ([f5bb1e5](https://github.com/MustardSeedNetworks/niac-go/commit/f5bb1e505fb15d07e14bee99216f5bd728ac3842))
* **deps:** update dependency react-hook-form to v7.86.0 ([#1530](https://github.com/MustardSeedNetworks/niac-go/issues/1530)) ([1c02c6a](https://github.com/MustardSeedNetworks/niac-go/commit/1c02c6aba3e19ef55dc9e191f01ab10f1d41aa7a))
* **deps:** update module github.com/mustardseednetworks/foundation to v0.5.2 ([#1529](https://github.com/MustardSeedNetworks/niac-go/issues/1529)) ([6660da9](https://github.com/MustardSeedNetworks/niac-go/commit/6660da955f46a7f0086dee59ff858bd52a4843a1))
* **i18n:** fail the key gate on any finding, and see &lt;Trans&gt; keys ([#1532](https://github.com/MustardSeedNetworks/niac-go/issues/1532)) ([b331af9](https://github.com/MustardSeedNetworks/niac-go/commit/b331af945d47d7ba01a5e49a98c7a71fb11a6614)), closes [#1531](https://github.com/MustardSeedNetworks/niac-go/issues/1531)
* **i18n:** localize the last 35 hardcoded strings and make the check block ([#1528](https://github.com/MustardSeedNetworks/niac-go/issues/1528)) ([6a7556c](https://github.com/MustardSeedNetworks/niac-go/commit/6a7556cd728037b60701cdb3024e5b1724ab9810))
* **i18n:** semgrep for the fallback rules, and drop --ratchet ([#1525](https://github.com/MustardSeedNetworks/niac-go/issues/1525)) ([fc9295a](https://github.com/MustardSeedNetworks/niac-go/commit/fc9295afb972bf6f12e84215a3348a3d68cf0ad3))


### Miscellaneous

* **deps:** lock file maintenance ([#1536](https://github.com/MustardSeedNetworks/niac-go/issues/1536)) ([1a739ca](https://github.com/MustardSeedNetworks/niac-go/commit/1a739cace9c8a48a3bf177c95e695093dbd36a3e))
* **deps:** lock file maintenance ([#1538](https://github.com/MustardSeedNetworks/niac-go/issues/1538)) ([c7e2e6c](https://github.com/MustardSeedNetworks/niac-go/commit/c7e2e6c2340af0ba5b41c20b24f72ca4f99f92cd))
* **deps:** lock file maintenance ([#1539](https://github.com/MustardSeedNetworks/niac-go/issues/1539)) ([8fc47d3](https://github.com/MustardSeedNetworks/niac-go/commit/8fc47d3fb79c4292033f4bff5f38860da629b907))
* **deps:** update dependency @testing-library/user-event to v14.6.6 ([#1535](https://github.com/MustardSeedNetworks/niac-go/issues/1535)) ([8459bf0](https://github.com/MustardSeedNetworks/niac-go/commit/8459bf03a880f795782fe1360978fb7daccc1f09))
* **i18n:** adopt the shared gate from MustardSeedNetworks/.github ([#1534](https://github.com/MustardSeedNetworks/niac-go/issues/1534)) ([0c557be](https://github.com/MustardSeedNetworks/niac-go/commit/0c557be89c48c356065fadab5e9553c9c7b0d7ea)), closes [#1533](https://github.com/MustardSeedNetworks/niac-go/issues/1533)

## [0.94.74](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.73...v0.94.74) (2026-08-24)


### Continuous Integration

* compile the darwin build on a macOS runner ([#1521](https://github.com/MustardSeedNetworks/niac-go/issues/1521)) ([a916e77](https://github.com/MustardSeedNetworks/niac-go/commit/a916e77cec2d033f4ea8545747fa617701f56fb9))


### Miscellaneous

* **deps:** lock file maintenance ([#1522](https://github.com/MustardSeedNetworks/niac-go/issues/1522)) ([b1d6339](https://github.com/MustardSeedNetworks/niac-go/commit/b1d633985f51769d55f017ecda95909615ee26a9))

## [0.94.73](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.72...v0.94.73) (2026-08-24)


### Miscellaneous

* **deps:** update dependency @biomejs/biome to v2.5.10 ([#1517](https://github.com/MustardSeedNetworks/niac-go/issues/1517)) ([7696e90](https://github.com/MustardSeedNetworks/niac-go/commit/7696e907de7e30faa39f99517339a4d7e7d942d7))

## [0.94.72](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.71...v0.94.72) (2026-08-24)


### Bug Fixes

* **i18n:** assert exact pins instead of a hardcoded version target ([#1513](https://github.com/MustardSeedNetworks/niac-go/issues/1513)) ([f73c4de](https://github.com/MustardSeedNetworks/niac-go/commit/f73c4de30a21c5f686a43923c374651436c9fd32))

## [0.94.71](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.70...v0.94.71) (2026-08-24)


### Miscellaneous

* **deps:** lock file maintenance ([#1509](https://github.com/MustardSeedNetworks/niac-go/issues/1509)) ([7b38eb3](https://github.com/MustardSeedNetworks/niac-go/commit/7b38eb341bea7aaf16bb2352caec7b18b7fa5d25))

## [0.94.70](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.69...v0.94.70) (2026-08-24)


### Bug Fixes

* **capture:** require payload before a port names an application protocol ([#1507](https://github.com/MustardSeedNetworks/niac-go/issues/1507)) ([9262e7a](https://github.com/MustardSeedNetworks/niac-go/commit/9262e7a78a245f09806e354dde9c4bf294bfa3e7)), closes [#1506](https://github.com/MustardSeedNetworks/niac-go/issues/1506)

## [0.94.69](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.68...v0.94.69) (2026-08-23)


### Miscellaneous

* **deps:** move to Go 1.27.0 ([#1504](https://github.com/MustardSeedNetworks/niac-go/issues/1504)) ([57422a8](https://github.com/MustardSeedNetworks/niac-go/commit/57422a8d9bfb12a330976b97299a84ad10d76081))

## [0.94.68](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.67...v0.94.68) (2026-08-23)


### Documentation

* **design:** record the completed Wave 8 sweep ([#1502](https://github.com/MustardSeedNetworks/niac-go/issues/1502)) ([34eb786](https://github.com/MustardSeedNetworks/niac-go/commit/34eb786de972bad2ffacedc167d3cf4ac7a34dd4))

## [0.94.67](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.66...v0.94.67) (2026-08-23)


### Bug Fixes

* **ui:** render server error details wherever an error is shown inline ([#1500](https://github.com/MustardSeedNetworks/niac-go/issues/1500)) ([2f7f961](https://github.com/MustardSeedNetworks/niac-go/commit/2f7f9619b168625d0e2800104f06413a81be6864)), closes [#1499](https://github.com/MustardSeedNetworks/niac-go/issues/1499)

## [0.94.66](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.65...v0.94.66) (2026-08-23)


### Bug Fixes

* **pcap:** show packet addresses and let application protocols be followed ([#1495](https://github.com/MustardSeedNetworks/niac-go/issues/1495)) ([4d6b073](https://github.com/MustardSeedNetworks/niac-go/commit/4d6b073912314e85311ea75f8f72d5689660edcc)), closes [#1497](https://github.com/MustardSeedNetworks/niac-go/issues/1497)

## [0.94.65](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.64...v0.94.65) (2026-08-23)


### Bug Fixes

* **api:** say why a walk capture failed ([#1489](https://github.com/MustardSeedNetworks/niac-go/issues/1489)) ([fb8a387](https://github.com/MustardSeedNetworks/niac-go/commit/fb8a3877ff99112eb134066f42f2dc59d18de0fb)), closes [#1488](https://github.com/MustardSeedNetworks/niac-go/issues/1488)
* **ui:** enable Behaviors from any device that has an interface ([#1492](https://github.com/MustardSeedNetworks/niac-go/issues/1492)) ([c9656e5](https://github.com/MustardSeedNetworks/niac-go/commit/c9656e56072380ec6d0faa98ae961f3c19bae500)), closes [#1491](https://github.com/MustardSeedNetworks/niac-go/issues/1491)

## [0.94.64](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.63...v0.94.64) (2026-08-23)


### Bug Fixes

* **ui:** stop pages scrolling sideways at phone and tablet widths ([#1487](https://github.com/MustardSeedNetworks/niac-go/issues/1487)) ([386fd1c](https://github.com/MustardSeedNetworks/niac-go/commit/386fd1c96a1909da979253bfc754744ebe72098e)), closes [#1483](https://github.com/MustardSeedNetworks/niac-go/issues/1483)

## [0.94.63](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.62...v0.94.63) (2026-08-23)


### Bug Fixes

* **deps:** update dependency i18next to v26.4.0 ([#1479](https://github.com/MustardSeedNetworks/niac-go/issues/1479)) ([79dc98a](https://github.com/MustardSeedNetworks/niac-go/commit/79dc98a137e9e36a36d64755d64fd31e3f437ab9))
* **deps:** update dependency react-i18next to v17.0.12 ([#1478](https://github.com/MustardSeedNetworks/niac-go/issues/1478)) ([9f0a6b1](https://github.com/MustardSeedNetworks/niac-go/commit/9f0a6b10f93a9f9eef3ce197e99cdcd414c3489a))
* **ui:** accept the interface names error injection's own dropdown offers ([#1480](https://github.com/MustardSeedNetworks/niac-go/issues/1480)) ([98f8168](https://github.com/MustardSeedNetworks/niac-go/commit/98f8168ef77dc88fa9dbd32228f1a2c799ac6728)), closes [#1476](https://github.com/MustardSeedNetworks/niac-go/issues/1476)
* **ui:** make the packet filter chips toggle and combine with OR ([#1482](https://github.com/MustardSeedNetworks/niac-go/issues/1482)) ([71b6fac](https://github.com/MustardSeedNetworks/niac-go/commit/71b6fac4fd20eb38f3c8cdd2cf97f3621009ef9b)), closes [#1481](https://github.com/MustardSeedNetworks/niac-go/issues/1481)


### Miscellaneous

* **deps:** lock file maintenance ([#1484](https://github.com/MustardSeedNetworks/niac-go/issues/1484)) ([63e72ed](https://github.com/MustardSeedNetworks/niac-go/commit/63e72ed72adc0cb60f7533e791ede6788cea48da))
* **deps:** lock file maintenance ([#1486](https://github.com/MustardSeedNetworks/niac-go/issues/1486)) ([55621e8](https://github.com/MustardSeedNetworks/niac-go/commit/55621e8b5b9294aca2efae0984474a1a850351e5))
* **deps:** update dependency @vitejs/plugin-react to v6.1.0 ([#1443](https://github.com/MustardSeedNetworks/niac-go/issues/1443)) ([6b8cef0](https://github.com/MustardSeedNetworks/niac-go/commit/6b8cef0458d8b6a54609ee6dfeec7d59dada00ac))
* **deps:** update storybook monorepo to v10.5.10 ([#1477](https://github.com/MustardSeedNetworks/niac-go/issues/1477)) ([b117033](https://github.com/MustardSeedNetworks/niac-go/commit/b117033d3e4fea059a07f9057dcc68e603c55a37))

## [0.94.62](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.61...v0.94.62) (2026-08-23)


### Documentation

* **design:** update the closeout for v0.94.61 and the eighth follow-up ([#1474](https://github.com/MustardSeedNetworks/niac-go/issues/1474)) ([46275f4](https://github.com/MustardSeedNetworks/niac-go/commit/46275f41fa8599c470b460cabcff7a2633aeb1f2))

## [0.94.61](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.60...v0.94.61) (2026-08-23)


### Bug Fixes

* **ui:** show the preflight details the server sends ([#1473](https://github.com/MustardSeedNetworks/niac-go/issues/1473)) ([812bd4d](https://github.com/MustardSeedNetworks/niac-go/commit/812bd4d2fe353290acc7f320b40312cfb8f918f8)), closes [#1472](https://github.com/MustardSeedNetworks/niac-go/issues/1472)


### Documentation

* **design:** record the UI defect remediation closeout ([#1470](https://github.com/MustardSeedNetworks/niac-go/issues/1470)) ([d91526b](https://github.com/MustardSeedNetworks/niac-go/commit/d91526b3a4daeeb56c765247209484dfa7b962b4))

## [0.94.60](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.59...v0.94.60) (2026-08-23)


### Features

* **daemon:** let a native (untagged) session run alongside tagged ones ([#1454](https://github.com/MustardSeedNetworks/niac-go/issues/1454)) ([f097216](https://github.com/MustardSeedNetworks/niac-go/commit/f097216c4688a194532fe0a34c421c472e06c3d0)), closes [#1426](https://github.com/MustardSeedNetworks/niac-go/issues/1426) [#1437](https://github.com/MustardSeedNetworks/niac-go/issues/1437)


### Bug Fixes

* **api:** complete the ADR-0007 camelCase migration on the client ([#1441](https://github.com/MustardSeedNetworks/niac-go/issues/1441)) ([69e15ae](https://github.com/MustardSeedNetworks/niac-go/commit/69e15ae9003b6fb55822889ef48cdb59d6e41ebf))
* **api:** group the segments view by per-device VLAN ([#1452](https://github.com/MustardSeedNetworks/niac-go/issues/1452)) ([8c1ea2c](https://github.com/MustardSeedNetworks/niac-go/commit/8c1ea2c20e53ca8da71254995b27c223bf3206bd)), closes [#1425](https://github.com/MustardSeedNetworks/niac-go/issues/1425) [#1437](https://github.com/MustardSeedNetworks/niac-go/issues/1437)
* **config:** make wizard drafts startable and preflight failures legible ([#1464](https://github.com/MustardSeedNetworks/niac-go/issues/1464)) ([da6b4df](https://github.com/MustardSeedNetworks/niac-go/commit/da6b4dfbf5d80e3c4ea0a9e60dae7107b6d4ff36)), closes [#1460](https://github.com/MustardSeedNetworks/niac-go/issues/1460) [#1461](https://github.com/MustardSeedNetworks/niac-go/issues/1461) [#1463](https://github.com/MustardSeedNetworks/niac-go/issues/1463)
* **daemon:** report device count from the running stack, not the start-time snapshot ([#1447](https://github.com/MustardSeedNetworks/niac-go/issues/1447)) ([9afd7cd](https://github.com/MustardSeedNetworks/niac-go/commit/9afd7cdabfed90190c8050b404c7629451e72fd3)), closes [#1423](https://github.com/MustardSeedNetworks/niac-go/issues/1423) [#1437](https://github.com/MustardSeedNetworks/niac-go/issues/1437)
* **fabric:** marshal empty topology collections as [] rather than null ([#1445](https://github.com/MustardSeedNetworks/niac-go/issues/1445)) ([ceb9d85](https://github.com/MustardSeedNetworks/niac-go/commit/ceb9d8501b44b493a15b5aaeea4ee844ca639e92)), closes [#1421](https://github.com/MustardSeedNetworks/niac-go/issues/1421) [#1437](https://github.com/MustardSeedNetworks/niac-go/issues/1437)
* **fabric:** return arrays, not null, from every preflight path ([#1468](https://github.com/MustardSeedNetworks/niac-go/issues/1468)) ([b863efc](https://github.com/MustardSeedNetworks/niac-go/commit/b863efc5a57c9156443963e9eb3fed6e27749a7a)), closes [#1467](https://github.com/MustardSeedNetworks/niac-go/issues/1467)
* **library:** ship one validated template tree instead of three ([#1456](https://github.com/MustardSeedNetworks/niac-go/issues/1456)) ([b2ffce3](https://github.com/MustardSeedNetworks/niac-go/commit/b2ffce34530efd9d9ac486a8fd49e407285c1d41)), closes [#1428](https://github.com/MustardSeedNetworks/niac-go/issues/1428) [#1437](https://github.com/MustardSeedNetworks/niac-go/issues/1437)
* **packets:** stop the inspector reporting a zero arrival time and a phantom IPv4 layer ([#1459](https://github.com/MustardSeedNetworks/niac-go/issues/1459)) ([e84331e](https://github.com/MustardSeedNetworks/niac-go/commit/e84331e6a0e1e53381a91ed01faff5cce42e2e3a)), closes [#1457](https://github.com/MustardSeedNetworks/niac-go/issues/1457) [#1458](https://github.com/MustardSeedNetworks/niac-go/issues/1458)
* **sse:** decode tagged L2 control frames instead of reporting Unknown ([#1455](https://github.com/MustardSeedNetworks/niac-go/issues/1455)) ([4a95cda](https://github.com/MustardSeedNetworks/niac-go/commit/4a95cdada9e547cec656e4c366936f766101f959)), closes [#1427](https://github.com/MustardSeedNetworks/niac-go/issues/1427) [#1437](https://github.com/MustardSeedNetworks/niac-go/issues/1437)
* surface the reasons the API already knows ([#1449](https://github.com/MustardSeedNetworks/niac-go/issues/1449)) ([724aedf](https://github.com/MustardSeedNetworks/niac-go/commit/724aedfa83e8604b2f17ed2b630e01441b108c4a)), closes [#1429](https://github.com/MustardSeedNetworks/niac-go/issues/1429) [#1430](https://github.com/MustardSeedNetworks/niac-go/issues/1430) [#1431](https://github.com/MustardSeedNetworks/niac-go/issues/1431) [#1437](https://github.com/MustardSeedNetworks/niac-go/issues/1437)
* **ui:** derive About versions from the manifest and explain the disabled timeline button ([#1450](https://github.com/MustardSeedNetworks/niac-go/issues/1450)) ([71593d6](https://github.com/MustardSeedNetworks/niac-go/commit/71593d6b2c06b87a9067c0377f8178842950f616))
* **ui:** let Escape reach the focus trap so modals actually close ([#1466](https://github.com/MustardSeedNetworks/niac-go/issues/1466)) ([19da728](https://github.com/MustardSeedNetworks/niac-go/commit/19da72810d00ef84f65262e6220adfab19036030)), closes [#1465](https://github.com/MustardSeedNetworks/niac-go/issues/1465)
* **ui:** load the device into the editor from the flat API response ([#1446](https://github.com/MustardSeedNetworks/niac-go/issues/1446)) ([74c6768](https://github.com/MustardSeedNetworks/niac-go/commit/74c676885bb057790d9402857dae12cd4506b060)), closes [#1422](https://github.com/MustardSeedNetworks/niac-go/issues/1422) [#1437](https://github.com/MustardSeedNetworks/niac-go/issues/1437)
* **ui:** make clipped help tabs and the topology export menu reachable ([#1448](https://github.com/MustardSeedNetworks/niac-go/issues/1448)) ([aa5acb6](https://github.com/MustardSeedNetworks/niac-go/commit/aa5acb6c6e6ce5b586fcb606fe0b64dfbea66775)), closes [#1432](https://github.com/MustardSeedNetworks/niac-go/issues/1432) [#1433](https://github.com/MustardSeedNetworks/niac-go/issues/1433) [#1434](https://github.com/MustardSeedNetworks/niac-go/issues/1434) [#1437](https://github.com/MustardSeedNetworks/niac-go/issues/1437)
* **ui:** paint modal dialogs above their own backdrop ([#1442](https://github.com/MustardSeedNetworks/niac-go/issues/1442)) ([0e306e7](https://github.com/MustardSeedNetworks/niac-go/commit/0e306e7205293f797364eeba0eadfd11d74281bd)), closes [#1419](https://github.com/MustardSeedNetworks/niac-go/issues/1419) [#1420](https://github.com/MustardSeedNetworks/niac-go/issues/1420) [#1437](https://github.com/MustardSeedNetworks/niac-go/issues/1437)
* **ui:** read the protocols the device API already returns ([#1451](https://github.com/MustardSeedNetworks/niac-go/issues/1451)) ([9846cb5](https://github.com/MustardSeedNetworks/niac-go/commit/9846cb5c0090fe01200ebe337030daa260f13928)), closes [#1424](https://github.com/MustardSeedNetworks/niac-go/issues/1424) [#1437](https://github.com/MustardSeedNetworks/niac-go/issues/1437)


### Miscellaneous

* **deps:** update dependency vite to v8.2.2 ([#1453](https://github.com/MustardSeedNetworks/niac-go/issues/1453)) ([846cca2](https://github.com/MustardSeedNetworks/niac-go/commit/846cca28266a419de836248f1ad203fbea50bf42))

## [0.94.59](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.58...v0.94.59) (2026-08-23)


### Documentation

* **design:** add UI defect remediation plan (19 defects, 8 waves) ([#1438](https://github.com/MustardSeedNetworks/niac-go/issues/1438)) ([a81dfb6](https://github.com/MustardSeedNetworks/niac-go/commit/a81dfb6fc421b096b1b8b80345e564052642f07e))

## [0.94.58](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.57...v0.94.58) (2026-08-22)


### Miscellaneous

* **deps:** lock file maintenance ([#1416](https://github.com/MustardSeedNetworks/niac-go/issues/1416)) ([965c1de](https://github.com/MustardSeedNetworks/niac-go/commit/965c1de84072df487f0b3fb12a2b6ef30c338317))

## [0.94.57](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.56...v0.94.57) (2026-08-22)


### Continuous Integration

* serialise Release Please so concurrent runs stop racing the ref ([#1414](https://github.com/MustardSeedNetworks/niac-go/issues/1414)) ([86ff65c](https://github.com/MustardSeedNetworks/niac-go/commit/86ff65c5e0600577e6381ee35d257ffd3e97c52a))

## [0.94.56](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.55...v0.94.56) (2026-08-22)


### Bug Fixes

* **deps:** update dependency lucide-react to v1.33.0 ([#1405](https://github.com/MustardSeedNetworks/niac-go/issues/1405)) ([5aa3f8e](https://github.com/MustardSeedNetworks/niac-go/commit/5aa3f8e0645f16e85d0911e8aac26ac5a5a520fb))

## [0.94.55](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.54...v0.94.55) (2026-08-22)


### Bug Fixes

* **deps:** update dependency react-i18next to v17.0.12 ([#1402](https://github.com/MustardSeedNetworks/niac-go/issues/1402)) ([7802c92](https://github.com/MustardSeedNetworks/niac-go/commit/7802c9289c3fcec0fa6ce9d460f59e4c20d59faf))

## [0.94.54](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.53...v0.94.54) (2026-08-22)


### Bug Fixes

* **deps:** update dependency i18next to v26.4.0 ([#1404](https://github.com/MustardSeedNetworks/niac-go/issues/1404)) ([669af9f](https://github.com/MustardSeedNetworks/niac-go/commit/669af9f2e05627e1dad21bcc5875d47dd9293614))
* **deps:** update dependency immer to v11.1.18 ([#1401](https://github.com/MustardSeedNetworks/niac-go/issues/1401)) ([e5d33f4](https://github.com/MustardSeedNetworks/niac-go/commit/e5d33f49aff5436194ffca18fd60bd0ae5f65907))
* **deps:** update dependency react-hook-form to v7.86.0 ([#1406](https://github.com/MustardSeedNetworks/niac-go/issues/1406)) ([8b8429a](https://github.com/MustardSeedNetworks/niac-go/commit/8b8429ac995f76215b2f03a5a2a51ac391923f60))


### Miscellaneous

* **deps:** update dependency @biomejs/biome to v2.5.10 ([#1398](https://github.com/MustardSeedNetworks/niac-go/issues/1398)) ([fa45259](https://github.com/MustardSeedNetworks/niac-go/commit/fa45259b70ce949a45d137d7092dd450931b6f75))
* **deps:** update dependency @vitejs/plugin-react to v6.1.0 ([#1403](https://github.com/MustardSeedNetworks/niac-go/issues/1403)) ([a0e670f](https://github.com/MustardSeedNetworks/niac-go/commit/a0e670f8ea1c1731220f35f821a7f67977a08ebb))
* **deps:** update frontend toolchain ([#1399](https://github.com/MustardSeedNetworks/niac-go/issues/1399)) ([0c75d09](https://github.com/MustardSeedNetworks/niac-go/commit/0c75d096296d811f1ce5ed996fd1780139c47b28))
* **deps:** update storybook monorepo to v10.5.10 ([#1400](https://github.com/MustardSeedNetworks/niac-go/issues/1400)) ([fb9affe](https://github.com/MustardSeedNetworks/niac-go/commit/fb9affec034a9ac51679bdafeb968162b97e0811))

## [0.94.53](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.52...v0.94.53) (2026-08-22)


### Bug Fixes

* **deps:** let npm regenerate the lockfile, which unblocks Renovate ([#1396](https://github.com/MustardSeedNetworks/niac-go/issues/1396)) ([8d80b3b](https://github.com/MustardSeedNetworks/niac-go/commit/8d80b3b460665d1b58ea32283b50a4c9067514b0))

## [0.94.52](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.51...v0.94.52) (2026-08-22)


### Continuous Integration

* exempt bots from the issue-title lint ([#1393](https://github.com/MustardSeedNetworks/niac-go/issues/1393)) ([2531ed5](https://github.com/MustardSeedNetworks/niac-go/commit/2531ed569e3934e7c754c3b30cabe473c2209970))


### Miscellaneous

* **deps:** update dependency @biomejs/biome to v2.5.9 ([#1394](https://github.com/MustardSeedNetworks/niac-go/issues/1394)) ([846ce4b](https://github.com/MustardSeedNetworks/niac-go/commit/846ce4b7880ff6a19ba529515053cf40fccf8adf))

## [0.94.51](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.50...v0.94.51) (2026-08-21)


### Miscellaneous

* **deps:** update github actions ([#1367](https://github.com/MustardSeedNetworks/niac-go/issues/1367)) ([2efe236](https://github.com/MustardSeedNetworks/niac-go/commit/2efe236aabda334977d24a34a61555d996dace49))
* **deps:** update module honnef.co/go/tools to v0.8.1 ([#1389](https://github.com/MustardSeedNetworks/niac-go/issues/1389)) ([727108a](https://github.com/MustardSeedNetworks/niac-go/commit/727108a9ad1e39e13a8b7db90883b967c9d3eb8f))

## [0.94.50](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.49...v0.94.50) (2026-08-21)


### Bug Fixes

* **deps:** update dependency @hookform/resolvers to v5.9.1 ([#1373](https://github.com/MustardSeedNetworks/niac-go/issues/1373)) ([04dd4f8](https://github.com/MustardSeedNetworks/niac-go/commit/04dd4f84072a22b904f65c099754a8174a552d71))
* **deps:** update dependency lucide-react to v1.32.0 ([#1383](https://github.com/MustardSeedNetworks/niac-go/issues/1383)) ([c9f8df2](https://github.com/MustardSeedNetworks/niac-go/commit/c9f8df22c42ab516257efdfeda93b445ab7dda39))


### Miscellaneous

* **deps:** update dependency @testing-library/user-event to v14.6.5 ([#1263](https://github.com/MustardSeedNetworks/niac-go/issues/1263)) ([59f4071](https://github.com/MustardSeedNetworks/niac-go/commit/59f40712f8976c344cc83f72085633b3e0c61545))
* **deps:** update dependency i18next-cli to v1.71.0 ([#1381](https://github.com/MustardSeedNetworks/niac-go/issues/1381)) ([9e8f38a](https://github.com/MustardSeedNetworks/niac-go/commit/9e8f38a53b3156b93ca2a13ca85cf48818551088))

## [0.94.49](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.48...v0.94.49) (2026-08-21)


### Continuous Integration

* refuse to start tests while orphaned test binaries are running ([#1384](https://github.com/MustardSeedNetworks/niac-go/issues/1384)) ([c1f8197](https://github.com/MustardSeedNetworks/niac-go/commit/c1f81977d5337646e1fb408ed9d5a20698a8a631))

## [0.94.48](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.47...v0.94.48) (2026-08-21)


### Features

* **devices:** open the selected device's own config in the detail pane ([#1380](https://github.com/MustardSeedNetworks/niac-go/issues/1380)) ([bf16750](https://github.com/MustardSeedNetworks/niac-go/commit/bf167505f75a8f6ec1fa5c952fbf27a8bedd64e2))


### Miscellaneous

* **deps:** update module github.com/golangci/golangci-lint/v2/cmd/golangci-lint to v2.13.1 ([#1375](https://github.com/MustardSeedNetworks/niac-go/issues/1375)) ([3379b7c](https://github.com/MustardSeedNetworks/niac-go/commit/3379b7cdbf1a5a3d4e2c65facf0dba3b5b5a0cb8))
* **ts:** converge the strictness flags and gate the contract ([#1377](https://github.com/MustardSeedNetworks/niac-go/issues/1377)) ([08fa699](https://github.com/MustardSeedNetworks/niac-go/commit/08fa699d8e0fed719548546143e0cab7c9b6f1ed))

## [0.94.47](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.46...v0.94.47) (2026-08-20)


### Features

* **acceptance:** record the provenance a pinned result needs to be evidence ([#1360](https://github.com/MustardSeedNetworks/niac-go/issues/1360)) ([bf32992](https://github.com/MustardSeedNetworks/niac-go/commit/bf32992ccce1a3ec2cfa9a11e08a7d72e590e26b))
* **scenario:** bump the manifest to v4 with observations SEED can assert ([#1363](https://github.com/MustardSeedNetworks/niac-go/issues/1363)) ([ee12ed4](https://github.com/MustardSeedNetworks/niac-go/commit/ee12ed45c7338ceded80f19d441216b7b8e0ed8f))
* **topology:** draw the archetype's core-to-access tier bands ([#1365](https://github.com/MustardSeedNetworks/niac-go/issues/1365)) ([4d8780a](https://github.com/MustardSeedNetworks/niac-go/commit/4d8780a42c718875a64314076293d15be2018dc5))


### Bug Fixes

* **deps:** bump golang.org/x/mod to v0.40.0 for CVE-2026-56864/56865 ([#1372](https://github.com/MustardSeedNetworks/niac-go/issues/1372)) ([36af8b4](https://github.com/MustardSeedNetworks/niac-go/commit/36af8b403e1cee103bc5713f3e56fbfe6e3266da)), closes [#1371](https://github.com/MustardSeedNetworks/niac-go/issues/1371)
* **deps:** update module charm.land/bubbletea/v2 to v2.0.9 ([#1366](https://github.com/MustardSeedNetworks/niac-go/issues/1366)) ([ed6092e](https://github.com/MustardSeedNetworks/niac-go/commit/ed6092ed3bc2624ca19d9510e5331687de8a429c))
* **lint:** scope Biome's dist, build and coverage excludes to where they land ([#1369](https://github.com/MustardSeedNetworks/niac-go/issues/1369)) ([2746c8c](https://github.com/MustardSeedNetworks/niac-go/commit/2746c8cd3d145d7892ba49ef3cc44273e2724e9c))


### Miscellaneous

* **deps:** update module github.com/golangci/golangci-lint/v2/cmd/golangci-lint to v2.13.0 ([#1368](https://github.com/MustardSeedNetworks/niac-go/issues/1368)) ([39fcfb5](https://github.com/MustardSeedNetworks/niac-go/commit/39fcfb56e6280940ca3ac0390fcc70a8032ffb49))
* **ts:** turn on noUncheckedIndexedAccess across niac's UI ([#1374](https://github.com/MustardSeedNetworks/niac-go/issues/1374)) ([60792c7](https://github.com/MustardSeedNetworks/niac-go/commit/60792c75b6c63c5578bb481c0763bd2416040ad2)), closes [#1338](https://github.com/MustardSeedNetworks/niac-go/issues/1338)

## [0.94.46](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.45...v0.94.46) (2026-08-19)


### Bug Fixes

* **topology:** stop reporting a device status the daemon never measured ([#1354](https://github.com/MustardSeedNetworks/niac-go/issues/1354)) ([3df44e5](https://github.com/MustardSeedNetworks/niac-go/commit/3df44e5990fbe000ec28793ccb1a8f34e4beb221))


### Miscellaneous

* **deps:** lock file maintenance ([#1355](https://github.com/MustardSeedNetworks/niac-go/issues/1355)) ([0a2a351](https://github.com/MustardSeedNetworks/niac-go/commit/0a2a351afef01f36ee87704eb6781fd85cc7296d))

## [0.94.45](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.44...v0.94.45) (2026-08-18)


### Continuous Integration

* reconcile skipped releases with a 3-hourly release-please run ([#1351](https://github.com/MustardSeedNetworks/niac-go/issues/1351)) ([91ab871](https://github.com/MustardSeedNetworks/niac-go/commit/91ab87174638f2dfd8f0f967cfc24011134c5bb3))

## [0.94.44](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.43...v0.94.44) (2026-08-18)


### Features

* **ui:** add the Inspector archetype and converge the two packet inspectors ([#1348](https://github.com/MustardSeedNetworks/niac-go/issues/1348)) ([f0241ff](https://github.com/MustardSeedNetworks/niac-go/commit/f0241ff74853a3c3fbe0bd300dadc80083ae2f1f))


### Bug Fixes

* **ci:** size job budgets to their apt steps' worst case ([#1350](https://github.com/MustardSeedNetworks/niac-go/issues/1350)) ([5b6aafa](https://github.com/MustardSeedNetworks/niac-go/commit/5b6aafacf87194604a34a2fdc8d70845e8e3c1cf))

## [0.94.43](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.42...v0.94.43) (2026-08-18)


### Code Refactoring

* **ui:** settle the status vocabulary and stop bypassing StatusConfig ([#1345](https://github.com/MustardSeedNetworks/niac-go/issues/1345)) ([6acedbe](https://github.com/MustardSeedNetworks/niac-go/commit/6acedbeb39d134e63b80268ae4cc5a194bbbdf98))

## [0.94.42](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.41...v0.94.42) (2026-08-18)


### Features

* **ui:** move page help into the drawer's Pages tab ([#1339](https://github.com/MustardSeedNetworks/niac-go/issues/1339)) ([527a034](https://github.com/MustardSeedNetworks/niac-go/commit/527a0342a1d2e611290fb0da2bf02f9f3ce39021))

## [0.94.41](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.40...v0.94.41) (2026-08-17)


### Bug Fixes

* **protocols:** correct EDP wire format and mirror FDP address TLV on CDP ([#1335](https://github.com/MustardSeedNetworks/niac-go/issues/1335)) ([f984d0c](https://github.com/MustardSeedNetworks/niac-go/commit/f984d0cf7780ea0b98697620f547dd053dd3bbfe))

## [0.94.40](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.39...v0.94.40) (2026-08-17)


### Bug Fixes

* **protocols:** correct LLDP and CDP management address encodings ([#1332](https://github.com/MustardSeedNetworks/niac-go/issues/1332)) ([d7c16a4](https://github.com/MustardSeedNetworks/niac-go/commit/d7c16a4dcbc4ec8f24c231e52f157e14197b0355)), closes [#1330](https://github.com/MustardSeedNetworks/niac-go/issues/1330)
* **protocols:** emit a conforming CDP Address TLV protocol type ([#1326](https://github.com/MustardSeedNetworks/niac-go/issues/1326)) ([8983b5e](https://github.com/MustardSeedNetworks/niac-go/commit/8983b5eb7dc496d8abf9aa5c99d0e31507864017))


### Continuous Integration

* make CI conformance a blocking gate ([#1327](https://github.com/MustardSeedNetworks/niac-go/issues/1327)) ([109626e](https://github.com/MustardSeedNetworks/niac-go/commit/109626ec935fac62cfebf61777d53a359d49b3dd))


### Miscellaneous

* **deps:** lock file maintenance ([#1324](https://github.com/MustardSeedNetworks/niac-go/issues/1324)) ([824bd1c](https://github.com/MustardSeedNetworks/niac-go/commit/824bd1c55363815b323ab03af4d3fb20a077fa09))
* remove Lighthouse residue left by its deletion ([#1334](https://github.com/MustardSeedNetworks/niac-go/issues/1334)) ([720907f](https://github.com/MustardSeedNetworks/niac-go/commit/720907f05df47c9bd7e80ecad3717199408c7ee0)), closes [#1314](https://github.com/MustardSeedNetworks/niac-go/issues/1314)
* **ui:** remove stale license remnants ([#1331](https://github.com/MustardSeedNetworks/niac-go/issues/1331)) ([ef5f15d](https://github.com/MustardSeedNetworks/niac-go/commit/ef5f15d2d360101e50187a91a5af562d9d79e80d))

## [0.94.39](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.38...v0.94.39) (2026-08-17)


### Tests

* **e2e:** run chromium and webkit only ([#1318](https://github.com/MustardSeedNetworks/niac-go/issues/1318)) ([5c40201](https://github.com/MustardSeedNetworks/niac-go/commit/5c40201fbf08f63a45e95027d2168d612e348a12))


### Continuous Integration

* drop Lighthouse ([#1315](https://github.com/MustardSeedNetworks/niac-go/issues/1315)) ([3a469e6](https://github.com/MustardSeedNetworks/niac-go/commit/3a469e62fc79677d9f84eb8348f01e1fe9eb37d3))


### Miscellaneous

* **deps:** update module github.com/google/go-licenses to v2 ([#1313](https://github.com/MustardSeedNetworks/niac-go/issues/1313)) ([6af5f55](https://github.com/MustardSeedNetworks/niac-go/commit/6af5f55d8f4632059b3f344b2b2f033dd342bf97))

## [0.94.38](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.37...v0.94.38) (2026-08-16)


### Bug Fixes

* **ci:** give markdown lint a base ref in the merge queue ([#1304](https://github.com/MustardSeedNetworks/niac-go/issues/1304)) ([aab13ba](https://github.com/MustardSeedNetworks/niac-go/commit/aab13ba717a63ff7fe0658a73f468a11320b7306))


### Miscellaneous

* make revive doc-comment gate real, fix nolintlint findings ([#1308](https://github.com/MustardSeedNetworks/niac-go/issues/1308)) ([ba83130](https://github.com/MustardSeedNetworks/niac-go/commit/ba8313035727a5150ad9970d82d881f5dcdfbd96))

## [0.94.37](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.36...v0.94.37) (2026-08-16)


### Bug Fixes

* **release:** generate SBOMs for .deb and .rpm ([#1299](https://github.com/MustardSeedNetworks/niac-go/issues/1299)) ([e16ae30](https://github.com/MustardSeedNetworks/niac-go/commit/e16ae30be672a25bdcf9d7413299a02a3e48b723))


### Continuous Integration

* make required checks report on merge_group ([#1302](https://github.com/MustardSeedNetworks/niac-go/issues/1302)) ([9a70f1e](https://github.com/MustardSeedNetworks/niac-go/commit/9a70f1e5d71ff71e9df163a89ebb4536e49a09ec))
* stop PRs writing their own cache copies ([#1297](https://github.com/MustardSeedNetworks/niac-go/issues/1297)) ([2b41521](https://github.com/MustardSeedNetworks/niac-go/commit/2b415219e41a80f2be923e4749da9683ea5a3821))

## [0.94.36](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.35...v0.94.36) (2026-08-16)


### Bug Fixes

* **ci:** make the i18n gate actually gate ([#1286](https://github.com/MustardSeedNetworks/niac-go/issues/1286)) ([ffbe751](https://github.com/MustardSeedNetworks/niac-go/commit/ffbe7519408da78f1284732e47c3844fe9ce27b4))


### Continuous Integration

* consolidate duplicated apt-install blocks, right-size gates ([#1288](https://github.com/MustardSeedNetworks/niac-go/issues/1288)) ([4cd043d](https://github.com/MustardSeedNetworks/niac-go/commit/4cd043dbcfb70fe7aae5bf3bea3bfafcd45aa9f3))
* declare which jobs deliberately do not gate a merge ([#1295](https://github.com/MustardSeedNetworks/niac-go/issues/1295)) ([7000aaf](https://github.com/MustardSeedNetworks/niac-go/commit/7000aaf93ac5513b2b64d516e533eb69ca1adf7c))


### Miscellaneous

* **ci:** migrate to the shared fleet apt-install composite ([#1293](https://github.com/MustardSeedNetworks/niac-go/issues/1293)) ([d34f2c2](https://github.com/MustardSeedNetworks/niac-go/commit/d34f2c2773c25d126cfaf5690af223f9a7036215))
* **release:** drop the no-op trigger-release job ([#1291](https://github.com/MustardSeedNetworks/niac-go/issues/1291)) ([c0dc828](https://github.com/MustardSeedNetworks/niac-go/commit/c0dc82858eceac394c20f7ffe928ab31ebd55c38))

## [0.94.35](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.34...v0.94.35) (2026-08-16)


### Bug Fixes

* **ci:** bound apt waits in Playwright OS-deps step ([#1279](https://github.com/MustardSeedNetworks/niac-go/issues/1279)) ([bd686c8](https://github.com/MustardSeedNetworks/niac-go/commit/bd686c80473d10fe3156ab17e5f25e82da68a399))
* **ci:** bound apt waits in remaining install steps ([#1284](https://github.com/MustardSeedNetworks/niac-go/issues/1284)) ([a7f6274](https://github.com/MustardSeedNetworks/niac-go/commit/a7f6274f7ae27c53014aedd3caf28ad5617c0454))

## [0.94.34](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.33...v0.94.34) (2026-08-15)


### Bug Fixes

* **ci:** exempt bot PRs from the required PR-body check ([#1258](https://github.com/MustardSeedNetworks/niac-go/issues/1258)) ([40572fc](https://github.com/MustardSeedNetworks/niac-go/commit/40572fc757c942c52a253df7e39b684b06a0f699))
* **release:** stop requesting an App permission that is not granted ([#1275](https://github.com/MustardSeedNetworks/niac-go/issues/1275)) ([04be703](https://github.com/MustardSeedNetworks/niac-go/commit/04be7038d98e91927c43ff17bf294d2578ae98ff))


### Continuous Integration

* cache Playwright browsers and document the CI pipeline ([#1254](https://github.com/MustardSeedNetworks/niac-go/issues/1254)) ([3d0ac13](https://github.com/MustardSeedNetworks/niac-go/commit/3d0ac13b4fec881eae778c5f73c82fe0fcdc6842))
* pin Node via the setup-node composite everywhere ([#1256](https://github.com/MustardSeedNetworks/niac-go/issues/1256)) ([ef54556](https://github.com/MustardSeedNetworks/niac-go/commit/ef54556829a760b0e1a8ff60c3fd4f1dea2e0f6e))
* scope workflow permissions to jobs and narrow the release-please App token ([#1253](https://github.com/MustardSeedNetworks/niac-go/issues/1253)) ([1b684dd](https://github.com/MustardSeedNetworks/niac-go/commit/1b684dd183d768912cea92ba1e96d64a28e2d933))
* split race tests, verify UIBuildHash, add missing gates ([#1251](https://github.com/MustardSeedNetworks/niac-go/issues/1251)) ([97c8a8c](https://github.com/MustardSeedNetworks/niac-go/commit/97c8a8c7f8c125e3f68b4fcd1154f1014cffb7ab))


### Miscellaneous

* **deps:** bump remaining outdated dependencies to latest ([#1274](https://github.com/MustardSeedNetworks/niac-go/issues/1274)) ([63c9832](https://github.com/MustardSeedNetworks/niac-go/commit/63c983213ad4ba9cc211ea0a8a6ae0d1128b0b66)), closes [#1273](https://github.com/MustardSeedNetworks/niac-go/issues/1273)
* **deps:** take dependencies to latest and adopt TypeScript 7 ([#1270](https://github.com/MustardSeedNetworks/niac-go/issues/1270)) ([9fc602d](https://github.com/MustardSeedNetworks/niac-go/commit/9fc602d41f5a10b14dcca3c55f69a9c5971a1098))
* **deps:** update dependency lint-staged to v17.3.0 ([#1271](https://github.com/MustardSeedNetworks/niac-go/issues/1271)) ([5aa26cc](https://github.com/MustardSeedNetworks/niac-go/commit/5aa26cc5a280fdbfedec94ace71d432e54f68834))

## [0.94.33](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.32...v0.94.33) (2026-08-14)


### Bug Fixes

* **icmpv6:** send the reply body, not just the ICMPv6 header ([#1249](https://github.com/MustardSeedNetworks/niac-go/issues/1249)) ([a9beceb](https://github.com/MustardSeedNetworks/niac-go/commit/a9beceb31b143cba2870d674536ce91ee8eae63a))

## [0.94.32](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.31...v0.94.32) (2026-08-14)


### Bug Fixes

* **icmpv6:** answer neighbor discovery, and answer it on the right VLAN ([#1247](https://github.com/MustardSeedNetworks/niac-go/issues/1247)) ([a6d7628](https://github.com/MustardSeedNetworks/niac-go/commit/a6d7628798c54e19e109ed0feb828cfd71060f93))

## [0.94.31](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.30...v0.94.31) (2026-08-14)


### Features

* **lab:** prove the six packs are actually isolated ([#1231](https://github.com/MustardSeedNetworks/niac-go/issues/1231)) ([54924c5](https://github.com/MustardSeedNetworks/niac-go/commit/54924c5ec0c4da0e4094e2dfd05a622a626454e4))
* **linklive-acceptance:** make every report say what it compared ([#1229](https://github.com/MustardSeedNetworks/niac-go/issues/1229)) ([10bd4b4](https://github.com/MustardSeedNetworks/niac-go/commit/10bd4b4293640437efb10bef8e8bedf8a53b17ee))
* **scenario:** distinct warehouse and manufacturing shapes, and the AP utilization rule ([#1222](https://github.com/MustardSeedNetworks/niac-go/issues/1222)) ([a9c45fb](https://github.com/MustardSeedNetworks/niac-go/commit/a9c45fbc42883ed7bef0d84bb3bc1aedc83c5158))
* **scenario:** give campus, retail, and the provider POPs their own shape ([#1225](https://github.com/MustardSeedNetworks/niac-go/issues/1225)) ([f71c7bd](https://github.com/MustardSeedNetworks/niac-go/commit/f71c7bd7936bb53febc7f85fd007d1d2680816fc))
* **scenario:** give the hospital demo something to find ([#1227](https://github.com/MustardSeedNetworks/niac-go/issues/1227)) ([d84ed4f](https://github.com/MustardSeedNetworks/niac-go/commit/d84ed4f3b3dcc59f3fc14d5e29df4e0776c79a3d))
* **snmp:** answer SNMP queries over IPv6 ([#1244](https://github.com/MustardSeedNetworks/niac-go/issues/1244)) ([51acfcc](https://github.com/MustardSeedNetworks/niac-go/commit/51acfcc0a8fad24704619bd87c686aa542e90b26))


### Bug Fixes

* **acceptance:** expect the warning a pack deliberately authors ([#1228](https://github.com/MustardSeedNetworks/niac-go/issues/1228)) ([399f1be](https://github.com/MustardSeedNetworks/niac-go/commit/399f1be6d54da79ad9297d0f44c7febac5cb7d48))
* **api:** stop the device editor throwing away the fields it just showed you ([#1235](https://github.com/MustardSeedNetworks/niac-go/issues/1235)) ([4d30ae8](https://github.com/MustardSeedNetworks/niac-go/commit/4d30ae803da04f8a226286d47ad9978dbacb3ca7))
* **cli:** make niac monitor work, and report real numbers ([#1240](https://github.com/MustardSeedNetworks/niac-go/issues/1240)) ([4bc7316](https://github.com/MustardSeedNetworks/niac-go/commit/4bc73164da377f75f50358e0fc0f40fc7fd5eb73))
* **cli:** reconnect the last five commands to the daemon ([#1241](https://github.com/MustardSeedNetworks/niac-go/issues/1241)) ([b3314ed](https://github.com/MustardSeedNetworks/niac-go/commit/b3314ed7eafe6219953d19250f50ff27b5321555))
* **config:** refuse a second capture playback instead of deleting it ([#1236](https://github.com/MustardSeedNetworks/niac-go/issues/1236)) ([4dab243](https://github.com/MustardSeedNetworks/niac-go/commit/4dab243d00e51e158132c2ce095d6a124cb13c15))
* **deps:** move the nanoid pin to 3.3.18 to clear GHSA-2v37-7h3g-55p8 again ([#1242](https://github.com/MustardSeedNetworks/niac-go/issues/1242)) ([db3876e](https://github.com/MustardSeedNetworks/niac-go/commit/db3876efaa406256c5b15e7822e92ac8df7f8a84))
* **lab:** stop the isolation check reporting a leak that is not there ([#1234](https://github.com/MustardSeedNetworks/niac-go/issues/1234)) ([a4d6d49](https://github.com/MustardSeedNetworks/niac-go/commit/a4d6d49c05b8052c3bf28721e9afc46951e9f78f))
* **library:** report bundle-installed content as bundle content ([#1243](https://github.com/MustardSeedNetworks/niac-go/issues/1243)) ([1e1a007](https://github.com/MustardSeedNetworks/niac-go/commit/1e1a007c396f595e4382559cea270d57cbcc2e1b))
* **logs:** make --filter mean the same thing with and without --follow ([#1237](https://github.com/MustardSeedNetworks/niac-go/issues/1237)) ([1bc3aa2](https://github.com/MustardSeedNetworks/niac-go/commit/1bc3aa2df696f7f07690bd7cd68c9344e7104981))
* **security:** bump Go to 1.26.6 for five reachable stdlib CVEs ([#1246](https://github.com/MustardSeedNetworks/niac-go/issues/1246)) ([421976b](https://github.com/MustardSeedNetworks/niac-go/commit/421976b2003d4208d3fcbf93427a1e17bca58285))
* **snmp:** refuse a SET to an object the agent serves read-only ([#1233](https://github.com/MustardSeedNetworks/niac-go/issues/1233)) ([96cfd16](https://github.com/MustardSeedNetworks/niac-go/commit/96cfd1695e38569841aa67fb28136a16e52efecc))


### Documentation

* publish the scale baseline for the 531-device workload ([#1230](https://github.com/MustardSeedNetworks/niac-go/issues/1230)) ([1b5b9a9](https://github.com/MustardSeedNetworks/niac-go/commit/1b5b9a98d79070f0b1f3906e055800bd5bb8fb12))
* record the hospital story evidence and the limit on its error half ([#1232](https://github.com/MustardSeedNetworks/niac-go/issues/1232)) ([159464f](https://github.com/MustardSeedNetworks/niac-go/commit/159464f61634cee101e5fdd593431240dc75c39b))
* record the M4 acceptance results and the two capture preconditions ([#1226](https://github.com/MustardSeedNetworks/niac-go/issues/1226)) ([8f498c1](https://github.com/MustardSeedNetworks/niac-go/commit/8f498c1a2a1c77392c5509a08c20da4b2873de1a))


### Continuous Integration

* exempt bot PRs from the human change-PR body template ([#1245](https://github.com/MustardSeedNetworks/niac-go/issues/1245)) ([5ecf273](https://github.com/MustardSeedNetworks/niac-go/commit/5ecf2738d3efccdd1599af1499c82ae96e23725f))

## [0.94.30](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.29...v0.94.30) (2026-08-08)


### Documentation

* record how the lab is actually driven and what M4-1 settled ([#1221](https://github.com/MustardSeedNetworks/niac-go/issues/1221)) ([a328369](https://github.com/MustardSeedNetworks/niac-go/commit/a328369236914633bfcf1c9482c873ce055315af))


### Miscellaneous

* **deps-dev:** bump js-yaml ([#1210](https://github.com/MustardSeedNetworks/niac-go/issues/1210)) ([8379c04](https://github.com/MustardSeedNetworks/niac-go/commit/8379c04b21f1179aa89d4f9b71d2700aee28e4c9))

## [0.94.29](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.28...v0.94.29) (2026-08-08)


### Features

* **mdns:** let devices announce themselves on multicast DNS ([#1215](https://github.com/MustardSeedNetworks/niac-go/issues/1215)) ([294d4ce](https://github.com/MustardSeedNetworks/niac-go/commit/294d4ce60aefd9c6bc5a0aba0c45259df7009519))
* **scenario:** split endpoints by device class so hosts are not managed gear ([#1217](https://github.com/MustardSeedNetworks/niac-go/issues/1217)) ([cdc18b5](https://github.com/MustardSeedNetworks/niac-go/commit/cdc18b5fd844618a5a0fd6ce9ac496b02e868c84))


### Bug Fixes

* **acceptance:** stop the comparator expecting what Link-Live never reports ([#1219](https://github.com/MustardSeedNetworks/niac-go/issues/1219)) ([9c3a2b9](https://github.com/MustardSeedNetworks/niac-go/commit/9c3a2b9f6d111ab0c3f697b983dc1f13cf6fc86c))
* **deps:** pin nanoid to 3.3.17 to clear GHSA-2v37-7h3g-55p8 ([#1216](https://github.com/MustardSeedNetworks/niac-go/issues/1216)) ([1778b24](https://github.com/MustardSeedNetworks/niac-go/commit/1778b24f5b721297e7dc9a47a5fa8ed67d35c0d5))
* **netbios:** answer node status requests so endpoints are not anonymous ([#1214](https://github.com/MustardSeedNetworks/niac-go/issues/1214)) ([e10732b](https://github.com/MustardSeedNetworks/niac-go/commit/e10732b3f8b2da165ca0aedf0776c72e72ee79f0))
* **netbios:** answer the port the query came from ([#1218](https://github.com/MustardSeedNetworks/niac-go/issues/1218)) ([0967131](https://github.com/MustardSeedNetworks/niac-go/commit/09671312dedb41ff9a77e1732e27f12d95d9bb8f))
* **scenario:** report endpoint sysDescr the way a real agent would ([f7075c9](https://github.com/MustardSeedNetworks/niac-go/commit/f7075c963962970729ed088c5561c996a3bc5d98))


### Documentation

* decide vertical device realism and where imitation stops ([a7a0f02](https://github.com/MustardSeedNetworks/niac-go/commit/a7a0f0262b49fb31c08b2c05dd249ac005879f3a))
* modernise the topology spine and add the server and storage tier ([2fd5a2d](https://github.com/MustardSeedNetworks/niac-go/commit/2fd5a2d1ba321c911aa98ab40007e18979dba78b))

## [0.94.28](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.27...v0.94.28) (2026-08-07)


### Features

* **api:** address one simulation session explicitly ([0dcc6b4](https://github.com/MustardSeedNetworks/niac-go/commit/0dcc6b41b1c17a7af767f9c2e947b4c875e8a42e))
* **behavior,devicestate:** make timeline replay deterministic and checkpoints complete ([050dfab](https://github.com/MustardSeedNetworks/niac-go/commit/050dfab7aff93e66da64d2815dc8da41d14e0617))
* **daemon,api,ui:** report trunk capture health and degrade its sessions ([92a2166](https://github.com/MustardSeedNetworks/niac-go/commit/92a2166572c4782bcdb94e2d788a6210550fc956))
* **daemon:** enforce device and session budgets across the whole daemon ([5630a85](https://github.com/MustardSeedNetworks/niac-go/commit/5630a8544a6265a08fe20b47351549fe593bc41e))
* **devicestate:** add a link-down fault, and only that one ([156c2f0](https://github.com/MustardSeedNetworks/niac-go/commit/156c2f09e92ebd196ec394f8142e6844abb74a3e))
* **scenario:** author utilization that looks like a network under load ([622a262](https://github.com/MustardSeedNetworks/niac-go/commit/622a262afc887d06f2e67239524af1711bcd5984))
* **ui:** read runtime state from a named session ([55fc242](https://github.com/MustardSeedNetworks/niac-go/commit/55fc24204a8aed739a882767004e411e7f6c68ba))


### Bug Fixes

* **daemon:** stop a new session taking the default from a running one ([c99dc95](https://github.com/MustardSeedNetworks/niac-go/commit/c99dc952f6981960e29074553abc641364353075))
* **dns:** stop emitting a trailing empty label in PTR replies ([319ecd7](https://github.com/MustardSeedNetworks/niac-go/commit/319ecd706a06ea35b9e8c48a116c5393afcdaa4f)), closes [#1200](https://github.com/MustardSeedNetworks/niac-go/issues/1200)
* **lab:** name the logical attachment when starting a pack ([f7277fe](https://github.com/MustardSeedNetworks/niac-go/commit/f7277fe4f814fd9143170d943734a2e74a00bc2d))
* **scenario:** give endpoints an SNMP identity so they stop rendering as IPs ([46efb38](https://github.com/MustardSeedNetworks/niac-go/commit/46efb38dea8658ef8ea4d8ed843ca37ed875c1e7))
* **scenario:** keep authored utilization below the Link-Live warning line ([4d56ada](https://github.com/MustardSeedNetworks/niac-go/commit/4d56ada3bf00388945b7cc2aee5bf64c38ed3eac))


### Code Refactoring

* **api,daemon,cli:** remove feature gating and the 402 path ([bef04b5](https://github.com/MustardSeedNetworks/niac-go/commit/bef04b517ab2cd5c2748c47467a3cba051359546))
* **api:** drop the template upload and delete surface ([41d44e0](https://github.com/MustardSeedNetworks/niac-go/commit/41d44e059b1c6a748d8184ae91613a756ce6af54))
* delete the runtime license package, endpoint and CLI ([11b5804](https://github.com/MustardSeedNetworks/niac-go/commit/11b5804784b42f7cbdd2afeab362e950fda2233f))
* move home-directory resolution out of the license package ([2d4e5fd](https://github.com/MustardSeedNetworks/niac-go/commit/2d4e5fddcd31e86c94ebb9bd64c000c8867af8e8))
* **ui:** remove the license page, context and feature gates ([6299f97](https://github.com/MustardSeedNetworks/niac-go/commit/6299f97b2ef58f1576caec33e43e8d97a9650141))


### Documentation

* supersede runtime licensing and drop tier claims ([3ab5e37](https://github.com/MustardSeedNetworks/niac-go/commit/3ab5e3795aa942a6fcd1cdbd9d1de3498defbcc7)), closes [#1203](https://github.com/MustardSeedNetworks/niac-go/issues/1203)

## [0.94.27](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.26...v0.94.27) (2026-08-06)


### Features

* **api:** hold simulation state per session ([5196c6d](https://github.com/MustardSeedNetworks/niac-go/commit/5196c6dc13226999914e0609685916f1a477273a))
* **api:** scope SSE packet and stats streams to a session ([cedf2d2](https://github.com/MustardSeedNetworks/niac-go/commit/cedf2d2c6b91070ab0395857f4f02de37ff60a75))
* **api:** select and stop simulations by session ([8a1b51e](https://github.com/MustardSeedNetworks/niac-go/commit/8a1b51e87c4114e6c4d9d55587af1a076a3ae6dd))
* **config:** accept endpoint device types and ieee80211 interfaces ([98a885f](https://github.com/MustardSeedNetworks/niac-go/commit/98a885f7263aa6371dcad7f8b675f7beb0ab315e))
* **daemon,cli:** run concurrent trunk-VLAN sessions ([82461f2](https://github.com/MustardSeedNetworks/niac-go/commit/82461f2b75e9ae8873cdbdeb733e90447485bded))
* **daemon:** add trunk VLAN capture and replay primitives ([e936678](https://github.com/MustardSeedNetworks/niac-go/commit/e936678eb789ececd568e680e924bb3ef17fa0f2))
* **daemon:** persist and recover every active session ([3dcf788](https://github.com/MustardSeedNetworks/niac-go/commit/3dcf78801b07493e60c1c9f1e4be0a4c7d502582))
* **daemon:** track concurrent simulations in a session registry ([2021b74](https://github.com/MustardSeedNetworks/niac-go/commit/2021b74979013c45b4ca5db6aeec934c3ac5a538))
* **fabric:** add trunk attachment mode with an allowed-VLAN set ([91353b1](https://github.com/MustardSeedNetworks/niac-go/commit/91353b168487b9ea511ad9982ea9d4b97d39e1e3))
* **lab:** add a one-command Link-Live acceptance loop per pack ([05f0a09](https://github.com/MustardSeedNetworks/niac-go/commit/05f0a0940defd94780467b495fdeddbfdcd24a9e))
* **linklive-acceptance:** select the latest ready discovery by unit ([9964380](https://github.com/MustardSeedNetworks/niac-go/commit/9964380c935d5bcbddc226449c9750bfcb3415fe))
* **linklive:** compare authored interfaces against discovered telemetry ([99b7bd1](https://github.com/MustardSeedNetworks/niac-go/commit/99b7bd1d3f9cc06354a926a73b051a38334b8b38))
* **linklive:** refresh the access token when Link-Live returns 426 ([6868a8a](https://github.com/MustardSeedNetworks/niac-go/commit/6868a8a9824a250e938ff5c838e95aea3a4159e3))
* **protocols:** confine trunk traffic to the bound physical VLAN ([498c149](https://github.com/MustardSeedNetworks/niac-go/commit/498c149630c174ed3bad884823f79ce5ad36dd4e))
* **scenario:** give AP radios real identity and retier the scenario packs ([12566d0](https://github.com/MustardSeedNetworks/niac-go/commit/12566d00b2af34f792a6b55579ebc4934d41999e))
* **snmp:** publish ifType 71 for ieee80211 interfaces ([51c6225](https://github.com/MustardSeedNetworks/niac-go/commit/51c6225c0eba0e421348a4547ac8b746b80d6070))
* **ui:** offer trunk attachment and a session ID at preflight ([d8197e4](https://github.com/MustardSeedNetworks/niac-go/commit/d8197e420a3f3d3b84812809b7c7f590fd8028fe))
* **ui:** scope the live packet stream to the running session ([bd17bdc](https://github.com/MustardSeedNetworks/niac-go/commit/bd17bdc469a398f790bf426c1ae4f2be8fc66449))
* **ui:** separate presentation packs from the scale workload ([73e2003](https://github.com/MustardSeedNetworks/niac-go/commit/73e2003fa050ff503a854734bd782d38c01032a1))
* **ui:** show and control concurrent scenario sessions ([b0642c8](https://github.com/MustardSeedNetworks/niac-go/commit/b0642c8e1996a5e45557f35ab180e994dd21a6b6))


### Bug Fixes

* **i18n:** allow-list Enterprise in the enterprise-scale pack name ([8332701](https://github.com/MustardSeedNetworks/niac-go/commit/833270101976589744cb05138b2ffb67e440c6fc))
* **i18n:** restore the extractor's key order in the en page catalog ([727bd94](https://github.com/MustardSeedNetworks/niac-go/commit/727bd944dcd9db4adc9b825e9b84760f988cce30))
* **ui:** send the camel-case wire contract for simulation requests ([07853ff](https://github.com/MustardSeedNetworks/niac-go/commit/07853ff377f849749e1b1d9f1a715fc8ca981ec4))


### Code Refactoring

* **protocols,capture,replay:** depend on transport interfaces ([36bca82](https://github.com/MustardSeedNetworks/niac-go/commit/36bca82a7f31996928be98d6be8b3514fb751079))


### Documentation

* add 2026-08 NIAC architecture and program planning docs ([469895d](https://github.com/MustardSeedNetworks/niac-go/commit/469895da42d86a21fa9d8abf0321bf3727954a03))
* correct the scenario pack list in the REST reference ([126d60c](https://github.com/MustardSeedNetworks/niac-go/commit/126d60c68a9ee22239bb573380f69a2ac2e82af6))
* sequence all scenario-shape work into M4 ([13abe33](https://github.com/MustardSeedNetworks/niac-go/commit/13abe334d9b0f96f569e7b8cf66e3f73ad39da98))


### Tests

* **api:** cover session selection and SSE scoping ([dcd549d](https://github.com/MustardSeedNetworks/niac-go/commit/dcd549d5926af08dfeb8200a014499719854f0db))
* **api:** guard the scenario config size ceiling and pack count ([db465f5](https://github.com/MustardSeedNetworks/niac-go/commit/db465f593903bf42b2018e57ac012b3a52d1f6dd))
* **e2e:** report the response body when a daemon start assertion fails ([ec552d4](https://github.com/MustardSeedNetworks/niac-go/commit/ec552d4bce0446e63d78afc131b86c143040dab5))


### Miscellaneous

* **deps-dev:** bump fast-uri ([#1185](https://github.com/MustardSeedNetworks/niac-go/issues/1185)) ([d4163f1](https://github.com/MustardSeedNetworks/niac-go/commit/d4163f1af3a87bb7a2322fd447ba06c4693f4d2b))
* **ui:** bump brace-expansion to 5.0.9 ([50db5ad](https://github.com/MustardSeedNetworks/niac-go/commit/50db5ad483b6445621b48f6aba0ed13269c14233))

## [0.94.26](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.25...v0.94.26) (2026-07-30)


### Bug Fixes

* align generated topology with Link-Live ([#1183](https://github.com/MustardSeedNetworks/niac-go/issues/1183)) ([d453341](https://github.com/MustardSeedNetworks/niac-go/commit/d45334182fdd2d4eb8e83dffdb984272850b8f79))

## [0.94.25](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.24...v0.94.25) (2026-07-30)


### Documentation

* sync dependency metadata ([#1179](https://github.com/MustardSeedNetworks/niac-go/issues/1179)) ([144492e](https://github.com/MustardSeedNetworks/niac-go/commit/144492e039d7fa59128763bc25116bd094a18555))


### Tests

* **e2e:** replace weak page assertions ([#1181](https://github.com/MustardSeedNetworks/niac-go/issues/1181)) ([487929a](https://github.com/MustardSeedNetworks/niac-go/commit/487929aee23e60515e98a006678ba48acd2d3647))

## [0.94.24](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.23...v0.94.24) (2026-07-30)


### Bug Fixes

* complete Phase 6 defect hardening ([#1177](https://github.com/MustardSeedNetworks/niac-go/issues/1177)) ([3848e8e](https://github.com/MustardSeedNetworks/niac-go/commit/3848e8e33bdb51bb1b431719f22d910798097a7e))

## [0.94.23](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.22...v0.94.23) (2026-07-30)


### Features

* add fleet generation to scenario wizard ([#1159](https://github.com/MustardSeedNetworks/niac-go/issues/1159)) ([f334cab](https://github.com/MustardSeedNetworks/niac-go/commit/f334cab05fb725a518bb1872add305d5af231f17))
* add revision-safe draft topology mutations ([#1160](https://github.com/MustardSeedNetworks/niac-go/issues/1160)) ([b298a6a](https://github.com/MustardSeedNetworks/niac-go/commit/b298a6a1b5f88f668fd10b7728a71dcff8a62094))
* add revisioned scenario draft storage ([#1154](https://github.com/MustardSeedNetworks/niac-go/issues/1154)) ([8a88165](https://github.com/MustardSeedNetworks/niac-go/commit/8a88165bb282bc7a24f0223a30f83f8c754cebbe))
* add saved behavior timelines ([#1163](https://github.com/MustardSeedNetworks/niac-go/issues/1163)) ([072af57](https://github.com/MustardSeedNetworks/niac-go/commit/072af571cdf1de3f6476a3d8a8bf17937305b2ee))
* add versioned scenario packs ([#1165](https://github.com/MustardSeedNetworks/niac-go/issues/1165)) ([b9c9126](https://github.com/MustardSeedNetworks/niac-go/commit/b9c91267020eb93c8d4a47660f8111e721aaba30))
* add visual draft topology composer ([#1161](https://github.com/MustardSeedNetworks/niac-go/issues/1161)) ([37f78a7](https://github.com/MustardSeedNetworks/niac-go/commit/37f78a7b2082ead4d46f89291b6f5154c6970ac8))
* add walk capture profile review ([#1162](https://github.com/MustardSeedNetworks/niac-go/issues/1162)) ([131a886](https://github.com/MustardSeedNetworks/niac-go/commit/131a8862dcafabbb6521f7b40c31a12e57e028a3))
* expose protected scenario draft API ([#1155](https://github.com/MustardSeedNetworks/niac-go/issues/1155)) ([68a8071](https://github.com/MustardSeedNetworks/niac-go/commit/68a807164ed979a593c895872457f9b3f42f9a89))
* generate deterministic scenario fleets ([#1158](https://github.com/MustardSeedNetworks/niac-go/issues/1158)) ([2124508](https://github.com/MustardSeedNetworks/niac-go/commit/2124508bd8ae8860c37c43e1ab3ddc2284cbc8a6))
* make simulation wizard draft-first ([#1156](https://github.com/MustardSeedNetworks/niac-go/issues/1156)) ([1eb8842](https://github.com/MustardSeedNetworks/niac-go/commit/1eb88424e28d6a62dd7c7a1bb74e234ec9fddbce))


### Performance Improvements

* cache deterministic OUI allocation ([#1157](https://github.com/MustardSeedNetworks/niac-go/issues/1157)) ([a6d5f2d](https://github.com/MustardSeedNetworks/niac-go/commit/a6d5f2d2d5777d43924cc762f322a2081eed0136))


### Documentation

* plan customer scenario authoring ([#1150](https://github.com/MustardSeedNetworks/niac-go/issues/1150)) ([b3310b8](https://github.com/MustardSeedNetworks/niac-go/commit/b3310b814c141c21afbd9adf0d0fe67de190d770))

## [0.94.22](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.21...v0.94.22) (2026-07-28)


### Bug Fixes

* **discovery:** render complete multisite topology in Link-Live ([#1148](https://github.com/MustardSeedNetworks/niac-go/issues/1148)) ([e375361](https://github.com/MustardSeedNetworks/niac-go/commit/e375361b3769397713240bc7839498dc922da26d))

## [0.94.21](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.20...v0.94.21) (2026-07-26)


### Bug Fixes

* **snmp:** preserve discovery device roles ([#1146](https://github.com/MustardSeedNetworks/niac-go/issues/1146)) ([be443f6](https://github.com/MustardSeedNetworks/niac-go/commit/be443f6cb2f5ded1ee36371de6c24f155a65076f))

## [0.94.20](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.19...v0.94.20) (2026-07-26)


### Bug Fixes

* **snmp:** correlate routed clients in discovery ([#1142](https://github.com/MustardSeedNetworks/niac-go/issues/1142)) ([af08071](https://github.com/MustardSeedNetworks/niac-go/commit/af08071fe0b4df3b6417378087cd14374cd9af33))

## [0.94.19](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.18...v0.94.19) (2026-07-25)


### Bug Fixes

* improve discovery topology fidelity ([#1139](https://github.com/MustardSeedNetworks/niac-go/issues/1139)) ([a2feb7d](https://github.com/MustardSeedNetworks/niac-go/commit/a2feb7d5630ed9d6c0704fe53fed176da6321b7d))

## [0.94.18](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.17...v0.94.18) (2026-07-25)


### Bug Fixes

* harden Npcap SDK release dependency ([#1137](https://github.com/MustardSeedNetworks/niac-go/issues/1137)) ([68d5875](https://github.com/MustardSeedNetworks/niac-go/commit/68d587514536e9406eb2d3fb38382c79d0c9686f))

## [0.94.17](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.16...v0.94.17) (2026-07-25)


### Features

* **snmp:** simulate realistic interface telemetry ([#1134](https://github.com/MustardSeedNetworks/niac-go/issues/1134)) ([77926ef](https://github.com/MustardSeedNetworks/niac-go/commit/77926eff92d07650459ead41111334b869271425))

## [0.94.16](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.15...v0.94.16) (2026-07-25)


### Bug Fixes

* **snmp:** merge authored interfaces into synthesized MIB ([#1130](https://github.com/MustardSeedNetworks/niac-go/issues/1130)) ([4ea8956](https://github.com/MustardSeedNetworks/niac-go/commit/4ea89568cfef2bf22ab9e27579ffaa40438fe249))

## [0.94.15](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.14...v0.94.15) (2026-07-25)


### Bug Fixes

* **snmp:** preserve authored physical identity ([#1128](https://github.com/MustardSeedNetworks/niac-go/issues/1128)) ([21b370d](https://github.com/MustardSeedNetworks/niac-go/commit/21b370d72cafd89c66fff345b38c72222a968611)), closes [#1127](https://github.com/MustardSeedNetworks/niac-go/issues/1127)

## [0.94.14](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.13...v0.94.14) (2026-07-25)


### Bug Fixes

* **api:** enforce device limits on config generation ([#1114](https://github.com/MustardSeedNetworks/niac-go/issues/1114)) ([282beba](https://github.com/MustardSeedNetworks/niac-go/commit/282beba10a6043e7cf94e12d4575b302d8421107))
* **api:** project live simulation topology ([#1102](https://github.com/MustardSeedNetworks/niac-go/issues/1102)) ([117dac8](https://github.com/MustardSeedNetworks/niac-go/commit/117dac84b89119962715e80c68e58c493ea37115))
* **api:** publish authoritative stats stream ([#1111](https://github.com/MustardSeedNetworks/niac-go/issues/1111)) ([e5aefc4](https://github.com/MustardSeedNetworks/niac-go/commit/e5aefc4e566c16699facd238c180b18ae1f52db5))
* **auth:** secure browser API sessions ([#1090](https://github.com/MustardSeedNetworks/niac-go/issues/1090)) ([60d431e](https://github.com/MustardSeedNetworks/niac-go/commit/60d431e087c35e7053ec0ea53cd89f9ad764ba2c))
* **catalog:** enforce reproducible semantic sync ([#1107](https://github.com/MustardSeedNetworks/niac-go/issues/1107)) ([3eed7ad](https://github.com/MustardSeedNetworks/niac-go/commit/3eed7addd7569bd2ba66615188e07ae679eae6b2))
* **ci:** stop grading private UI as crawlable ([#1122](https://github.com/MustardSeedNetworks/niac-go/issues/1122)) ([5b443cf](https://github.com/MustardSeedNetworks/niac-go/commit/5b443cf86578065848534a4a56cf2914b4deee9c))
* **config:** reject duplicate segment tags ([#1093](https://github.com/MustardSeedNetworks/niac-go/issues/1093)) ([1916d87](https://github.com/MustardSeedNetworks/niac-go/commit/1916d8702c1b2f7651afe0dfc699fda6725f323f))
* **config:** resolve segment references at load time ([#1095](https://github.com/MustardSeedNetworks/niac-go/issues/1095)) ([50fcaed](https://github.com/MustardSeedNetworks/niac-go/commit/50fcaed2e0d5a3a352fa486ca9e57566c26debcd))
* **daemon:** recover active simulation after restart ([#1106](https://github.com/MustardSeedNetworks/niac-go/issues/1106)) ([8deada0](https://github.com/MustardSeedNetworks/niac-go/commit/8deada022b498307b19d66a1afd56090ceb12e44))
* **daemon:** recover from capture loop exit ([#1113](https://github.com/MustardSeedNetworks/niac-go/issues/1113)) ([f95a4d7](https://github.com/MustardSeedNetworks/niac-go/commit/f95a4d7be7645de96d155054d7631def78405c5f))
* **daemon:** replace simulations transactionally ([#1100](https://github.com/MustardSeedNetworks/niac-go/issues/1100)) ([f90e89b](https://github.com/MustardSeedNetworks/niac-go/commit/f90e89b853317ca20347e431a579d9e7c2a1145a))
* **daemon:** validate interfaces during preflight ([#1096](https://github.com/MustardSeedNetworks/niac-go/issues/1096)) ([497d635](https://github.com/MustardSeedNetworks/niac-go/commit/497d63573b6322c65a2bc4772664428d720a458a))
* **deps:** update foundation to v0.2.1 ([#1086](https://github.com/MustardSeedNetworks/niac-go/issues/1086)) ([694223b](https://github.com/MustardSeedNetworks/niac-go/commit/694223bc0e6fadbe7a3b41e7f626e6619d0541ad))
* **fabric:** account TTL expiry as a drop ([#1094](https://github.com/MustardSeedNetworks/niac-go/issues/1094)) ([ba7a5f8](https://github.com/MustardSeedNetworks/niac-go/commit/ba7a5f8869f208741c242c3b537a33a0f3506297))
* **fabric:** validate DHCP leases and options ([#1097](https://github.com/MustardSeedNetworks/niac-go/issues/1097)) ([965c17f](https://github.com/MustardSeedNetworks/niac-go/commit/965c17fefa7733f0821432145618ab25402033c9))
* **fabric:** validate live static route next hops ([#1092](https://github.com/MustardSeedNetworks/niac-go/issues/1092)) ([08d4e5d](https://github.com/MustardSeedNetworks/niac-go/commit/08d4e5d34b03108273d78b0595e3c765d8f87fe5))
* **i18n:** refresh runtime catalog extraction ([#1099](https://github.com/MustardSeedNetworks/niac-go/issues/1099)) ([50e0e87](https://github.com/MustardSeedNetworks/niac-go/commit/50e0e87c93764592568f29ac17ce7a27ed6a1094))
* **protocols:** bound UDP proxy concurrency ([#1088](https://github.com/MustardSeedNetworks/niac-go/issues/1088)) ([56078ce](https://github.com/MustardSeedNetworks/niac-go/commit/56078ce220c086ba0682f569095e69efca343424))
* **protocols:** confine routed discovery egress ([#1124](https://github.com/MustardSeedNetworks/niac-go/issues/1124)) ([26d71fa](https://github.com/MustardSeedNetworks/niac-go/commit/26d71fa542312608ed101dc54b60b9fefd641d7a))
* **protocols:** enforce single-use stack lifecycle ([#1105](https://github.com/MustardSeedNetworks/niac-go/issues/1105)) ([ee1ac0b](https://github.com/MustardSeedNetworks/niac-go/commit/ee1ac0b5de8ba1d5f47ee1f9119539549c866256))
* **protocols:** expire idle SSH sessions consistently ([#1112](https://github.com/MustardSeedNetworks/niac-go/issues/1112)) ([94d98ea](https://github.com/MustardSeedNetworks/niac-go/commit/94d98ea7295f58bd2d22120101d0a96c96c53b2b))
* **protocols:** route notifications through fabric ([#1101](https://github.com/MustardSeedNetworks/niac-go/issues/1101)) ([89e4317](https://github.com/MustardSeedNetworks/niac-go/commit/89e43172793ef238a639b5defa7544d9cfe68335))
* **protocols:** validate routed IPv4 ingress ([#1104](https://github.com/MustardSeedNetworks/niac-go/issues/1104)) ([6b466ba](https://github.com/MustardSeedNetworks/niac-go/commit/6b466bae443dbb8b8eca1465ac15e422c97d9e27))
* **snmp:** advertise authored CDP peer addresses ([#1126](https://github.com/MustardSeedNetworks/niac-go/issues/1126)) ([6d704df](https://github.com/MustardSeedNetworks/niac-go/commit/6d704df71e99ac6ab441edc99d55135caa8f2ee9))
* **snmp:** project runtime listener tables ([#1103](https://github.com/MustardSeedNetworks/niac-go/issues/1103)) ([5cb1c0f](https://github.com/MustardSeedNetworks/niac-go/commit/5cb1c0f33d9dba1596109d6178171580b31212a1))
* **ui:** consume packet SSE envelope ([#1108](https://github.com/MustardSeedNetworks/niac-go/issues/1108)) ([6e5f273](https://github.com/MustardSeedNetworks/niac-go/commit/6e5f27325f907e22e3159064b93afb613ec74350))
* **ui:** keep offline PCAP analysis accessible ([#1110](https://github.com/MustardSeedNetworks/niac-go/issues/1110)) ([8e12c02](https://github.com/MustardSeedNetworks/niac-go/commit/8e12c020bbbb1983fe1d1482abfa015bfecc77b6))
* **ui:** preflight runtime simulation starts ([#1098](https://github.com/MustardSeedNetworks/niac-go/issues/1098)) ([d04c3f3](https://github.com/MustardSeedNetworks/niac-go/commit/d04c3f352a9b56c6ef3ef16322de9bad67847fff))
* **ui:** resolve page metadata interpolation ([#1120](https://github.com/MustardSeedNetworks/niac-go/issues/1120)) ([084de53](https://github.com/MustardSeedNetworks/niac-go/commit/084de535321471371d8283b9df6a82ffcdd3bf95))


### Documentation

* **plan:** record stage two completion ([#1091](https://github.com/MustardSeedNetworks/niac-go/issues/1091)) ([78e6e55](https://github.com/MustardSeedNetworks/niac-go/commit/78e6e553a2512a5a7c9e7bc624640963b963f8f0))


### Tests

* **web:** enforce first-class browser matrix ([#1109](https://github.com/MustardSeedNetworks/niac-go/issues/1109)) ([a70385d](https://github.com/MustardSeedNetworks/niac-go/commit/a70385dc1687d0f5964cbb391146010dacfd7146))


### Miscellaneous

* **i18n:** migrate extraction to maintained cli ([#1089](https://github.com/MustardSeedNetworks/niac-go/issues/1089)) ([28587c2](https://github.com/MustardSeedNetworks/niac-go/commit/28587c2f8fef73a112427b3ff5f4d0b316407778))
* reconcile pre-1.0 closeout plans ([#1116](https://github.com/MustardSeedNetworks/niac-go/issues/1116)) ([773679e](https://github.com/MustardSeedNetworks/niac-go/commit/773679e9a6a7e3beb6ebfa1a094810189894c442))

## [0.94.13](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.12...v0.94.13) (2026-07-23)


### Bug Fixes

* **release:** keep content handoff outside checkout ([#1084](https://github.com/MustardSeedNetworks/niac-go/issues/1084)) ([83f888a](https://github.com/MustardSeedNetworks/niac-go/commit/83f888a8e71b30998608739b5cc5a8ca90530991))

## [0.94.12](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.11...v0.94.12) (2026-07-23)


### Bug Fixes

* **config:** confine nested segment configurations ([#1071](https://github.com/MustardSeedNetworks/niac-go/issues/1071)) ([e8c9194](https://github.com/MustardSeedNetworks/niac-go/commit/e8c91940f43c9415086482b1bafe13e09f560798))
* **release:** cover content packages with integrity metadata ([#1083](https://github.com/MustardSeedNetworks/niac-go/issues/1083)) ([eb4bd47](https://github.com/MustardSeedNetworks/niac-go/commit/eb4bd47863da7488e23bc5204ef93b5b407dfdd5))
* **tooling:** fail pre-commit checks closed ([#1067](https://github.com/MustardSeedNetworks/niac-go/issues/1067)) ([eb41e19](https://github.com/MustardSeedNetworks/niac-go/commit/eb41e196be34f4c818ad6acb86b6df2b5f46b494))


### Documentation

* add NIAC master closeout plan ([#1064](https://github.com/MustardSeedNetworks/niac-go/issues/1064)) ([cd059b2](https://github.com/MustardSeedNetworks/niac-go/commit/cd059b2d4e58f7e3679006c4d9b36573bc8fdb39))
* advance Stage 0 ledger ([#1070](https://github.com/MustardSeedNetworks/niac-go/issues/1070)) ([1864dc9](https://github.com/MustardSeedNetworks/niac-go/commit/1864dc968c1ac5155aed391d24b38317a82e4c46))
* complete Stage 0 defect reconciliation ([#1082](https://github.com/MustardSeedNetworks/niac-go/issues/1082)) ([b0e1e18](https://github.com/MustardSeedNetworks/niac-go/commit/b0e1e18aca3184dca03c27bd21d94116cbed8080))
* reconcile remaining remediation issues ([#1074](https://github.com/MustardSeedNetworks/niac-go/issues/1074)) ([ee44d48](https://github.com/MustardSeedNetworks/niac-go/commit/ee44d48c06e42072eee7a168ee3666d6b948b293))
* record nested containment delivery ([#1072](https://github.com/MustardSeedNetworks/niac-go/issues/1072)) ([c83c4ca](https://github.com/MustardSeedNetworks/niac-go/commit/c83c4ca27a87186da7345e22af999d3aab63cc35))
* record Stage 2 security reproductions ([#1073](https://github.com/MustardSeedNetworks/niac-go/issues/1073)) ([16ac84b](https://github.com/MustardSeedNetworks/niac-go/commit/16ac84b3119de7c765ecc6e2da2c15c9c6875976))

## [0.94.11](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.10...v0.94.11) (2026-07-23)


### Bug Fixes

* **api:** enforce HTTPS-only listeners ([#1063](https://github.com/MustardSeedNetworks/niac-go/issues/1063)) ([24dda89](https://github.com/MustardSeedNetworks/niac-go/commit/24dda89dedb2ecccda84a290758ca0010a56e0c7))


### Documentation

* retire stale pre-1.0 claims ([#1065](https://github.com/MustardSeedNetworks/niac-go/issues/1065)) ([90ad10b](https://github.com/MustardSeedNetworks/niac-go/commit/90ad10b90e7f956061fb534ec68494752361688a))

## [0.94.10](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.9...v0.94.10) (2026-07-23)


### Features

* **faults:** make interface faults observable ([#1059](https://github.com/MustardSeedNetworks/niac-go/issues/1059)) ([bb5493f](https://github.com/MustardSeedNetworks/niac-go/commit/bb5493f5bc893000e86e614dc77e1585c3157ffb))


### Bug Fixes

* **license:** pin Free tier device contract ([#1062](https://github.com/MustardSeedNetworks/niac-go/issues/1062)) ([a4291a7](https://github.com/MustardSeedNetworks/niac-go/commit/a4291a7099b38796d96396e07ef5d8c6c9ca4100))


### Documentation

* expand v0.94.9 remediation plan ([#1060](https://github.com/MustardSeedNetworks/niac-go/issues/1060)) ([5babd60](https://github.com/MustardSeedNetworks/niac-go/commit/5babd600e4d812c9bf45e5a00d7d16ebd0775278))

## [0.94.9](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.8...v0.94.9) (2026-07-23)


### Documentation

* plan post-release remediation ([#1054](https://github.com/MustardSeedNetworks/niac-go/issues/1054)) ([57a5348](https://github.com/MustardSeedNetworks/niac-go/commit/57a53481193f4b4fe9850e2c8058c5ebc84557fc))


### Miscellaneous

* **deps:** refresh current dependency metadata ([#1058](https://github.com/MustardSeedNetworks/niac-go/issues/1058)) ([fdd5797](https://github.com/MustardSeedNetworks/niac-go/commit/fdd5797b121a2110c9cc54b182d667c23cd8ef5a))

## [0.94.8](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.7...v0.94.8) (2026-07-23)


### Features

* add routed lab observability ([#1031](https://github.com/MustardSeedNetworks/niac-go/issues/1031)) ([4e5202b](https://github.com/MustardSeedNetworks/niac-go/commit/4e5202ba91ec61e8e64240253e976ed732fc0ca0))
* add stateful device CLI and packet-backed SSH ([#1028](https://github.com/MustardSeedNetworks/niac-go/issues/1028)) ([565bacd](https://github.com/MustardSeedNetworks/niac-go/commit/565bacd0f87ade17eb6ff15c814debd41cd3f98b))


### Bug Fixes

* **api:** declare template-use license gate ([#1023](https://github.com/MustardSeedNetworks/niac-go/issues/1023)) ([edfcd83](https://github.com/MustardSeedNetworks/niac-go/commit/edfcd83d0de6c134af854c2a712394c8cd6fdd9e))
* **release:** make manual dispatch snapshot-only ([#1027](https://github.com/MustardSeedNetworks/niac-go/issues/1027)) ([bd91a9f](https://github.com/MustardSeedNetworks/niac-go/commit/bd91a9fefad2fa131ebb33b3de415de556815c52))


### Documentation

* define pre-1.0 stabilization contract ([#1025](https://github.com/MustardSeedNetworks/niac-go/issues/1025)) ([fac8229](https://github.com/MustardSeedNetworks/niac-go/commit/fac82292a16ede00bbb61c8338db838d72c4be99))


### Miscellaneous

* align release CI toolchain ([#1026](https://github.com/MustardSeedNetworks/niac-go/issues/1026)) ([ebe050a](https://github.com/MustardSeedNetworks/niac-go/commit/ebe050a0c0c14eebd69fc0c004834639870b1d91))
* **deps:** refresh release and build tooling ([#1050](https://github.com/MustardSeedNetworks/niac-go/issues/1050)) ([d448016](https://github.com/MustardSeedNetworks/niac-go/commit/d448016cb7eac61e8b4a42a8ff8e29d893809aba))

## [0.94.7](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.6...v0.94.7) (2026-07-22)


### Features

* expose SNMP parity overlays in device editor ([#1020](https://github.com/MustardSeedNetworks/niac-go/issues/1020)) ([0824653](https://github.com/MustardSeedNetworks/niac-go/commit/0824653953262bdac9f573f190eadd369d4b6553))

## [0.94.6](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.5...v0.94.6) (2026-07-21)


### Features

* complete CyberScope routed MIB-II telemetry ([#1017](https://github.com/MustardSeedNetworks/niac-go/issues/1017)) ([f9b4298](https://github.com/MustardSeedNetworks/niac-go/commit/f9b42983aa1b2678970047dce9a5436d321bee49))
* **replay:** add bounded loop-count (--loop=N) ([#999](https://github.com/MustardSeedNetworks/niac-go/issues/999)) ([de57260](https://github.com/MustardSeedNetworks/niac-go/commit/de5726027219b02303ffe855931ece93bc4fa2c3))
* **replay:** add topspeed / pps / Mbps-cap rate modes ([#998](https://github.com/MustardSeedNetworks/niac-go/issues/998)) ([b37e2e5](https://github.com/MustardSeedNetworks/niac-go/commit/b37e2e54ce716cccc9411ba2b99a9d055be283a8))
* **replay:** apply a BPF filter on replay (selective subset) ([#1000](https://github.com/MustardSeedNetworks/niac-go/issues/1000)) ([8bb6e7a](https://github.com/MustardSeedNetworks/niac-go/commit/8bb6e7a1d4b33e3f9efad58ad806ad106423fd0d))
* **replay:** expose rate/loop/filter controls in the UI ([#1001](https://github.com/MustardSeedNetworks/niac-go/issues/1001)) ([e30c67a](https://github.com/MustardSeedNetworks/niac-go/commit/e30c67a2bd159c74792c35e07847df0ee52d6af6))
* **replay:** localize the whole ReplayControlPanel ([#1002](https://github.com/MustardSeedNetworks/niac-go/issues/1002)) ([49b1d5a](https://github.com/MustardSeedNetworks/niac-go/commit/49b1d5a139c2a0c683740c464f803a878a39d700))
* **replay:** stream pcap replay instead of materializing the whole file ([#997](https://github.com/MustardSeedNetworks/niac-go/issues/997)) ([2241cf2](https://github.com/MustardSeedNetworks/niac-go/commit/2241cf2f033c0fb9d317a3b8f357a6e2bd638bbe))


### Bug Fixes

* **capture:** open replay pcap via os.Root, read via pure-Go pcapgo ([#987](https://github.com/MustardSeedNetworks/niac-go/issues/987)) ([1c38205](https://github.com/MustardSeedNetworks/niac-go/commit/1c3820538e824bc13cd9304b8dd0b3b9f1593d12))
* **license:** remove BGP/OSPF/traffic_shaping vaporware surface ([#1011](https://github.com/MustardSeedNetworks/niac-go/issues/1011)) ([08bce6c](https://github.com/MustardSeedNetworks/niac-go/commit/08bce6ceb9f17143a6763e2fe583ab8cdb0002b9))
* **replay:** anchor pcap os.Root at the validated allowed dir ([#993](https://github.com/MustardSeedNetworks/niac-go/issues/993)) ([e288cf0](https://github.com/MustardSeedNetworks/niac-go/commit/e288cf0f65fcc179de385be7036c8905f23358f0)), closes [#986](https://github.com/MustardSeedNetworks/niac-go/issues/986)


### Code Refactoring

* **license:** consume shared foundation module for license + csrf ([#1005](https://github.com/MustardSeedNetworks/niac-go/issues/1005)) ([4a117eb](https://github.com/MustardSeedNetworks/niac-go/commit/4a117eb40138b62c24300954a5c8947d0080d7fe))


### Continuous Integration

* **license-check:** consume the fleet-shared reusable workflow ([#1007](https://github.com/MustardSeedNetworks/niac-go/issues/1007)) ([d20046b](https://github.com/MustardSeedNetworks/niac-go/commit/d20046b0912b9c49649a3d165276fc39843ece25))
* **license-check:** ignore first-party foundation module ([#1006](https://github.com/MustardSeedNetworks/niac-go/issues/1006)) ([35640bf](https://github.com/MustardSeedNetworks/niac-go/commit/35640bfebb3f2628440159deb732e15493ad6a4b))
* pin fleet-shared reusable workflows to a tagged SHA ([#1010](https://github.com/MustardSeedNetworks/niac-go/issues/1010)) ([1420659](https://github.com/MustardSeedNetworks/niac-go/commit/142065928a9f76014146e07f37aad1396bfaea67))
* **semgrep:** consume the fleet-shared reusable Semgrep workflow ([#1008](https://github.com/MustardSeedNetworks/niac-go/issues/1008)) ([40d4435](https://github.com/MustardSeedNetworks/niac-go/commit/40d443557d833006da6b962ad961026e8ad8db3f))


### Miscellaneous

* delete dead subsystems (simulator, mibdb, ipc server, snmp traps) ([#1012](https://github.com/MustardSeedNetworks/niac-go/issues/1012)) ([4020ff5](https://github.com/MustardSeedNetworks/niac-go/commit/4020ff5f4f444fa4aa087664308d88fc0254f733))
* onboard Renovate via shared org preset ([#1003](https://github.com/MustardSeedNetworks/niac-go/issues/1003)) ([f0b39cc](https://github.com/MustardSeedNetworks/niac-go/commit/f0b39ccd3229882e484b7c7cb5cd259977624808))
* **ui:** drop unused @tanstack/react-query dep ([#992](https://github.com/MustardSeedNetworks/niac-go/issues/992)) ([8812969](https://github.com/MustardSeedNetworks/niac-go/commit/8812969aad175caf6fe17895cf3810c150d51dee))
* **ui:** drop unused babel-plugin-react-compiler devDep ([#989](https://github.com/MustardSeedNetworks/niac-go/issues/989)) ([49da1e0](https://github.com/MustardSeedNetworks/niac-go/commit/49da1e0d63dc8e45d8416d91a36ca39e6f415f28))

## [0.94.5](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.4...v0.94.5) (2026-07-09)


### Bug Fixes

* **content:** enforce bundle extraction containment via os.Root ([#985](https://github.com/MustardSeedNetworks/niac-go/issues/985)) ([0665243](https://github.com/MustardSeedNetworks/niac-go/commit/0665243f7a8daaca101dadb279e854cd1a98345d))


### Code Refactoring

* use min/max builtins and range-over-int (D2) ([#982](https://github.com/MustardSeedNetworks/niac-go/issues/982)) ([568d6ea](https://github.com/MustardSeedNetworks/niac-go/commit/568d6ea41902d5a865fd96651caa7232350f34af))
* use slices.ContainsFunc for membership checks (D1) ([#980](https://github.com/MustardSeedNetworks/niac-go/issues/980)) ([e094fde](https://github.com/MustardSeedNetworks/niac-go/commit/e094fded9bcfbe41bc8a1e94e690d92e6c7972db))
* use sync.OnceValues and maps.Keys/slices.Sorted (D3+D4) ([#983](https://github.com/MustardSeedNetworks/niac-go/issues/983)) ([fa63df6](https://github.com/MustardSeedNetworks/niac-go/commit/fa63df660b098f68d296173f59260aca086c2312))


### Miscellaneous

* **deps:** bump to latest (frontend patches + Go x/ minor) ([#978](https://github.com/MustardSeedNetworks/niac-go/issues/978)) ([ad0f05e](https://github.com/MustardSeedNetworks/niac-go/commit/ad0f05eee2ffdb04e2ada86322bc42a52a652989))

## [0.94.4](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.3...v0.94.4) (2026-07-08)


### Features

* **release:** build Windows natively with CGO + Npcap SDK ([#976](https://github.com/MustardSeedNetworks/niac-go/issues/976)) ([7258170](https://github.com/MustardSeedNetworks/niac-go/commit/72581708c50e9622fdfce24beb97abfcf44f1639))

## [0.94.3](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.2...v0.94.3) (2026-07-08)


### Bug Fixes

* **release:** gitignore corpus checkout + writable daemon cert dir ([#974](https://github.com/MustardSeedNetworks/niac-go/issues/974)) ([1cf0280](https://github.com/MustardSeedNetworks/niac-go/commit/1cf02802c4906dc84f67b46dad7b3f6c50c174c4))

## [0.94.2](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.1...v0.94.2) (2026-07-08)


### Bug Fixes

* **release:** force bash on container SBOM/bundle steps ([#968](https://github.com/MustardSeedNetworks/niac-go/issues/968)) ([f331398](https://github.com/MustardSeedNetworks/niac-go/commit/f331398e1c592d45843bcd00c0ddc141fc80bce7))
* **security:** bump Go 1.26.4 -&gt; 1.26.5 for GO-2026-4970 ([#970](https://github.com/MustardSeedNetworks/niac-go/issues/970)) ([a883bb2](https://github.com/MustardSeedNetworks/niac-go/commit/a883bb22a875b580615859ec1ca0bde159aae414))


### Continuous Integration

* **lint:** exclude SA5011 in tests under Go 1.26.5 tooling lag ([#972](https://github.com/MustardSeedNetworks/niac-go/issues/972)) ([d2bcd1c](https://github.com/MustardSeedNetworks/niac-go/commit/d2bcd1cefaedcaa35c5953ef5f73bcc477363bf1))
* **release:** use msn-ci-bot App token for release-please ([#969](https://github.com/MustardSeedNetworks/niac-go/issues/969)) ([d02b216](https://github.com/MustardSeedNetworks/niac-go/commit/d02b216ff1ad5fc9793c4a4a52f310dea2200136))

## [0.94.1](https://github.com/MustardSeedNetworks/niac-go/compare/v0.94.0...v0.94.1) (2026-07-08)


### Features

* add SNMP walk analyzer and multi-VLAN segments views ([#906](https://github.com/MustardSeedNetworks/niac-go/issues/906)) ([ca39928](https://github.com/MustardSeedNetworks/niac-go/commit/ca39928f14bf4aa9f1a1e4d3c38f4cbf23f9e9d0))
* **api:** batch device delete in one request with per-hostname results ([#948](https://github.com/MustardSeedNetworks/niac-go/issues/948)) ([bfa2b43](https://github.com/MustardSeedNetworks/niac-go/commit/bfa2b437551de2b46c3648008aaa8a45e3a6872c))
* **api:** library install route + content-bundle upload UI ([#967](https://github.com/MustardSeedNetworks/niac-go/issues/967)) ([84a3b7c](https://github.com/MustardSeedNetworks/niac-go/commit/84a3b7c2607f6c6e622ed4b37d830420160b4622))
* **api:** library walk sanitize route + internal/sanitize extraction ([#954](https://github.com/MustardSeedNetworks/niac-go/issues/954)) ([8de0b48](https://github.com/MustardSeedNetworks/niac-go/commit/8de0b48d289246c17cf68b72b74f6e5b314f9129))
* **api:** per-device interfaces read route + Error-Injection interface dropdown ([#946](https://github.com/MustardSeedNetworks/niac-go/issues/946)) ([882f1d1](https://github.com/MustardSeedNetworks/niac-go/commit/882f1d196bd8a1237320e546f460098544a46907))
* **api:** structured YAML parse errors (line/column) + editor line highlight ([#947](https://github.com/MustardSeedNetworks/niac-go/issues/947)) ([5e1b515](https://github.com/MustardSeedNetworks/niac-go/commit/5e1b515788a81b2961b51f0544e9e802494c73f3))
* **api:** surface real libpcap BPF compile error to the UI ([#942](https://github.com/MustardSeedNetworks/niac-go/issues/942)) ([abbd929](https://github.com/MustardSeedNetworks/niac-go/commit/abbd929ed9b5138f27a2a465c51e1a5f1c7549f4))
* **content:** bundle generator + niac-content deb/rpm [UNVALIDATED packaging] ([#966](https://github.com/MustardSeedNetworks/niac-go/issues/966)) ([502acd7](https://github.com/MustardSeedNetworks/niac-go/commit/502acd76637b312b81ab1e9c769c511178d1d4d4))
* **library:** embed essential walks seeded on first run ([#965](https://github.com/MustardSeedNetworks/niac-go/issues/965)) ([f37130b](https://github.com/MustardSeedNetworks/niac-go/commit/f37130b8d607c78192414a92d50190efeabae4ea))
* **license:** human-readable labels + descriptions per feature flag ([#944](https://github.com/MustardSeedNetworks/niac-go/issues/944)) ([8f66983](https://github.com/MustardSeedNetworks/niac-go/commit/8f6698366c2d8a41eab9a5eebd712b5d8968172f))
* **release:** migrate to goreleaser-cross, retire native-runner matrix ([#908](https://github.com/MustardSeedNetworks/niac-go/issues/908)) ([ed1733f](https://github.com/MustardSeedNetworks/niac-go/commit/ed1733f8d8750d23c77944195ca7827aba37e9ff))
* **replay:** live progress (packets/bytes sent, percent) in status + UI bar ([#945](https://github.com/MustardSeedNetworks/niac-go/issues/945)) ([b4239a7](https://github.com/MustardSeedNetworks/niac-go/commit/b4239a7fb116e1698ab067bd18afc7fc55f8e23a))
* **ui:** accessible jargon-disclosure popovers + filter captions ([#932](https://github.com/MustardSeedNetworks/niac-go/issues/932)) ([8f739cb](https://github.com/MustardSeedNetworks/niac-go/commit/8f739cb9342495ba87efe3034d67877445283003))
* **ui:** Block/Overlay merge-view toggle in ConfigDiff ([#955](https://github.com/MustardSeedNetworks/niac-go/issues/955)) ([f02cb2b](https://github.com/MustardSeedNetworks/niac-go/commit/f02cb2b1d9b44d0768fb6cb875c58e0e3ed8e784))
* **ui:** clarify template preview CTAs (Use as-is / Customize first) ([#939](https://github.com/MustardSeedNetworks/niac-go/issues/939)) ([819b197](https://github.com/MustardSeedNetworks/niac-go/commit/819b197d3ee12ab5ddd60790642025734a7949a5))
* **ui:** complete the topology legend to match rendered encodings ([#931](https://github.com/MustardSeedNetworks/niac-go/issues/931)) ([80446b4](https://github.com/MustardSeedNetworks/niac-go/commit/80446b4d44d41aec9d658193310af209dd4447f5))
* **ui:** dashboard honesty + shared sim-status chip in header ([#927](https://github.com/MustardSeedNetworks/niac-go/issues/927)) ([2ca6dea](https://github.com/MustardSeedNetworks/niac-go/commit/2ca6dea7d40318c64eb8c8fea3538edcc558a5b6))
* **ui:** expand-all/collapse-all controls in ProtocolTree ([#953](https://github.com/MustardSeedNetworks/niac-go/issues/953)) ([a9fd433](https://github.com/MustardSeedNetworks/niac-go/commit/a9fd4338867e4e14592b1d1211c8a29c42844526))
* **ui:** group TrafficSection into three collapsible subsections ([#952](https://github.com/MustardSeedNetworks/niac-go/issues/952)) ([206f8f4](https://github.com/MustardSeedNetworks/niac-go/commit/206f8f4bcc723913a0532a8f78af62919493a5b8))
* **ui:** guided New Simulation wizard composing existing config flow ([#962](https://github.com/MustardSeedNetworks/niac-go/issues/962)) ([ee4eb1e](https://github.com/MustardSeedNetworks/niac-go/commit/ee4eb1edfb313d0455f71f8e4ece3321854da87c))
* **ui:** non-color direction glyphs in StreamView ([#951](https://github.com/MustardSeedNetworks/niac-go/issues/951)) ([15f15ad](https://github.com/MustardSeedNetworks/niac-go/commit/15f15ad49c73468ecc51234cc1988c59b879109a))
* **ui:** pcap upload progress bar and friendly over-limit error ([#960](https://github.com/MustardSeedNetworks/niac-go/issues/960)) ([6bdf4e0](https://github.com/MustardSeedNetworks/niac-go/commit/6bdf4e0445857daac8d3e0e8c2745eec28ddc96e))
* **ui:** persist topology layout to localStorage ([#956](https://github.com/MustardSeedNetworks/niac-go/issues/956)) ([e50bdd3](https://github.com/MustardSeedNetworks/niac-go/commit/e50bdd3d1cbae5f5d75dfdce014983a5271f29c1))
* **ui:** sanitize actions on the library walks view ([#961](https://github.com/MustardSeedNetworks/niac-go/issues/961)) ([fed1a6a](https://github.com/MustardSeedNetworks/niac-go/commit/fed1a6af703eddff4ee925cfa8fab869c70cda2a))
* **ui:** wire batch walk validation (validate-all) with per-file results ([#940](https://github.com/MustardSeedNetworks/niac-go/issues/940)) ([529adea](https://github.com/MustardSeedNetworks/niac-go/commit/529adea99aa5a1e7e048fe5ccbd39ea8e25e3b8b))
* **walk:** add OID to validation issues + OID column/filter in the validator ([#943](https://github.com/MustardSeedNetworks/niac-go/issues/943)) ([4bb0e56](https://github.com/MustardSeedNetworks/niac-go/commit/4bb0e5650384e9a3bf1625db960c305c4fe43ee0))


### Bug Fixes

* **api:** size pcap upload body cap for base64 overhead so 100MB raw is accepted ([#922](https://github.com/MustardSeedNetworks/niac-go/issues/922)) ([c5f7c2a](https://github.com/MustardSeedNetworks/niac-go/commit/c5f7c2a116226280e05683c61dad8171a8f9f110))
* **build:** make quality gates fail honestly, dedupe vars.mk ([#903](https://github.com/MustardSeedNetworks/niac-go/issues/903)) ([74b38d0](https://github.com/MustardSeedNetworks/niac-go/commit/74b38d0afe8da68bbe7954e2445236dddf8e8c33))
* **i18n:** keep 'walk'/'Walks' verbatim in es + gate it in CI ([#941](https://github.com/MustardSeedNetworks/niac-go/issues/941)) ([6a451b4](https://github.com/MustardSeedNetworks/niac-go/commit/6a451b4ad0845b7a54bcdb0b1367d851b17bc7e0))
* **release:** correct release-please manifest to 0.94.0 ([#964](https://github.com/MustardSeedNetworks/niac-go/issues/964)) ([ab4a957](https://github.com/MustardSeedNetworks/niac-go/commit/ab4a957394507453ec938875d36fe73562a30bc6))
* **security:** contain path traversal in converter + library (CodeQL go/path-injection) ([#907](https://github.com/MustardSeedNetworks/niac-go/issues/907)) ([f39ed36](https://github.com/MustardSeedNetworks/niac-go/commit/f39ed36b0e1844c8ec5dc9480889f3091e5be6c1))
* **ui:** data-safety guards (FTP password, walk auto-fix confirm, unsaved-changes guard) ([#925](https://github.com/MustardSeedNetworks/niac-go/issues/925)) ([5868328](https://github.com/MustardSeedNetworks/niac-go/commit/586832838eb03f1ab4f272b705b7aa5c6064f191))
* **ui:** mock useSimulationStatus in PacketInspectorPage filtered-export test ([#930](https://github.com/MustardSeedNetworks/niac-go/issues/930)) ([0f56374](https://github.com/MustardSeedNetworks/niac-go/commit/0f563744ab7d421ba6335bebf68cccaeb3923b63))
* **ui:** packet inspector auto-scroll, filtered export, hex header boundary ([#923](https://github.com/MustardSeedNetworks/niac-go/issues/923)) ([aacb4bd](https://github.com/MustardSeedNetworks/niac-go/commit/aacb4bd962d753d122dda75aa7f5ced392d9863a))
* **ui:** surface silent failures and correct misleading copy ([#924](https://github.com/MustardSeedNetworks/niac-go/issues/924)) ([4269b39](https://github.com/MustardSeedNetworks/niac-go/commit/4269b39e9aff8f99196ee7e5fe225723989d7876))


### Code Refactoring

* **api:** unify config store — migrate legacy configs into library networks ([#963](https://github.com/MustardSeedNetworks/niac-go/issues/963)) ([f480c9c](https://github.com/MustardSeedNetworks/niac-go/commit/f480c9cad5033b5b502d484972e69ac51bae9cd5))
* **ui:** consolidate delete-confirm modals onto shared ConfirmModal ([#928](https://github.com/MustardSeedNetworks/niac-go/issues/928)) ([70ce5ec](https://github.com/MustardSeedNetworks/niac-go/commit/70ce5ec80b8c05f04099208143f17fc99b12ccd7))
* **ui:** extract DataTable primitive and migrate fitting list views ([#957](https://github.com/MustardSeedNetworks/niac-go/issues/957)) ([83cdccc](https://github.com/MustardSeedNetworks/niac-go/commit/83cdcccd059908e508337e95e026b5495b6b9aa7))
* **ui:** resolve Devices naming collision + regroup nav (Tools/System, Fault Injection) ([#926](https://github.com/MustardSeedNetworks/niac-go/issues/926)) ([6e3abb2](https://github.com/MustardSeedNetworks/niac-go/commit/6e3abb222e24fdf1d9c5c7a1db6cf1aede3d5655))
* **ui:** wire global toast error surface, retire per-page banners ([#929](https://github.com/MustardSeedNetworks/niac-go/issues/929)) ([da0f479](https://github.com/MustardSeedNetworks/niac-go/commit/da0f479509c0cffa6770b94adc16acd368f76af2))


### Continuous Integration

* gate frontend on vitest + retry flaky libpcap-dev apt install ([#934](https://github.com/MustardSeedNetworks/niac-go/issues/934)) ([4211e0a](https://github.com/MustardSeedNetworks/niac-go/commit/4211e0ab335724ba18d013bded3066dc5c7ba5e8))
* **governance:** exempt release-please PRs from human PR-body template ([#918](https://github.com/MustardSeedNetworks/niac-go/issues/918)) ([11934a6](https://github.com/MustardSeedNetworks/niac-go/commit/11934a6f5b4bfe400e32aaf8766f8bfd42ade983))
* **perf:** enable Go cache, dedup -race, least-privilege ([#914](https://github.com/MustardSeedNetworks/niac-go/issues/914)) ([a3c43a1](https://github.com/MustardSeedNetworks/niac-go/commit/a3c43a153c9c803f76021151f5c0ac0d1fc21d53)), closes [#913](https://github.com/MustardSeedNetworks/niac-go/issues/913)
* **release:** use GITHUB_TOKEN for release-please ([#949](https://github.com/MustardSeedNetworks/niac-go/issues/949)) ([2acc0a2](https://github.com/MustardSeedNetworks/niac-go/commit/2acc0a2623c1ee024afe13678de357e41d4bfdbd))
* **security:** add Semgrep SAST gate ([#916](https://github.com/MustardSeedNetworks/niac-go/issues/916)) ([1ff7b02](https://github.com/MustardSeedNetworks/niac-go/commit/1ff7b0267394b31aa697871d5ee64d60b055a9cb))


### Miscellaneous

* **content:** remove phone-home online download path ([#959](https://github.com/MustardSeedNetworks/niac-go/issues/959)) ([021b97d](https://github.com/MustardSeedNetworks/niac-go/commit/021b97dec04554f7783cacca29a179bea5e4463c))
* **i18n:** externalize config-diff + library + debug + help/sidebar labels (en+es) ([#936](https://github.com/MustardSeedNetworks/niac-go/issues/936)) ([63bd846](https://github.com/MustardSeedNetworks/niac-go/commit/63bd8460d361fcf21ca11ddacc531c87b6e6cd98))
* **i18n:** externalize packet inspector + pcap + topology strings (en+es) ([#937](https://github.com/MustardSeedNetworks/niac-go/issues/937)) ([ba8d314](https://github.com/MustardSeedNetworks/niac-go/commit/ba8d31459833c98547f64da6f4bc56144ec3dd7a))
* **i18n:** externalize runtime + device-list/editor strings (en+es) ([#938](https://github.com/MustardSeedNetworks/niac-go/issues/938)) ([cb1b99a](https://github.com/MustardSeedNetworks/niac-go/commit/cb1b99a0755c66b4afd9ae7d0731fa54172d789c))
* **license:** add license-key-circumvention clause to BUSL Additional Use Grant ([#912](https://github.com/MustardSeedNetworks/niac-go/issues/912)) ([ae360a3](https://github.com/MustardSeedNetworks/niac-go/commit/ae360a34973452fe842573360a38b11c55273ed8))
* **ui:** remove dead components + stem storage-key leak + redundant Java-DSL card ([#921](https://github.com/MustardSeedNetworks/niac-go/issues/921)) ([3d018a3](https://github.com/MustardSeedNetworks/niac-go/commit/3d018a3180ef1c6e6297ff6d63dea2977a4b5061))

## [0.94.0](https://github.com/MustardSeedNetworks/niac-go/compare/v0.93.0...v0.94.0) (2026-07-05)


### Features

* **dhcp:** answer DHCPINFORM + quarantine DHCPDECLINE (completes [#881](https://github.com/MustardSeedNetworks/niac-go/issues/881)) ([#894](https://github.com/MustardSeedNetworks/niac-go/issues/894)) ([7c574ad](https://github.com/MustardSeedNetworks/niac-go/commit/7c574adb869cda2519d56a49812e50e3849a269d))
* multi-VLAN segment playback — run N demos at once (ADR 0008) ([#888](https://github.com/MustardSeedNetworks/niac-go/issues/888)) ([727281f](https://github.com/MustardSeedNetworks/niac-go/commit/727281f4cb78fc709a2125baa0341111d2d139f1))
* **ui:** add License page surfacing tier + features ([#901](https://github.com/MustardSeedNetworks/niac-go/issues/901)) ([80a2f54](https://github.com/MustardSeedNetworks/niac-go/commit/80a2f5458ce4e53967c50ed221fc1db4bedb8782))
* **ui:** localize TierGate upgrade tooltip via react-i18next ([#896](https://github.com/MustardSeedNetworks/niac-go/issues/896)) ([1727f83](https://github.com/MustardSeedNetworks/niac-go/commit/1727f833f025ad641665cbe76fbdc800a8c3573b)), closes [#713](https://github.com/MustardSeedNetworks/niac-go/issues/713)
* **ui:** migrate device editor to react-hook-form + valibot ([#899](https://github.com/MustardSeedNetworks/niac-go/issues/899)) ([0a3b5c4](https://github.com/MustardSeedNetworks/niac-go/commit/0a3b5c4d3fe54305cb71d36c585efd6d46cbadbc)), closes [#730](https://github.com/MustardSeedNetworks/niac-go/issues/730)


### Bug Fixes

* **dhcp:** NAK a REQUEST the server cannot satisfy instead of silence ([#886](https://github.com/MustardSeedNetworks/niac-go/issues/886)) ([e390184](https://github.com/MustardSeedNetworks/niac-go/commit/e39018458865ff5dcc9fab8764ca54041fef741c))
* **dns:** reply on the request VLAN so tagged testers resolve names ([#883](https://github.com/MustardSeedNetworks/niac-go/issues/883)) ([dcb88c2](https://github.com/MustardSeedNetworks/niac-go/commit/dcb88c206989708767515af95c8ec2172ebe05c8))
* **netbios:** reject invalid level-2 name encoding ([#850](https://github.com/MustardSeedNetworks/niac-go/issues/850)) ([#895](https://github.com/MustardSeedNetworks/niac-go/issues/895)) ([923a125](https://github.com/MustardSeedNetworks/niac-go/commit/923a125c96984dce6165715ef3090c3379ae6640))
* **ui:** remove dead exports and wire up non-functional settings/debug controls ([#900](https://github.com/MustardSeedNetworks/niac-go/issues/900)) ([f930c5f](https://github.com/MustardSeedNetworks/niac-go/commit/f930c5fafa1e4769edf8f0f47263a7b058b652fd))

## [0.93.0](https://github.com/MustardSeedNetworks/niac-go/compare/v0.92.0...v0.93.0) (2026-07-04)


### Features

* **dns:** wildcard catch-all forward record ([#864](https://github.com/MustardSeedNetworks/niac-go/issues/864)) ([d60ac9a](https://github.com/MustardSeedNetworks/niac-go/commit/d60ac9a01e3523287afe25ce20be9c9d396ed273))
* **protocols:** tag CDP/LLDP/FDP/EDP advertisements onto the device VLAN ([#860](https://github.com/MustardSeedNetworks/niac-go/issues/860)) ([b059989](https://github.com/MustardSeedNetworks/niac-go/commit/b059989a6f83bd17d033f462105fe7cd93c5cde4))
* **protocols:** tag reactive replies onto the request's VLAN ([#859](https://github.com/MustardSeedNetworks/niac-go/issues/859)) ([78a5a98](https://github.com/MustardSeedNetworks/niac-go/commit/78a5a98e1b96157bc7e5367d576114a45a9eca37))
* **snmp:** device identity (sysName) is authored, never from the walk ([#862](https://github.com/MustardSeedNetworks/niac-go/issues/862)) ([f5a76f1](https://github.com/MustardSeedNetworks/niac-go/commit/f5a76f179f8d9334042382c1c4c7c18def8fc27a))
* **snmp:** learn downstream host MACs into the bridge FDB (Nearest Switch) ([#869](https://github.com/MustardSeedNetworks/niac-go/issues/869)) ([1b7a8d1](https://github.com/MustardSeedNetworks/niac-go/commit/1b7a8d19fcfc07c03e3afc67a1f3cc1e9ee4d0b1))
* **snmp:** let authored trunk_ports topology win over walk neighbour tables ([#861](https://github.com/MustardSeedNetworks/niac-go/issues/861)) ([a98b310](https://github.com/MustardSeedNetworks/niac-go/commit/a98b310ba81bc3641d665337a15d1b3250440caa))
* **udp:** reflect NetAlly UDP performance probes back to the tester ([#879](https://github.com/MustardSeedNetworks/niac-go/issues/879)) ([d121b62](https://github.com/MustardSeedNetworks/niac-go/commit/d121b623bcaebbe08eddb74300a5d9c66cb7dc54))
* **udp:** reply ICMP port-unreachable from a closed UDP port ([#878](https://github.com/MustardSeedNetworks/niac-go/issues/878)) ([e8c4c4b](https://github.com/MustardSeedNetworks/niac-go/commit/e8c4c4b79cc5e7fb925b7042a2b42a4a92394faf))


### Bug Fixes

* **icmp:** VLAN-tag Time Exceeded so path analysis reaches a tagged tester ([#872](https://github.com/MustardSeedNetworks/niac-go/issues/872)) ([a903bd5](https://github.com/MustardSeedNetworks/niac-go/commit/a903bd514ac8526aa7dffc0149a81e49339742ed))
* **sanitize:** scrub device identity echoed outside the system group ([#857](https://github.com/MustardSeedNetworks/niac-go/issues/857)) ([df4771f](https://github.com/MustardSeedNetworks/niac-go/commit/df4771f070eaafbedf5f9792191d743d1da2d065))
* **snmp:** derive FDB bridge ports by inferred offset, not trailing number ([#874](https://github.com/MustardSeedNetworks/niac-go/issues/874)) ([7573d7b](https://github.com/MustardSeedNetworks/niac-go/commit/7573d7b3a2cc9d9abcf206eac3c7a3218fced504))
* **snmp:** learned FDB port must be an in-range bridge port ([#870](https://github.com/MustardSeedNetworks/niac-go/issues/870)) ([2633f73](https://github.com/MustardSeedNetworks/niac-go/commit/2633f7304d3b005e0940b61449eba3f66196941d))
* **snmp:** make NIAC fully discoverable by an SNMP scanner (GET-BULK, sort, cadence) ([#866](https://github.com/MustardSeedNetworks/niac-go/issues/866)) ([9a2ef82](https://github.com/MustardSeedNetworks/niac-go/commit/9a2ef820fb5ad6ee2ccc303081fe048e119e4d44))
* **stack:** in VLAN mode, ignore untagged frames (no native/default replay) ([#865](https://github.com/MustardSeedNetworks/niac-go/issues/865)) ([64ccc32](https://github.com/MustardSeedNetworks/niac-go/commit/64ccc32305b91b4c4c0cc4da1c0ffcd2ee85db7f))
* **stack:** received packets must not alias the reused capture buffer ([#863](https://github.com/MustardSeedNetworks/niac-go/issues/863)) ([f4bc403](https://github.com/MustardSeedNetworks/niac-go/commit/f4bc40324ef1dd1bd4e963b4248c49253c22a87c))
* **tcp:** reply to the request's VLAN and source MAC, add SYN-ACK for path probes ([#876](https://github.com/MustardSeedNetworks/niac-go/issues/876)) ([c6cf3a7](https://github.com/MustardSeedNetworks/niac-go/commit/c6cf3a756af235cd96de48616fc7d7cf33971789))

## [0.92.0](https://github.com/MustardSeedNetworks/niac-go/compare/v0.91.0...v0.92.0) (2026-07-03)


### Features

* **api:** make HTTP method and body-size authoritative in route registry ([#810](https://github.com/MustardSeedNetworks/niac-go/issues/810)) ([04e8f53](https://github.com/MustardSeedNetworks/niac-go/commit/04e8f539d513477b176097a4ca40acb8ee1813da))
* **snmp:** implement SNMPv3 USM authoritative engine (authPriv) ([#853](https://github.com/MustardSeedNetworks/niac-go/issues/853)) ([a3c5976](https://github.com/MustardSeedNetworks/niac-go/commit/a3c59763ef3bdc171bee6102462c6054d694c5e1))


### Bug Fixes

* **ci:** run E2E daemon with NIAC_E2E_DRY_RUN_SIMULATION=1 ([#817](https://github.com/MustardSeedNetworks/niac-go/issues/817)) ([43d2272](https://github.com/MustardSeedNetworks/niac-go/commit/43d2272b6d898a6f544d3da2a8e358c7125d93d6))
* **protocols:** bound DHCPv6 DUID and throttle iperf3 session sweep ([#852](https://github.com/MustardSeedNetworks/niac-go/issues/852)) ([eebebf6](https://github.com/MustardSeedNetworks/niac-go/commit/eebebf6a371a8c0a802a3beae4a63f8569bb5522))
* **sanitize:** correct numeric-walk identity scrub and stop OID corruption ([#854](https://github.com/MustardSeedNetworks/niac-go/issues/854)) ([68fe16e](https://github.com/MustardSeedNetworks/niac-go/commit/68fe16e4453315ce76b8378f2dc2f5e8ffea4a6e))
* **snmp:** make walk replay scale and stop dropping valid OIDs ([#844](https://github.com/MustardSeedNetworks/niac-go/issues/844)) ([38a6e73](https://github.com/MustardSeedNetworks/niac-go/commit/38a6e733977eb1f233dfde8c6ba356ba04442c3a))
* **snmp:** stop the simulator returning 0 OIDs — crash-safety, capture latency, bulk sizing ([#848](https://github.com/MustardSeedNetworks/niac-go/issues/848)) ([2b9bc0f](https://github.com/MustardSeedNetworks/niac-go/commit/2b9bc0fdce4c84cf52dd7735fd39ae3a2117ff98))
* **ui:** make the '?' help shortcut actually open the drawer (N1, N5) ([#812](https://github.com/MustardSeedNetworks/niac-go/issues/812)) ([202ba67](https://github.com/MustardSeedNetworks/niac-go/commit/202ba6722a88c3e5c340b50e017db87707801c6d))

## [0.91.0](https://github.com/MustardSeedNetworks/niac-go/compare/v0.90.0...v0.91.0) (2026-06-16)


### Features

* **api:** capability registry for declarative route policy (register) ([#800](https://github.com/MustardSeedNetworks/niac-go/issues/800)) ([1edc18e](https://github.com/MustardSeedNetworks/niac-go/commit/1edc18e6af95416ff9f214c25e5febc4f3340d5c))
* **api:** make HTTP method and body-size authoritative in route registry ([#810](https://github.com/MustardSeedNetworks/niac-go/issues/810)) ([04e8f53](https://github.com/MustardSeedNetworks/niac-go/commit/04e8f539d513477b176097a4ca40acb8ee1813da))
* **license:** replace forgeable rotor cipher with Ed25519-signed tokens ([#802](https://github.com/MustardSeedNetworks/niac-go/issues/802)) ([d111685](https://github.com/MustardSeedNetworks/niac-go/commit/d111685688c4c1b8b13404e96c0329bec0b5fec5))
* modernize niac operator surfaces ([4ebb685](https://github.com/MustardSeedNetworks/niac-go/commit/4ebb6857729b4d4dd08769ae6aa55121e7abd2ae))


### Bug Fixes

* **ci:** run E2E daemon with NIAC_E2E_DRY_RUN_SIMULATION=1 ([#817](https://github.com/MustardSeedNetworks/niac-go/issues/817)) ([43d2272](https://github.com/MustardSeedNetworks/niac-go/commit/43d2272b6d898a6f544d3da2a8e358c7125d93d6))
* **e2e:** route device-crud to /device-config (was /devices = Running Devices) ([#787](https://github.com/MustardSeedNetworks/niac-go/issues/787)) ([3709b38](https://github.com/MustardSeedNetworks/niac-go/commit/3709b38f9885c48a58a8faca4a6c09c39d8912e6))
* **ui:** drop sidebar-*-button testids on the mobile aside copy ([#786](https://github.com/MustardSeedNetworks/niac-go/issues/786)) ([995865f](https://github.com/MustardSeedNetworks/niac-go/commit/995865fce59df32a8dd9ba437bb2050a6681d193))
* **ui:** make the '?' help shortcut actually open the drawer (N1, N5) ([#812](https://github.com/MustardSeedNetworks/niac-go/issues/812)) ([202ba67](https://github.com/MustardSeedNetworks/niac-go/commit/202ba6722a88c3e5c340b50e017db87707801c6d))

## [0.90.0](https://github.com/krisarmstrong/niac-go/compare/v0.89.0...v0.90.0) (2026-05-29)


### Features

* **a11y:** axe harness + topology node hover tooltips ([#772](https://github.com/krisarmstrong/niac-go/issues/772)) ([3e70bf9](https://github.com/krisarmstrong/niac-go/commit/3e70bf9503b5f3ac1d4d2cd45dd305fa2cc70de5))
* **cli:** fill help gaps on license/template/content/etc + completeness test ([#773](https://github.com/krisarmstrong/niac-go/issues/773)) ([ffe1487](https://github.com/krisarmstrong/niac-go/commit/ffe14878a302fe409904cbb8992c6250a3c45db7))
* **help:** version badge in HelpDrawer header (in-app About) ([#774](https://github.com/krisarmstrong/niac-go/issues/774)) ([e9f6922](https://github.com/krisarmstrong/niac-go/commit/e9f69225e3f9dc28359bf436091752a2c427b2d9))
* **i18n:** en/es key parity + DNT compliance test (niac) ([#775](https://github.com/krisarmstrong/niac-go/issues/775)) ([68ea922](https://github.com/krisarmstrong/niac-go/commit/68ea92230ef6877153ff6494ee905833c180c9cb))


### Bug Fixes

* **api:** atomic mint in CSRFManager.GetOrCreate ([#776](https://github.com/krisarmstrong/niac-go/issues/776)) ([64bb407](https://github.com/krisarmstrong/niac-go/commit/64bb40713992f12c3cfbae5681a57c1a1d1bf016))

## [0.89.0](https://github.com/krisarmstrong/niac-go/compare/v0.88.3...v0.89.0) (2026-05-29)


### Features

* **api,ui:** scope-discovery endpoint + UI gating primitives ([#762](https://github.com/krisarmstrong/niac-go/issues/762)) ([#770](https://github.com/krisarmstrong/niac-go/issues/770)) ([e7f21a4](https://github.com/krisarmstrong/niac-go/commit/e7f21a4b78f5906299368b155d8d83b8ca537a3f))
* **api:** per-session CSRF tokens via CSRFManager ([#1257](https://github.com/krisarmstrong/niac-go/issues/1257) sub-4) ([#771](https://github.com/krisarmstrong/niac-go/issues/771)) ([2a84be9](https://github.com/krisarmstrong/niac-go/commit/2a84be9ea614a38447ad7e2cbf5ace614d27041b))
* **api:** ScopeAdmin tier gates whole-config replacement ([#743](https://github.com/krisarmstrong/niac-go/issues/743)) ([#769](https://github.com/krisarmstrong/niac-go/issues/769)) ([cb81b5c](https://github.com/krisarmstrong/niac-go/commit/cb81b5c2b0e63d3cfae089752ca2284f654081f0))
* **api:** unify authz-denial event name with seed/stem ([#1257](https://github.com/krisarmstrong/niac-go/issues/1257)) ([#767](https://github.com/krisarmstrong/niac-go/issues/767)) ([b107e47](https://github.com/krisarmstrong/niac-go/commit/b107e47f432c2139c0076366d87eeceab6a870eb))

## [0.88.3](https://github.com/krisarmstrong/niac-go/compare/v0.88.2...v0.88.3) (2026-05-29)


### Bug Fixes

* **ui:** repair token-discipline guard; close leaks it now catches ([#760](https://github.com/krisarmstrong/niac-go/issues/760)) ([e0145ab](https://github.com/krisarmstrong/niac-go/commit/e0145ab3315f38611f96ac785cc8daf79430ca5b))

## [0.88.2](https://github.com/krisarmstrong/niac-go/compare/v0.88.1...v0.88.2) (2026-05-29)


### Bug Fixes

* **security:** harden empty-store auth bypass + close CSRF gaps ([#739](https://github.com/krisarmstrong/niac-go/issues/739), [#740](https://github.com/krisarmstrong/niac-go/issues/740)) ([#755](https://github.com/krisarmstrong/niac-go/issues/755)) ([f698b04](https://github.com/krisarmstrong/niac-go/commit/f698b040fe35e0a5a4883b9af977599a706bc098))

## [0.88.1](https://github.com/krisarmstrong/niac-go/compare/v0.88.0...v0.88.1) (2026-05-28)


### Bug Fixes

* **ui:** re-sync shell from stem + add app.title — sidebar shows "NIAC" ([#750](https://github.com/krisarmstrong/niac-go/issues/750)) ([f63e297](https://github.com/krisarmstrong/niac-go/commit/f63e29720f525cc05e43a869f2a87174ff8a4318))

## [0.88.0](https://github.com/krisarmstrong/niac-go/compare/v0.87.0...v0.88.0) (2026-05-28)


### Features

* **ui:** add slim HeaderBar (Phase 2) ([#744](https://github.com/krisarmstrong/niac-go/issues/744)) ([3e36a05](https://github.com/krisarmstrong/niac-go/commit/3e36a0570d1dec229cf6ccf1c839352655b164dc))
* **ui:** sync canonical shell from stem + decouple drawers (Phase 1) ([#738](https://github.com/krisarmstrong/niac-go/issues/738)) ([5cf30fc](https://github.com/krisarmstrong/niac-go/commit/5cf30fc3bde7364a3b21b716d755fd32b4e4f12a))


### Bug Fixes

* **ci:** unblock NIAC main — duplicate heading + Lighthouse cert ([#745](https://github.com/krisarmstrong/niac-go/issues/745)) ([3d4a5ad](https://github.com/krisarmstrong/niac-go/commit/3d4a5ad8aea7c72e71d6d52c25f53455124e8d30))
* **ui:** restore missing spacing utility classes + dual-aside e2e selector ([#747](https://github.com/krisarmstrong/niac-go/issues/747)) ([f56ad75](https://github.com/krisarmstrong/niac-go/commit/f56ad75dbe724bdaf4f226ef218acd9a17a29483))

## [0.87.0](https://github.com/krisarmstrong/niac-go/compare/v0.86.0...v0.87.0) (2026-05-27)


### Features

* **license:** model bgp/ospf/snmpv3 on device, gate bgp+ospf ([#735](https://github.com/krisarmstrong/niac-go/issues/735)) ([5738d90](https://github.com/krisarmstrong/niac-go/commit/5738d905ad21efe9207e7c32c4381bb6a96c1e4b)), closes [#129](https://github.com/krisarmstrong/niac-go/issues/129)

## [0.86.0](https://github.com/krisarmstrong/niac-go/compare/v0.85.0...v0.86.0) (2026-05-27)


### Features

* **api:** add strict JSON decode helpers + sweep handlers ([#718](https://github.com/krisarmstrong/niac-go/issues/718)) ([#721](https://github.com/krisarmstrong/niac-go/issues/721)) ([b22344f](https://github.com/krisarmstrong/niac-go/commit/b22344f7a2bb845fe4bff3fa4f55e52de881e4d0))
* **forms:** adopt react-hook-form + zod resolver ([#725](https://github.com/krisarmstrong/niac-go/issues/725)) ([#729](https://github.com/krisarmstrong/niac-go/issues/729)) ([82bcdf4](https://github.com/krisarmstrong/niac-go/commit/82bcdf46e0664feb9684fdd45071a0dba826cec2))
* **forms:** migrate UploadTemplateModal + ErrorInjectionPanel ([#730](https://github.com/krisarmstrong/niac-go/issues/730)) ([#731](https://github.com/krisarmstrong/niac-go/issues/731)) ([c50d3a7](https://github.com/krisarmstrong/niac-go/commit/c50d3a729874b64c5f2f47ca3bd1966dd510f801))
* **i18n:** add check-keys.py — t() call ↔ EN locale cross-reference ([#723](https://github.com/krisarmstrong/niac-go/issues/723)) ([e71ca2b](https://github.com/krisarmstrong/niac-go/commit/e71ca2bdde321e852c48a3abb88e994beee91a10))
* **i18n:** add per-repo dynamic-prefixes allowlist for check-keys.py ([#732](https://github.com/krisarmstrong/niac-go/issues/732)) ([4ecab2c](https://github.com/krisarmstrong/niac-go/commit/4ecab2c2336dc25373c509b40c63d70a40184fbd))
* **i18n:** migrate 8 long-tail components to t() (16 strings) ([#720](https://github.com/krisarmstrong/niac-go/issues/720)) ([2caed9b](https://github.com/krisarmstrong/niac-go/commit/2caed9b2b4ce7d9dea9c2be115c5eeac969c950d))
* **i18n:** Phase 3 NIAC — bootstrap runtime + migrate ~110 hardcoded JSX strings ([#717](https://github.com/krisarmstrong/niac-go/issues/717)) ([ffeb7b1](https://github.com/krisarmstrong/niac-go/commit/ffeb7b1ff06cd913153b2af7f616d480ff372305))
* **i18n:** pluralization + Intl APIs + locale-aware formatters ([#719](https://github.com/krisarmstrong/niac-go/issues/719)) ([b8bf03d](https://github.com/krisarmstrong/niac-go/commit/b8bf03d9c7f11f6ff52b3288365f1d02a777eb1b))


### Bug Fixes

* **ci:** honor PLAYWRIGHT_IGNORE_HTTPS_ERRORS + Lighthouse cert flag ([#722](https://github.com/krisarmstrong/niac-go/issues/722)) ([eae281f](https://github.com/krisarmstrong/niac-go/commit/eae281f896529763055cffd11c2c6d0bde67842a))

## [0.85.0](https://github.com/krisarmstrong/niac-go/compare/v0.84.0...v0.85.0) (2026-05-26)


### Features

* **api:** add GET /api/v1/license read endpoint ([#710](https://github.com/krisarmstrong/niac-go/issues/710)) ([50e16c7](https://github.com/krisarmstrong/niac-go/commit/50e16c7ad8022f3594cd9cd93f63bd60237b41fc))

## [0.84.0](https://github.com/krisarmstrong/niac-go/compare/v0.83.1...v0.84.0) (2026-05-26)


### Features

* **i18n:** add errors.license.* keys for tier-gating UI ([#709](https://github.com/krisarmstrong/niac-go/issues/709)) ([76b2a93](https://github.com/krisarmstrong/niac-go/commit/76b2a934265c8cc7f5da120cab5ee4700ff4da4f))
* **license:** add per-route feature gating framework ([#704](https://github.com/krisarmstrong/niac-go/issues/704)) ([710b566](https://github.com/krisarmstrong/niac-go/commit/710b5668dd4a65dfb57d8df3ecf269aded4ee406))
* **license:** enforce Free-tier 10-device soft cap on device create ([#706](https://github.com/krisarmstrong/niac-go/issues/706)) ([44e254f](https://github.com/krisarmstrong/niac-go/commit/44e254fbfb34817ea8628a3a51072fb5ee10e714))
* **license:** gate pcap, templates, traffic_shaping, multi_ip ([#708](https://github.com/krisarmstrong/niac-go/issues/708)) ([0c87ce2](https://github.com/krisarmstrong/niac-go/commit/0c87ce2b2b72ba3ad04529cdfce6e831272bcaa6))
* **license:** gate STP/FTP/NetBIOS protocols on device create+update ([#707](https://github.com/krisarmstrong/niac-go/issues/707)) ([fe16dab](https://github.com/krisarmstrong/niac-go/commit/fe16dab121ce82fcc91c6a4eb699ae00bdb7ebf2))

## [0.83.1](https://github.com/krisarmstrong/niac-go/compare/v0.83.0...v0.83.1) (2026-05-26)


### Bug Fixes

* **ci:** switch e2e + lighthouse to HTTPS for TLS-only daemon ([#701](https://github.com/krisarmstrong/niac-go/issues/701)) ([b8aa9a5](https://github.com/krisarmstrong/niac-go/commit/b8aa9a536d45c24f9eee89f82a0286c37a87fe6b))
* **e2e:** switch fullstack config to HTTPS for TLS-only daemon ([#698](https://github.com/krisarmstrong/niac-go/issues/698)) ([7338525](https://github.com/krisarmstrong/niac-go/commit/7338525955e53a4195e2f5ac4bb679788ef82eb1))
* **license:** add RWMutex to Manager for safe concurrent access ([#703](https://github.com/krisarmstrong/niac-go/issues/703)) ([ed0d427](https://github.com/krisarmstrong/niac-go/commit/ed0d427440f9b9dd1ea3e7725b73b781450198dc))
* **scripts:** clean up all shellcheck warnings + pin severity=warning ([#696](https://github.com/krisarmstrong/niac-go/issues/696)) ([71ac82d](https://github.com/krisarmstrong/niac-go/commit/71ac82d544bfcbf148fc34ed28e6cbb90e230adc))

## [0.83.0](https://github.com/krisarmstrong/niac-go/compare/v0.82.0...v0.83.0) (2026-05-25)


### Features

* **converter:** add struct-tag validation for Config ([#685](https://github.com/krisarmstrong/niac-go/issues/685)) ([269a924](https://github.com/krisarmstrong/niac-go/commit/269a924d6d353b5ab183c5c57928ad14709d492d)), closes [#669](https://github.com/krisarmstrong/niac-go/issues/669)


### Bug Fixes

* **ci:** inject UIBuildHash ldflag (Universal Build Contract) ([#682](https://github.com/krisarmstrong/niac-go/issues/682)) ([d37b4f9](https://github.com/krisarmstrong/niac-go/commit/d37b4f961106dba2d37053e604995ee5a7e5f98d))
* **docs:** correct PR template 'cd web' -&gt; 'cd ui' ([#683](https://github.com/krisarmstrong/niac-go/issues/683)) ([3e5dc18](https://github.com/krisarmstrong/niac-go/commit/3e5dc187f64edd9083d13374676123f34d56a26b))
* **scripts:** deploy-validate add HTTPS support + canonical port 8445 ([#692](https://github.com/krisarmstrong/niac-go/issues/692)) ([740a746](https://github.com/krisarmstrong/niac-go/commit/740a746021241fde2fbcc1bfbf6037cfe9f8ed51))

## [0.82.0](https://github.com/krisarmstrong/niac-go/compare/v0.81.1...v0.82.0) (2026-05-25)


### Features

* **license:** add offline license framework with trial and keygen contract ([#671](https://github.com/krisarmstrong/niac-go/issues/671)) ([18d7b29](https://github.com/krisarmstrong/niac-go/commit/18d7b29c5ee774889198cc0d7816ee7ddb5aa043))
* **security:** Require HTTPS unconditionally ([#1070](https://github.com/krisarmstrong/niac-go/issues/1070)) ([#663](https://github.com/krisarmstrong/niac-go/issues/663)) ([1e33b59](https://github.com/krisarmstrong/niac-go/commit/1e33b59c64981a5555211dc62c9da844027cd38f))


### Bug Fixes

* **e2e:** repair niac config-diff strict-mode + gui-daemon test.skip typo ([#661](https://github.com/krisarmstrong/niac-go/issues/661)) ([2434259](https://github.com/krisarmstrong/niac-go/commit/2434259a04a85eeb23b6b61f11d178c003d3dbe5))

## [0.81.1](https://github.com/krisarmstrong/niac-go/compare/v0.81.0...v0.81.1) (2026-05-22)


### Performance Improvements

* **e2e:** bump CI workers 1-&gt;2 and retries 2-&gt;1 ([#658](https://github.com/krisarmstrong/niac-go/issues/658)) ([65afe89](https://github.com/krisarmstrong/niac-go/commit/65afe899a529814f34dca11892374e8310230e79))

## [0.81.0](https://github.com/krisarmstrong/niac-go/compare/v0.80.0...v0.81.0) (2026-05-22)


### Features

* **theme:** adopt botanical-earth surface palette (Phase 4) ([8133adb](https://github.com/krisarmstrong/niac-go/commit/8133adbe2e7fd01d8a4125d761fc1945a9b1256f))
* **theme:** adopt canonical responsive type scale (Phase 3) ([4f411f2](https://github.com/krisarmstrong/niac-go/commit/4f411f212b6d9269dc5a2c375735d740eb0d5d28))
* **theme:** Apply 2026-05-22 brand audit — NIAC becomes indigo + 5 modules ([26bc0f0](https://github.com/krisarmstrong/niac-go/commit/26bc0f0daaeaf8682ecf1b3d9d7cadd463863116))
* **theme:** differentiate NIAC modules + flatten component primitives (Phase 6) ([06764c0](https://github.com/krisarmstrong/niac-go/commit/06764c0d2f6ef0763dc0bcaf29c1e6e0ce77eb40))
* **theme:** identity shift — NIAC becomes indigo (Phase 5) ([5a2f110](https://github.com/krisarmstrong/niac-go/commit/5a2f110c1c08d7fb47cc81322cbaede9131a8d11))
* **theme:** self-host fonts via [@fontsource-variable](https://github.com/fontsource-variable), drop Space Grotesk (Phase 2) ([0010faf](https://github.com/krisarmstrong/niac-go/commit/0010faf6c4478efc29ecf55bd49e643a2239678f))
* **theme:** swap status palette to canonical brand-tied anchors (Phase 1) ([e806bb7](https://github.com/krisarmstrong/niac-go/commit/e806bb70b37e685a8a11ce6b3b5ac30783448dad))


### Bug Fixes

* **vite:** stop inlining font assets as data: URLs (CSP fix) ([4efa048](https://github.com/krisarmstrong/niac-go/commit/4efa0488194bd380e1b09c917fc4aea41b0a2d5f))
* **vite:** Stop inlining font assets as data: URLs (CSP fix) ([35e87ca](https://github.com/krisarmstrong/niac-go/commit/35e87ca52c333f517706d6c41aec8f09c0642ddc))

## [0.80.0](https://github.com/krisarmstrong/niac-go/compare/v0.79.0...v0.80.0) (2026-05-22)


### Features

* **stories:** cover 5 context-heavy src/ui/ primitives (Wave 5 / niac-W5-2c) ([#645](https://github.com/krisarmstrong/niac-go/issues/645)) ([d427a05](https://github.com/krisarmstrong/niac-go/commit/d427a05f23b4fa6d007965a21386783a98b2b25b))
* **stories:** cover 8 more src/ui/ primitives (Wave 5 / niac-W5-2b) ([#641](https://github.com/krisarmstrong/niac-go/issues/641)) ([191b195](https://github.com/krisarmstrong/niac-go/commit/191b195f0dc60b97c7df018159760fee6f099036))
* **stories:** primitive storybook coverage for src/ui/ (Wave 5 / niac-W5-2) ([#639](https://github.com/krisarmstrong/niac-go/issues/639)) ([6fac3bb](https://github.com/krisarmstrong/niac-go/commit/6fac3bb238e9ed55b9660e0a3883242d9e5346f0))
* **ui:** scaffold storybook 10 (Wave 5 / niac-W5-1, closes [#636](https://github.com/krisarmstrong/niac-go/issues/636)) ([#638](https://github.com/krisarmstrong/niac-go/issues/638)) ([4919cf1](https://github.com/krisarmstrong/niac-go/commit/4919cf14dc5085f0f4947dd3c5d381f1784c04f2))


### Bug Fixes

* **tsconfig:** drop deprecated baseUrl from tsconfig.app.json (TS 6 compat) ([#648](https://github.com/krisarmstrong/niac-go/issues/648)) ([a55317c](https://github.com/krisarmstrong/niac-go/commit/a55317cc65a6fa53c5daa0520894c7b02ffbd2d5))

## [0.79.0](https://github.com/krisarmstrong/niac-go/compare/v0.78.1...v0.79.0) (2026-05-20)


### Features

* **auth:** sighup token rotation + scoped tokens ([#632](https://github.com/krisarmstrong/niac-go/issues/632)) ([48567a9](https://github.com/krisarmstrong/niac-go/commit/48567a94e0444d7ca06148e1b7a3d1a86ea20a1e))
* TLS by default + canonical port 8445 + HTTP redirector + default-secure non-loopback (Wave 1) ([#630](https://github.com/krisarmstrong/niac-go/issues/630)) ([ea81fff](https://github.com/krisarmstrong/niac-go/commit/ea81fff4b3a42d8837ee569f62507a9c9237e998))

## [0.78.1](https://github.com/krisarmstrong/niac-go/compare/v0.78.0...v0.78.1) (2026-05-19)


### Bug Fixes

* **ci:** add target_tag input to SLSA backfill ([#75](https://github.com/krisarmstrong/niac-go/issues/75) follow-up) ([#626](https://github.com/krisarmstrong/niac-go/issues/626)) ([520803a](https://github.com/krisarmstrong/niac-go/commit/520803ad9daf6411fa9d77fcac287df15630862d))

## [0.78.0](https://github.com/krisarmstrong/niac-go/compare/v0.77.0...v0.78.0) (2026-05-19)


### Features

* **ci:** Add provenance_only mode for SLSA backfill ([#75](https://github.com/krisarmstrong/niac-go/issues/75)) ([#623](https://github.com/krisarmstrong/niac-go/issues/623)) ([d771b28](https://github.com/krisarmstrong/niac-go/commit/d771b28e3082990896dc3546db2c935714fa7b61))

## [0.77.0](https://github.com/krisarmstrong/niac-go/compare/v0.76.2...v0.77.0) (2026-05-19)


### Features

* Graceful port fallback when canonical port is in use ([#69](https://github.com/krisarmstrong/niac-go/issues/69)) ([#621](https://github.com/krisarmstrong/niac-go/issues/621)) ([ed835bd](https://github.com/krisarmstrong/niac-go/commit/ed835bdeb6c5f37029d5dd993dd6bd4d2f9863f6))

## [0.76.2](https://github.com/krisarmstrong/niac-go/compare/v0.76.1...v0.76.2) (2026-05-19)


### Bug Fixes

* **ci:** point Lighthouse at the real served URLs ([#65](https://github.com/krisarmstrong/niac-go/issues/65)) ([#619](https://github.com/krisarmstrong/niac-go/issues/619)) ([d8c04b3](https://github.com/krisarmstrong/niac-go/commit/d8c04b3c74b8e8b7485f3c6f7c6ac9378654c824))

## [0.76.1](https://github.com/krisarmstrong/niac-go/compare/v0.76.0...v0.76.1) (2026-05-19)


### Bug Fixes

* **ci:** exclude advisory e2e + lighthouse from CI Complete needs ([#616](https://github.com/krisarmstrong/niac-go/issues/616)) ([c49b92b](https://github.com/krisarmstrong/niac-go/commit/c49b92b0fe98ce7347f40338571053ee99f8b1df))

## [0.76.0](https://github.com/krisarmstrong/niac-go/compare/v0.75.1...v0.76.0) (2026-05-19)


### Features

* **ui:** Topbar with theme toggle + color sync with stem ([#613](https://github.com/krisarmstrong/niac-go/issues/613)) ([542771f](https://github.com/krisarmstrong/niac-go/commit/542771f21c41ff49db63d03c3c7a76ebd17742eb))

## [0.75.1](https://github.com/krisarmstrong/niac-go/compare/v0.75.0...v0.75.1) (2026-05-18)


### Bug Fixes

* **release:** replace broken SLSA generator with attest-build-provenance ([#611](https://github.com/krisarmstrong/niac-go/issues/611)) ([4471a33](https://github.com/krisarmstrong/niac-go/commit/4471a331ee60a9bd7ae024e311b8b79f484f8291))

## [0.75.0](https://github.com/krisarmstrong/niac-go/compare/v0.74.0...v0.75.0) (2026-05-18)


### Features

* dev-run target + product favicon + SPDX header migration ([#607](https://github.com/krisarmstrong/niac-go/issues/607)) ([557f3a9](https://github.com/krisarmstrong/niac-go/commit/557f3a97647a8e6883eab8b370e73f8cd99a5c26))
* **i18n:** add Spanish (es) locale with full namespace parity ([#608](https://github.com/krisarmstrong/niac-go/issues/608)) ([d04c911](https://github.com/krisarmstrong/niac-go/commit/d04c911a308263d12c1933d90600b3c6cd037da9))
* **ui:** add sun/moon toggle to sidebar footer + cmdk install ([#609](https://github.com/krisarmstrong/niac-go/issues/609)) ([08b2517](https://github.com/krisarmstrong/niac-go/commit/08b25171e2a29b8f7fb434c94e2a5f6563a028f8))

## [0.74.0](https://github.com/krisarmstrong/niac-go/compare/v0.73.0...v0.74.0) (2026-05-18)


### Features

* **ui:** harmonize color theme with seed/stem (MSN Green, dark default + light toggle) ([#604](https://github.com/krisarmstrong/niac-go/issues/604)) ([e475b0c](https://github.com/krisarmstrong/niac-go/commit/e475b0cdf49a482edf2ddb5d10718682309f5904))

## [0.73.0](https://github.com/krisarmstrong/niac-go/compare/v0.72.0...v0.73.0) (2026-05-18)


### Features

* **ui:** comprehensive tooltip parity — add ~16 tooltips for icon-only buttons + complex actions ([#602](https://github.com/krisarmstrong/niac-go/issues/602)) ([c80484d](https://github.com/krisarmstrong/niac-go/commit/c80484dd5f7d5d5cf5fa858ffc89933c99e9b518))

## [0.72.0](https://github.com/krisarmstrong/niac-go/compare/v0.71.0...v0.72.0) (2026-05-18)


### Features

* **ui:** comprehensive in-app help system with 7 tabbed sections ([#598](https://github.com/krisarmstrong/niac-go/issues/598)) ([7cb663e](https://github.com/krisarmstrong/niac-go/commit/7cb663ea22facd8197767b192e558090977899b8))

## [0.71.0](https://github.com/krisarmstrong/niac-go/compare/v0.70.0...v0.71.0) (2026-05-18)


### Bug Fixes

* **ci:** grant security-events: write to Security Scanning job ([#587](https://github.com/krisarmstrong/niac-go/issues/587)) ([9aa3af0](https://github.com/krisarmstrong/niac-go/commit/9aa3af0fc0be198aad6f648aaccd5258aaf468c0))


### Miscellaneous Chores

* cut v0.71.0 release with refactor + CI work ([#594](https://github.com/krisarmstrong/niac-go/issues/594)) ([ae4516e](https://github.com/krisarmstrong/niac-go/commit/ae4516e8c4ecfd5518e4005958c4ddd798e6c317))

## [Unreleased]

### Added
- Windows arm64 release binary (`niac-*-windows-arm64.zip`).
- Admin-side `--webhook-allowed-host` daemon flag (repeatable). When set,
  the alert webhook only dispatches to listed hostnames — exact-match
  allowlist, which is the canonical CodeQL barrier for `go/request-forgery`.
- `workflow_dispatch` `dry_run` input on `release.yml`. Dispatching with
  `dry_run=true` (default) builds and signs every artifact and uploads
  them as a workflow artifact for inspection, without publishing a
  GitHub release.
- `.gitattributes` (LF defaults, binary classification, linguist hints).
- `CODE_OF_CONDUCT.md` (Contributor Covenant 2.1).

### Changed
- **Packaging:** `release.yml` and the local `make deb` / `make rpm`
  targets both now use `nfpm` instead of `dpkg-deb` + `rpmbuild`. nfpm is
  pure Go, so cross-arch RPMs build cleanly without rpmrc/platform
  plumbing — the v0.66.34 arm64 .rpm gap is closed.
- **Dependency:** `github.com/google/gopacket v1.1.19` →
  `github.com/gopacket/gopacket v1.5.0` (maintained fork). Unblocks
  windows/arm64 (the upstream lacked `defs_windows_arm64.go`) and
  catches up to two years of security and protocol fixes.
- **CI workflows:** every multi-trigger workflow now has a concurrency
  group so rapid pushes cancel stale runs (`release.yml` keeps
  `cancel-in-progress: false` so back-to-back tags don't cancel each
  other — they're independent versions).
- **Action versions:** standardised to current latest majors —
  `actions/checkout@v6`, `actions/upload-artifact@v7`,
  `actions/download-artifact@v8`, `actions/setup-python@v6`,
  `actions/github-script@v9`, `aquasecurity/trivy-action@0.36.0`,
  `securego/gosec@v2.26.1`, `sigstore/cosign-installer@v4`,
  `softprops/action-gh-release@v3`, `codecov/codecov-action@v6`.
- **RPM filename:** uses canonical `x86_64` / `aarch64` instead of
  `amd64` / `arm64` (in-package `Architecture:` header was already
  correct via nfpm's translation).
- **Dependabot:** added the npm ecosystem (`/ui`), grouped per-ecosystem,
  conventional-commit prefixes (`chore(deps)`, `chore(ci)`,
  `chore(ui-deps)`).
- Migrated `pkg/` packages to `internal/` for better encapsulation.
- Renamed `pkg/httpapi` → `internal/api`.
- Moved `pkg/snmp` → `internal/protocols/snmp`.
- Renamed `test/` → `tests/` for consistency.
- Renamed `ui/src/context/` → `ui/src/contexts/` for consistency.

### Removed
- `.github/workflows/release-please.yml` — pointed `extra-files: VERSION`
  at a non-existent file, has been a no-op since it landed.
- `deploy/deb/{control,postinst,prerm,postrm}` and
  `deploy/rpm/niac.spec` — fully replaced by `.nfpm.yaml` and shared
  scripts under `deploy/nfpm/`.

### Security
- 15 of 16 open CodeQL alerts cleared in code (path-injection inline
  barriers at every filesystem sink, integer/allocation bound checks,
  SSRF host check inlined at the dispatch site). The remaining
  `go/request-forgery` alert was dismissed pending deployments adopting
  the new `--webhook-allowed-host` allowlist; tracked in #484.

## [0.66.35] - 2026-05-06

### Changed
- First release on the new nfpm-based pipeline. `release.yml` no longer
  invokes `dpkg-deb` or `rpmbuild`; both .deb and .rpm are produced from
  a single `.nfpm.yaml` per Linux arch.
- Replaces six earlier release attempts (v0.66.27..34) that all got
  stuck on Ubuntu's `rpm` cross-arch limitations.

### Added
- arm64 `.rpm` artifact restored (was dropped in v0.66.34 as a
  workaround).
- 32 signed artifacts: tar.gz + deb + rpm (linux), tar.gz + pkg (macOS),
  zip (windows-amd64), each with cosign bundle and CycloneDX SBOM, plus
  a signed `checksums.txt`.

## [0.66.27] – [0.66.34] - 2026-05-05/06 (release pipeline iteration)

These tags exist on the repository but did not produce GitHub releases.
Each one was a fix to the release pipeline, surfaced by the next stage
of the matrix:

- **0.66.27** — initial post-PR-#483 tag; broke on a dead `cp -r ui/dist/*`
  step in `release.yml` (vite emits straight to `internal/api/ui` so
  `ui/dist/` never exists).
- **0.66.28** — fixed the `ui/dist` copy; arm64 still failed because
  the apt-source-restriction `sed` mangled `microsoft-prod.list`.
- **0.66.29** — sed now skips pre-bracketed `deb [...]` lines; arm64
  still failed because Ubuntu noble's deb822 `ubuntu.sources` wasn't
  arch-restricted.
- **0.66.30** — replaced `ubuntu.sources` with explicit `[arch=amd64]`
  `.list` entries; arm64 still failed at `rpmbuild --target aarch64`.
- **0.66.31** — `--target aarch64-linux`; same rpmbuild error (Ubuntu's
  `rpm` package doesn't ship aarch64 platform configs).
- **0.66.32** — bypassed `--target` with explicit `_arch`/`_target_*`
  defines; same error (rpmrc itself gates valid build archs).
- **0.66.33** — synthesized the missing
  `/usr/lib/rpm/platform/aarch64-linux/macros` file inline; same error
  (rpmrc, not platform macros, is the gate).
- **0.66.34** — pragmatic workaround: skip arm64 .rpm entirely. Shipped
  4/5 platforms × 4 formats. The retro of this chain motivated the
  v0.66.35 nfpm migration.

The v0.66.27..33 tags remain as historical pointers to specific commits;
they are not formal releases (no GitHub release object, no published
artifacts).

## [0.1.0] - Initial Release

- Initial NIAC implementation.
