# Changelog

All notable changes to NIAC will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
