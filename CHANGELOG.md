# Changelog

All notable changes to NIAC will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
