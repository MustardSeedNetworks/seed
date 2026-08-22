# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.213.10](https://github.com/MustardSeedNetworks/seed/compare/v0.213.9...v0.213.10) (2026-08-22)


### Bug Fixes

* **deps:** let npm regenerate the lockfile, which unblocks Renovate ([#2011](https://github.com/MustardSeedNetworks/seed/issues/2011)) ([8665e36](https://github.com/MustardSeedNetworks/seed/commit/8665e36bbd7db3853ef6e385a105c9c7d871354c))

## [0.213.9](https://github.com/MustardSeedNetworks/seed/compare/v0.213.8...v0.213.9) (2026-08-22)


### Continuous Integration

* exempt bots from the issue-title lint ([#2008](https://github.com/MustardSeedNetworks/seed/issues/2008)) ([c82f58d](https://github.com/MustardSeedNetworks/seed/commit/c82f58d102d16ced6206e995c2b2293d06e53c31))


### Miscellaneous

* **deps:** update dependency @biomejs/biome to v2.5.9 ([#2009](https://github.com/MustardSeedNetworks/seed/issues/2009)) ([9424489](https://github.com/MustardSeedNetworks/seed/commit/9424489b231951749913444b8115fb58cda6da16))

## [0.213.8](https://github.com/MustardSeedNetworks/seed/compare/v0.213.7...v0.213.8) (2026-08-21)


### Miscellaneous

* **deps:** sweep the frontend dependencies in one change ([#2005](https://github.com/MustardSeedNetworks/seed/issues/2005)) ([191b5bb](https://github.com/MustardSeedNetworks/seed/commit/191b5bb0521c4b1fa8ac343e09f70b519efa165c))

## [0.213.7](https://github.com/MustardSeedNetworks/seed/compare/v0.213.6...v0.213.7) (2026-08-21)


### Bug Fixes

* **deps:** update dependency @hookform/resolvers to v5.9.1 ([#1987](https://github.com/MustardSeedNetworks/seed/issues/1987)) ([f00df75](https://github.com/MustardSeedNetworks/seed/commit/f00df75e328e1a7720880cf0a4ceaa26fd84e340))
* **deps:** update dependency immer to v11.1.17 ([#1965](https://github.com/MustardSeedNetworks/seed/issues/1965)) ([4588564](https://github.com/MustardSeedNetworks/seed/commit/4588564ab4647e06b1d0dc811b7b6ad9a286f935))
* **deps:** update dependency lucide-react to v1.32.0 ([#2000](https://github.com/MustardSeedNetworks/seed/issues/2000)) ([ee90789](https://github.com/MustardSeedNetworks/seed/commit/ee9078948d932f64f6838c67d0af3b91a0b46519))


### Miscellaneous

* **deps:** update module honnef.co/go/tools to v0.8.1 ([#2004](https://github.com/MustardSeedNetworks/seed/issues/2004)) ([74ae649](https://github.com/MustardSeedNetworks/seed/commit/74ae64946351544c903ec509a1be23d0121ce4f6))
* **deps:** update storybook monorepo to v10.5.9 ([#1855](https://github.com/MustardSeedNetworks/seed/issues/1855)) ([7e0a1b5](https://github.com/MustardSeedNetworks/seed/commit/7e0a1b5d291495969762f91fed9349302ba7a8e2))

## [0.213.6](https://github.com/MustardSeedNetworks/seed/compare/v0.213.5...v0.213.6) (2026-08-21)


### Code Refactoring

* **database:** canonicalize SNMP device credentials ([#1998](https://github.com/MustardSeedNetworks/seed/issues/1998)) ([ad11d26](https://github.com/MustardSeedNetworks/seed/commit/ad11d260ed5ab938b67c2390eafd4884e220c874))


### Continuous Integration

* refuse to start tests while orphaned test binaries are running ([#1997](https://github.com/MustardSeedNetworks/seed/issues/1997)) ([c39a7ae](https://github.com/MustardSeedNetworks/seed/commit/c39a7ae170c5ce01747b34568879803d46a5e076))

## [0.213.5](https://github.com/MustardSeedNetworks/seed/compare/v0.213.4...v0.213.5) (2026-08-21)


### Features

* **auth:** carry the session's owning client as an unforgeable claim ([#1992](https://github.com/MustardSeedNetworks/seed/issues/1992)) ([f14cf1c](https://github.com/MustardSeedNetworks/seed/commit/f14cf1c24808b4ce793ae42f6846928de3d1c80a)), closes [#1797](https://github.com/MustardSeedNetworks/seed/issues/1797)
* **build:** enable the React Compiler, keeping every existing memo ([#1993](https://github.com/MustardSeedNetworks/seed/issues/1993)) ([87cf357](https://github.com/MustardSeedNetworks/seed/commit/87cf3570622fbdc2bf32389e753837c20c91cdb7))


### Bug Fixes

* **test:** run the React Compiler in vitest, so tests exercise what ships ([#1995](https://github.com/MustardSeedNetworks/seed/issues/1995)) ([421beef](https://github.com/MustardSeedNetworks/seed/commit/421beeffd424a5ee7d999f26573a2b7b6f667bc4))


### Code Refactoring

* **polling:** separate scheduler and tenant-scoped management seams ([#1996](https://github.com/MustardSeedNetworks/seed/issues/1996)) ([1cf668c](https://github.com/MustardSeedNetworks/seed/commit/1cf668c91db314438502ef985e03c2757acbcbba))


### Miscellaneous

* **deps:** update module github.com/golangci/golangci-lint/v2/cmd/golangci-lint to v2.13.1 ([#1989](https://github.com/MustardSeedNetworks/seed/issues/1989)) ([82080d1](https://github.com/MustardSeedNetworks/seed/commit/82080d1cb997e585cf40e42dfab74ca72441e2f1))
* **deps:** update module honnef.co/go/tools to v0.8.0 ([#1990](https://github.com/MustardSeedNetworks/seed/issues/1990)) ([ffea7e7](https://github.com/MustardSeedNetworks/seed/commit/ffea7e7e4091d9234d2df45155a11bbad14153bb))
* **ts:** adopt verbatimModuleSyntax and gate the fleet's strictness contract ([#1988](https://github.com/MustardSeedNetworks/seed/issues/1988)) ([6d2c696](https://github.com/MustardSeedNetworks/seed/commit/6d2c6965cbb9e635b0e8da74eb9ccc534a8265fa))

## [0.213.4](https://github.com/MustardSeedNetworks/seed/compare/v0.213.3...v0.213.4) (2026-08-20)


### Bug Fixes

* **deps:** bump golang.org/x/mod to v0.40.0 for CVE-2026-56864/56865 ([#1983](https://github.com/MustardSeedNetworks/seed/issues/1983)) ([5eb06cb](https://github.com/MustardSeedNetworks/seed/commit/5eb06cbf41787f4fec9abff0e1f9af76b8f7fc83)), closes [#1982](https://github.com/MustardSeedNetworks/seed/issues/1982)
* **lint:** scope Biome's dist, build and coverage excludes to where they land ([#1980](https://github.com/MustardSeedNetworks/seed/issues/1980)) ([36c309c](https://github.com/MustardSeedNetworks/seed/commit/36c309c9cf404dc94d883c3984d2d1e529fec5b4))
* **polling:** fail closed at the SNMP dialer, not just at the resolver ([#1977](https://github.com/MustardSeedNetworks/seed/issues/1977)) ([7fb42ab](https://github.com/MustardSeedNetworks/seed/commit/7fb42abd60b9d8ac0d691db315f650e1414c52b2)), closes [#1798](https://github.com/MustardSeedNetworks/seed/issues/1798)
* **test:** give the airspace fixture the BSSView field it claims to have ([#1985](https://github.com/MustardSeedNetworks/seed/issues/1985)) ([ca893a2](https://github.com/MustardSeedNetworks/seed/commit/ca893a2af04f27f7968b4f17c0f17cefa5a6900f)), closes [#1984](https://github.com/MustardSeedNetworks/seed/issues/1984)
* **ui:** let a whole-page absence fill the page instead of one grid cell ([#1978](https://github.com/MustardSeedNetworks/seed/issues/1978)) ([802dc3a](https://github.com/MustardSeedNetworks/seed/commit/802dc3aee73dda598fec8f6398b4e34c0e725175))

## [0.213.3](https://github.com/MustardSeedNetworks/seed/compare/v0.213.2...v0.213.3) (2026-08-20)


### Features

* **ui:** adopt the Card grid archetype and make card absence explain itself ([#1976](https://github.com/MustardSeedNetworks/seed/issues/1976)) ([95891d2](https://github.com/MustardSeedNetworks/seed/commit/95891d2ca9834133aced87c613ea1c7a705aab86))


### Bug Fixes

* **ci:** run the E2E jobs in Playwright's container image ([#1961](https://github.com/MustardSeedNetworks/seed/issues/1961)) ([bf22ddb](https://github.com/MustardSeedNetworks/seed/commit/bf22ddb8df79b587699547276ea42581aa4d0137))
* **deps:** update module github.com/stretchr/testify to v1.12.1 ([#1970](https://github.com/MustardSeedNetworks/seed/issues/1970)) ([b190b47](https://github.com/MustardSeedNetworks/seed/commit/b190b47adeee6c488380eb1efecf24f210dd7b0d))
* **deps:** update module modernc.org/sqlite to v1.57.0 ([#1967](https://github.com/MustardSeedNetworks/seed/issues/1967)) ([dd8f0d3](https://github.com/MustardSeedNetworks/seed/commit/dd8f0d32885ff8f0f25c58d9daebd9b2ce7eb228))
* **polling:** scope SNMP credential reads to the target's client ([#1971](https://github.com/MustardSeedNetworks/seed/issues/1971)) ([0ec99c2](https://github.com/MustardSeedNetworks/seed/commit/0ec99c2c29cc9867b6693975ccc7872c8ca56855))


### Code Refactoring

* **topology:** adopt the List + detail archetype ([#1973](https://github.com/MustardSeedNetworks/seed/issues/1973)) ([9017b27](https://github.com/MustardSeedNetworks/seed/commit/9017b2704187415b900ee4242dcdcfb8f6dfebd7))


### Tests

* **topology:** cover TopologyPage before the archetype refactor touches it ([#1960](https://github.com/MustardSeedNetworks/seed/issues/1960)) ([8c3ac62](https://github.com/MustardSeedNetworks/seed/commit/8c3ac620d1aa9fe699e16be946bf56e2530c6a28))


### Continuous Integration

* reconcile skipped releases with a 3-hourly release-please run ([#1959](https://github.com/MustardSeedNetworks/seed/issues/1959)) ([555a3e2](https://github.com/MustardSeedNetworks/seed/commit/555a3e2d2c1e2a6ab97332a0c8a15f30e05f5039))


### Miscellaneous

* **deps:** lock file maintenance ([#1963](https://github.com/MustardSeedNetworks/seed/issues/1963)) ([1ad5d08](https://github.com/MustardSeedNetworks/seed/commit/1ad5d0873c25f0b2515afbce6023bbec72c81ad2))
* **deps:** update module github.com/golangci/golangci-lint/v2/cmd/golangci-lint to v2.13.0 ([#1975](https://github.com/MustardSeedNetworks/seed/issues/1975)) ([659d9c7](https://github.com/MustardSeedNetworks/seed/commit/659d9c71e26ab3735e739bcaf78d3c74a12b633f))

## [0.213.2](https://github.com/MustardSeedNetworks/seed/compare/v0.213.1...v0.213.2) (2026-08-18)


### Bug Fixes

* **ci:** replace the hand-rolled apt wrappers with the fleet composite ([#1958](https://github.com/MustardSeedNetworks/seed/issues/1958)) ([12568ec](https://github.com/MustardSeedNetworks/seed/commit/12568ec742d1d77e29460aa805f3ed654c565637))
* **ci:** wait out dpkg lock contention instead of retrying into it ([#1956](https://github.com/MustardSeedNetworks/seed/issues/1956)) ([4c7c429](https://github.com/MustardSeedNetworks/seed/commit/4c7c42956981ed38a6523fa6383194f1a83b33cc))


### Code Refactoring

* **ui:** delete the dead card type model and settle the status vocabulary ([#1953](https://github.com/MustardSeedNetworks/seed/issues/1953)) ([ba21e98](https://github.com/MustardSeedNetworks/seed/commit/ba21e98bbdea94e32dcbfa96602ca35322986bbe))

## [0.213.1](https://github.com/MustardSeedNetworks/seed/compare/v0.213.0...v0.213.1) (2026-08-18)


### Features

* **ui:** point the header's help at the drawer and add the List + detail archetype ([#1948](https://github.com/MustardSeedNetworks/seed/issues/1948)) ([650c779](https://github.com/MustardSeedNetworks/seed/commit/650c77908dcded91b5e536493a0c71d6fbe28554))

## [0.213.0](https://github.com/MustardSeedNetworks/seed/compare/v0.212.11...v0.213.0) (2026-08-17)


### ⚠ BREAKING CHANGES

* **wifi:** remove survey API runtime and survey_samples table ([#1934](https://github.com/MustardSeedNetworks/seed/issues/1934))

### Features

* **discovery:** record the VLAN each neighbour was heard on ([#1935](https://github.com/MustardSeedNetworks/seed/issues/1935)) ([c0042dd](https://github.com/MustardSeedNetworks/seed/commit/c0042dde587d4c3fdd3b5fb526612f541659c924)), closes [#1929](https://github.com/MustardSeedNetworks/seed/issues/1929)
* **polling:** resolve SNMP credentials and fail closed without them ([#1933](https://github.com/MustardSeedNetworks/seed/issues/1933)) ([26b0b09](https://github.com/MustardSeedNetworks/seed/commit/26b0b09c604de758c8becf7e75f52e42f90cb3c1))


### Bug Fixes

* **deps:** update module github.com/stretchr/testify to v1.12.0 ([#1926](https://github.com/MustardSeedNetworks/seed/issues/1926)) ([b343797](https://github.com/MustardSeedNetworks/seed/commit/b343797fd253c9faa42427a23e6403d83b441125))
* **discovery:** decode LLDP and CDP on 802.1Q trunks ([#1923](https://github.com/MustardSeedNetworks/seed/issues/1923)) ([49bbb09](https://github.com/MustardSeedNetworks/seed/commit/49bbb095484e669104e84fff15e4fca15ceea054)), closes [#1922](https://github.com/MustardSeedNetworks/seed/issues/1922)
* **discovery:** parse the real EDP header, not an invented one ([#1938](https://github.com/MustardSeedNetworks/seed/issues/1938)) ([68f08e2](https://github.com/MustardSeedNetworks/seed/commit/68f08e2a22b7db1ba8eed145dd2e517100cab060)), closes [#1937](https://github.com/MustardSeedNetworks/seed/issues/1937)


### Code Refactoring

* **wifi:** remove survey API runtime and survey_samples table ([#1934](https://github.com/MustardSeedNetworks/seed/issues/1934)) ([5736509](https://github.com/MustardSeedNetworks/seed/commit/573650946f8fac17cb63d8572df77c367461f6ce))


### Documentation

* drop Lighthouse from the CI job reference ([#1931](https://github.com/MustardSeedNetworks/seed/issues/1931)) ([ef1b346](https://github.com/MustardSeedNetworks/seed/commit/ef1b34684b9b56ddcae35341c57ef47b8ae7a152)), closes [#1911](https://github.com/MustardSeedNetworks/seed/issues/1911)


### Continuous Integration

* make CI conformance a blocking gate ([#1924](https://github.com/MustardSeedNetworks/seed/issues/1924)) ([a725441](https://github.com/MustardSeedNetworks/seed/commit/a725441450bdf9abe248f62b3d851e8a06ceb218))


### Miscellaneous

* **deps:** lock file maintenance ([#1919](https://github.com/MustardSeedNetworks/seed/issues/1919)) ([95e1b78](https://github.com/MustardSeedNetworks/seed/commit/95e1b780f428964b3d9d09d378a2232bd005cdce))
* remove dead SNMPCollector.CollectBatch and CollectorResult ([#1930](https://github.com/MustardSeedNetworks/seed/issues/1930)) ([c73087c](https://github.com/MustardSeedNetworks/seed/commit/c73087c5b736fc3f64acff6121e28f570970e57d))

## [0.212.11](https://github.com/MustardSeedNetworks/seed/compare/v0.212.10...v0.212.11) (2026-08-17)


### Tests

* **e2e:** drop duplicated smoke coverage and a test that could not fail ([#1913](https://github.com/MustardSeedNetworks/seed/issues/1913)) ([4f48ce4](https://github.com/MustardSeedNetworks/seed/commit/4f48ce4aa788623be1d95a05c7b93d2266c3c827))
* **storybook:** gate all passing stories instead of one synthetic story ([#1917](https://github.com/MustardSeedNetworks/seed/issues/1917)) ([a0d2daa](https://github.com/MustardSeedNetworks/seed/commit/a0d2daa11da18f36b24d1d2aea162f025adb1b57))


### Continuous Integration

* drop Lighthouse ([#1912](https://github.com/MustardSeedNetworks/seed/issues/1912)) ([bacf0b4](https://github.com/MustardSeedNetworks/seed/commit/bacf0b4cce493032cec14bb90a98dba428d2bdcf))


### Miscellaneous

* **deps:** update module github.com/google/go-licenses to v2 ([#1910](https://github.com/MustardSeedNetworks/seed/issues/1910)) ([4968dc5](https://github.com/MustardSeedNetworks/seed/commit/4968dc5eb9683a86d2a2144600eb0f462d01b4c6))

## [0.212.10](https://github.com/MustardSeedNetworks/seed/compare/v0.212.9...v0.212.10) (2026-08-16)


### Bug Fixes

* **ci:** give markdown lint a base ref in the merge queue ([#1901](https://github.com/MustardSeedNetworks/seed/issues/1901)) ([3c2d3e8](https://github.com/MustardSeedNetworks/seed/commit/3c2d3e8c2717b6345dceaa10aef7df1bd68f1b40))


### Miscellaneous

* un-fake the revive doc-comment gate and satisfy it ([#1905](https://github.com/MustardSeedNetworks/seed/issues/1905)) ([a6643c0](https://github.com/MustardSeedNetworks/seed/commit/a6643c02dfc779a79af91201ffc57fd02ce5d756))

## [0.212.9](https://github.com/MustardSeedNetworks/seed/compare/v0.212.8...v0.212.9) (2026-08-16)


### Continuous Integration

* make required checks report on merge_group ([#1898](https://github.com/MustardSeedNetworks/seed/issues/1898)) ([28bba41](https://github.com/MustardSeedNetworks/seed/commit/28bba41e4e2d932f0eb76d33c968800494db8112))
* stop PRs writing their own cache copies ([#1896](https://github.com/MustardSeedNetworks/seed/issues/1896)) ([021f6c1](https://github.com/MustardSeedNetworks/seed/commit/021f6c1c44bfee5b615c24a78a250b32f7b91dc4))

## [0.212.8](https://github.com/MustardSeedNetworks/seed/compare/v0.212.7...v0.212.8) (2026-08-16)


### Bug Fixes

* **release:** skip apt when the build toolchain is already present ([#1881](https://github.com/MustardSeedNetworks/seed/issues/1881)) ([8669da4](https://github.com/MustardSeedNetworks/seed/commit/8669da4db02fc9d6f79207808a4af40153f77fd9))

## [0.212.7](https://github.com/MustardSeedNetworks/seed/compare/v0.212.6...v0.212.7) (2026-08-16)


### Bug Fixes

* **lint:** align local golangci-lint with the version CI runs ([#1888](https://github.com/MustardSeedNetworks/seed/issues/1888)) ([9038f12](https://github.com/MustardSeedNetworks/seed/commit/9038f121dee107e943980a3916c86163b966ae0b))


### Continuous Integration

* declare Go tools in go.mod and drop redundant work ([#1883](https://github.com/MustardSeedNetworks/seed/issues/1883)) ([a81f855](https://github.com/MustardSeedNetworks/seed/commit/a81f855a1ed7179bf42cd7e77dadfaed19771acb))
* declare which jobs deliberately do not gate a merge ([#1893](https://github.com/MustardSeedNetworks/seed/issues/1893)) ([f6beb5f](https://github.com/MustardSeedNetworks/seed/commit/f6beb5fcf9dfb8102e44043cf72a509e64e0cde7))


### Miscellaneous

* **ci:** remove dependabot, Renovate covers the same ecosystems ([#1885](https://github.com/MustardSeedNetworks/seed/issues/1885)) ([9ab267e](https://github.com/MustardSeedNetworks/seed/commit/9ab267eee2a6e5671fe87aca96fda4c5517088f8))
* **deps:** update module github.com/golangci/golangci-lint/v2/cmd/golangci-lint to v2.12.2 ([#1886](https://github.com/MustardSeedNetworks/seed/issues/1886)) ([f064427](https://github.com/MustardSeedNetworks/seed/commit/f064427e15ef621e5936cc9a104c173f55ccf94e))
* **release:** drop the no-op trigger-release job ([#1891](https://github.com/MustardSeedNetworks/seed/issues/1891)) ([173fee3](https://github.com/MustardSeedNetworks/seed/commit/173fee3a93b254aff16cf820551d9b2c6c3ca338))

## [0.212.6](https://github.com/MustardSeedNetworks/seed/compare/v0.212.5...v0.212.6) (2026-08-16)


### Bug Fixes

* **ci:** bound apt steps so a stalled mirror fails fast ([#1878](https://github.com/MustardSeedNetworks/seed/issues/1878)) ([1419165](https://github.com/MustardSeedNetworks/seed/commit/1419165ab00f0c7a80333282f0a60f9f1799d43e))
* **ci:** bound apt waits in Playwright OS-deps step ([#1876](https://github.com/MustardSeedNetworks/seed/issues/1876)) ([33546e5](https://github.com/MustardSeedNetworks/seed/commit/33546e580c1f4b42fee80a3e72513be50adf3941))

## [0.212.5](https://github.com/MustardSeedNetworks/seed/compare/v0.212.4...v0.212.5) (2026-08-15)


### Miscellaneous

* **deps:** take codeql-action to v4.37.7 ([#1873](https://github.com/MustardSeedNetworks/seed/issues/1873)) ([178f9e5](https://github.com/MustardSeedNetworks/seed/commit/178f9e557787618370c6e0d4c1a6b88be29c06f0))

## [0.212.4](https://github.com/MustardSeedNetworks/seed/compare/v0.212.3...v0.212.4) (2026-08-15)


### Bug Fixes

* **ci:** exempt bot PRs from the required PR-body check ([#1847](https://github.com/MustardSeedNetworks/seed/issues/1847)) ([f6e2b1e](https://github.com/MustardSeedNetworks/seed/commit/f6e2b1e570271662f7b89f8c28aaefaf8fc7a6cf))
* **deps:** update dependency immer to v11.1.16 ([#1842](https://github.com/MustardSeedNetworks/seed/issues/1842)) ([1121303](https://github.com/MustardSeedNetworks/seed/commit/1121303fe07852eb472871bb24043d0d3bcd1bf6))
* **deps:** update go dependencies ([#1845](https://github.com/MustardSeedNetworks/seed/issues/1845)) ([926e9ae](https://github.com/MustardSeedNetworks/seed/commit/926e9aee194bd20f08fcf7a2b23780f2ca5aac52))
* **release:** stop requesting an App permission that is not granted ([#1866](https://github.com/MustardSeedNetworks/seed/issues/1866)) ([5f01cbb](https://github.com/MustardSeedNetworks/seed/commit/5f01cbb5456f6048ff6655321d8cb9e910694a50))


### Continuous Integration

* cache Playwright browsers and refresh CI.md ([#1834](https://github.com/MustardSeedNetworks/seed/issues/1834)) ([adc54a5](https://github.com/MustardSeedNetworks/seed/commit/adc54a5dd0b75d9f21e75d442742f825724ffd45))
* pin Node via the setup-node composite everywhere ([#1836](https://github.com/MustardSeedNetworks/seed/issues/1836)) ([8b6ad41](https://github.com/MustardSeedNetworks/seed/commit/8b6ad41aba5502a9fcf62846ad7b48463cb80c75))
* scope workflow permissions to jobs and gate on zizmor ([#1833](https://github.com/MustardSeedNetworks/seed/issues/1833)) ([1d9abe3](https://github.com/MustardSeedNetworks/seed/commit/1d9abe343240bc5d3319e190f1ecdc6673b1dccb))


### Miscellaneous

* **deps:** lock file maintenance ([#1861](https://github.com/MustardSeedNetworks/seed/issues/1861)) ([07a09e0](https://github.com/MustardSeedNetworks/seed/commit/07a09e03876250205d941d64f5fe2e644948630d))
* **deps:** lock file maintenance ([#1869](https://github.com/MustardSeedNetworks/seed/issues/1869)) ([4fc7e3e](https://github.com/MustardSeedNetworks/seed/commit/4fc7e3e14eb28edf2013da793e1cb900e9fbdc3c))
* **deps:** take remaining dependencies to latest ([#1864](https://github.com/MustardSeedNetworks/seed/issues/1864)) ([4a99249](https://github.com/MustardSeedNetworks/seed/commit/4a99249e84f5b0c08607369012c5fb7feb73f007))
* **deps:** take the frontend toolchain to latest and adopt TypeScript 7 ([#1852](https://github.com/MustardSeedNetworks/seed/issues/1852)) ([2928ca2](https://github.com/MustardSeedNetworks/seed/commit/2928ca2326eef0edfd9747b2884c14ff1edcef26))
* **deps:** update commitlint monorepo ([#1846](https://github.com/MustardSeedNetworks/seed/issues/1846)) ([2620f7a](https://github.com/MustardSeedNetworks/seed/commit/2620f7a6a56625a9e266515f533ed0f6b3fe181e))
* **deps:** update dependency lint-staged to v17.3.0 ([#1853](https://github.com/MustardSeedNetworks/seed/issues/1853)) ([55a9083](https://github.com/MustardSeedNetworks/seed/commit/55a90837a2d33d40cf1d3899657d8133aad7bf82))

## [0.212.3](https://github.com/MustardSeedNetworks/seed/compare/v0.212.2...v0.212.3) (2026-08-15)


### Bug Fixes

* **release:** harden release supply chain ([#1827](https://github.com/MustardSeedNetworks/seed/issues/1827)) ([7e0f617](https://github.com/MustardSeedNetworks/seed/commit/7e0f617c703ecc37a9051598da18270ecee55284))


### Continuous Integration

* cut CI wall-clock and make inert gates enforce ([#1831](https://github.com/MustardSeedNetworks/seed/issues/1831)) ([28d817f](https://github.com/MustardSeedNetworks/seed/commit/28d817f0b90b5dac34bc3694e26c81849da69014))
* **lint:** enforce full-tree Go lint ([#1828](https://github.com/MustardSeedNetworks/seed/issues/1828)) ([d969f5f](https://github.com/MustardSeedNetworks/seed/commit/d969f5fe6f3706f774c3afdfbf0582244dad5a92))


### Miscellaneous

* **discovery:** refresh embedded IEEE OUI registry (20260814) ([#1829](https://github.com/MustardSeedNetworks/seed/issues/1829)) ([6fe719a](https://github.com/MustardSeedNetworks/seed/commit/6fe719a7a27962b98e4c5b80034959b1a6376fbb))

## [0.212.2](https://github.com/MustardSeedNetworks/seed/compare/v0.212.1...v0.212.2) (2026-08-14)


### Bug Fixes

* **release:** keep Syft outside checkout ([#1825](https://github.com/MustardSeedNetworks/seed/issues/1825)) ([b2d38de](https://github.com/MustardSeedNetworks/seed/commit/b2d38dedf8baa7faebeedb231432588dd9885174))

## [0.212.1](https://github.com/MustardSeedNetworks/seed/compare/v0.212.0...v0.212.1) (2026-08-14)


### Features

* **ui:** per-item module accent colour in the sidebar (M1 follow-up) ([#1682](https://github.com/MustardSeedNetworks/seed/issues/1682)) ([2aa1f1c](https://github.com/MustardSeedNetworks/seed/commit/2aa1f1c730fd9da38315f6417dff246eee0d9558))


### Bug Fixes

* **build:** honest quality gates — propagate exit codes, fix hidden lint/race findings ([#1723](https://github.com/MustardSeedNetworks/seed/issues/1723)) ([474fcaf](https://github.com/MustardSeedNetworks/seed/commit/474fcafc4a5d974981dd6392035db2f89e67dc65))
* **csrf:** gate CSRF session key on JWT structure ([#1757](https://github.com/MustardSeedNetworks/seed/issues/1757)) ([e1dbefe](https://github.com/MustardSeedNetworks/seed/commit/e1dbefe7a6c1114bdb2dc76e7d02a962a7f50a01))
* **database:** enforce foreign keys across pool ([#1787](https://github.com/MustardSeedNetworks/seed/issues/1787)) ([a90f5cf](https://github.com/MustardSeedNetworks/seed/commit/a90f5cf266e52e11543379e3b811ed22f771c8ce))
* **license:** remove vaporware features from the Pro catalog ([#1761](https://github.com/MustardSeedNetworks/seed/issues/1761)) ([f5533b1](https://github.com/MustardSeedNetworks/seed/commit/f5533b1443dfcaaaf3b3d2b94112704af6040056))
* **release:** run Syft installation under Bash ([#1803](https://github.com/MustardSeedNetworks/seed/issues/1803)) ([5dc0f5c](https://github.com/MustardSeedNetworks/seed/commit/5dc0f5c016fc278e04ad5351160073d3ab1501cb))
* **release:** synchronize Seed version metadata ([#1824](https://github.com/MustardSeedNetworks/seed/issues/1824)) ([9925f88](https://github.com/MustardSeedNetworks/seed/commit/9925f8804ee6b2e3edb262528b3d27e8b8fcb384))
* **security:** bump Go 1.26.5 and gate Trivy SARIF ([#1747](https://github.com/MustardSeedNetworks/seed/issues/1747)) ([77378ed](https://github.com/MustardSeedNetworks/seed/commit/77378ed496106a14aa0565d121afde790530dd68))
* **security:** bump Go to 1.26.6 for seven reachable stdlib CVEs ([#1822](https://github.com/MustardSeedNetworks/seed/issues/1822)) ([f6bfe65](https://github.com/MustardSeedNetworks/seed/commit/f6bfe654596bf81ffa72547b897a30e248f0eb64))
* **security:** resolve CodeQL alerts (allocation caps, TLS inspect, secure RNG) ([#1728](https://github.com/MustardSeedNetworks/seed/issues/1728)) ([2a39e22](https://github.com/MustardSeedNetworks/seed/commit/2a39e22a6ae93d6e00843689e9bd1cdd4d887092))
* **tls:** remove ACME and plaintext serving ([#1804](https://github.com/MustardSeedNetworks/seed/issues/1804)) ([4255ea1](https://github.com/MustardSeedNetworks/seed/commit/4255ea1fe7d324a4c6e4dcec10fe8aa5b4a48a33))
* **ui:** close SEED_UI_ARCH_PLAN D-batch — distinct brand green (L1) + profile entry points (VL1) + on-brand AA (VL2) ([#1680](https://github.com/MustardSeedNetworks/seed/issues/1680)) ([af1e7d1](https://github.com/MustardSeedNetworks/seed/commit/af1e7d1496b965f3c507df7fa9cd6c8af3d7b35e))


### Code Refactoring

* **alerts:** relocate Alert/Rule/ListenerEvent rows to domain pkgs, make persistence-free (WS-B) ([#1676](https://github.com/MustardSeedNetworks/seed/issues/1676)) ([0323857](https://github.com/MustardSeedNetworks/seed/commit/0323857e9d56274430e766e76e3790415ed93b4a))
* **ci:** add domain-purity depguard rules for six capability packages (WS-C4) ([#1697](https://github.com/MustardSeedNetworks/seed/issues/1697)) ([134e9eb](https://github.com/MustardSeedNetworks/seed/commit/134e9ebd57c7fb54c09081c2fd0bed7a774deb4d))
* **ci:** empty the json-casing baseline; exempt external adapters by marker (ADR-0010 revised) ([#1684](https://github.com/MustardSeedNetworks/seed/issues/1684)) ([1df12dd](https://github.com/MustardSeedNetworks/seed/commit/1df12dd6c5a80fd8a9ade6b746aea8aabbe0be66))
* **csrf:** back CSRF with foundation module; converge keying to sha256(bearer) ([#1755](https://github.com/MustardSeedNetworks/seed/issues/1755)) ([db4dd31](https://github.com/MustardSeedNetworks/seed/commit/db4dd31954137650026e9823c8431b91663c84e4))
* **database:** decompose the discovery repository god-file by entity (WS-D) ([#1702](https://github.com/MustardSeedNetworks/seed/issues/1702)) ([57e1fbe](https://github.com/MustardSeedNetworks/seed/commit/57e1fbe40c031584e0d32cac9eb6c381416aa473))
* decompose four WS-D god-files by role (arp, users, snmp, engine) ([#1709](https://github.com/MustardSeedNetworks/seed/issues/1709)) ([cb8e937](https://github.com/MustardSeedNetworks/seed/commit/cb8e9375184d9e8881a1bb8465abcd020d19b12f))
* decompose seven more WS-D god-files by role (database, alerts, registry, netif, iperf installer) ([#1710](https://github.com/MustardSeedNetworks/seed/issues/1710)) ([ec72335](https://github.com/MustardSeedNetworks/seed/commit/ec723354cd0271bb704730784cce1556aa1c7ba3))
* **dhcp:** split lease-file parsing out of the dhcp god-file (WS-D) ([#1700](https://github.com/MustardSeedNetworks/seed/issues/1700)) ([d4c7680](https://github.com/MustardSeedNetworks/seed/commit/d4c76808c3e747ea8e94cc1a0726211d1acbe15e))
* **discovery:** decompose the fingerprint god-file by role (WS-D) ([#1708](https://github.com/MustardSeedNetworks/seed/issues/1708)) ([b90b0e2](https://github.com/MustardSeedNetworks/seed/commit/b90b0e26bed21434497c3edaea1a6bb77edef93e))
* **discovery:** decompose the traceroute god-file by protocol (WS-D) ([#1705](https://github.com/MustardSeedNetworks/seed/issues/1705)) ([05fccd7](https://github.com/MustardSeedNetworks/seed/commit/05fccd76ef4b15dcef67b17fe1bb4d481f0b5b9f))
* **health:** make health surface persistence-free (WS-B) ([#1678](https://github.com/MustardSeedNetworks/seed/issues/1678)) ([3215b97](https://github.com/MustardSeedNetworks/seed/commit/3215b97f13640d680083445865add9d5b27fcd16))
* **iperf:** decompose the iperf god-file by role (WS-D) ([#1706](https://github.com/MustardSeedNetworks/seed/issues/1706)) ([62db623](https://github.com/MustardSeedNetworks/seed/commit/62db62346e008343e74d02b7849c1c82013b1aa8))
* **license:** consume shared foundation module for license core ([#1753](https://github.com/MustardSeedNetworks/seed/issues/1753)) ([f35f069](https://github.com/MustardSeedNetworks/seed/commit/f35f0697017aa657c9530ace59996a6d148029fd))
* **polling:** relocate PollingTarget to domain pkg + orchestrator ports, make persistence-free (WS-B) ([#1675](https://github.com/MustardSeedNetworks/seed/issues/1675)) ([74c4162](https://github.com/MustardSeedNetworks/seed/commit/74c4162e637d459ad2be551b13e49e0489796b2c))
* **probe:** narrow persistence port to domain types (WS-B1) ([#1672](https://github.com/MustardSeedNetworks/seed/issues/1672)) ([e07d928](https://github.com/MustardSeedNetworks/seed/commit/e07d9285c75763bdd12bc59b3c59f10a06b6a4bb))
* **retention:** relocate rollup SQL adapters to internal/database (WS-B5) ([#1677](https://github.com/MustardSeedNetworks/seed/issues/1677)) ([7370c51](https://github.com/MustardSeedNetworks/seed/commit/7370c515dfd01046fd9c666a4a9a2c73c1edca34))
* **settings:** decompose the management god-file by role (WS-D) ([#1699](https://github.com/MustardSeedNetworks/seed/issues/1699)) ([54b3399](https://github.com/MustardSeedNetworks/seed/commit/54b33993b3a6553c525855a26abce9fe2474cd78))
* **snmp:** decompose the interface god-file by query area (WS-D) ([#1701](https://github.com/MustardSeedNetworks/seed/issues/1701)) ([cdbbc18](https://github.com/MustardSeedNetworks/seed/commit/cdbbc1828be47086d87f569761a7397402ec273d))
* **survey:** decompose the report god-file by role (WS-D) ([#1703](https://github.com/MustardSeedNetworks/seed/issues/1703)) ([b19db25](https://github.com/MustardSeedNetworks/seed/commit/b19db25c085437971690df31a103daa5f1869926))
* **survey:** decompose the survey manager god-file by role (WS-D) ([#1704](https://github.com/MustardSeedNetworks/seed/issues/1704)) ([c06374a](https://github.com/MustardSeedNetworks/seed/commit/c06374a2641f881582e3245220dba2df66b9aaf6))
* **topology:** relocate row types to domain pkgs, make persistence-free (WS-B) ([#1673](https://github.com/MustardSeedNetworks/seed/issues/1673)) ([e768fcc](https://github.com/MustardSeedNetworks/seed/commit/e768fcc7b884fd387b87faf46a799ca4740f7256))
* **ui:** function-first sidebar nav IA, retire botanical metaphor (M1, [#1452](https://github.com/MustardSeedNetworks/seed/issues/1452)) ([#1681](https://github.com/MustardSeedNetworks/seed/issues/1681)) ([546d937](https://github.com/MustardSeedNetworks/seed/commit/546d93724dd5f26a7ba984b32a0923bc61fe1ff2))
* **vuln:** decompose the scanner god-file by role (WS-D) ([#1707](https://github.com/MustardSeedNetworks/seed/issues/1707)) ([2701fef](https://github.com/MustardSeedNetworks/seed/commit/2701fefb81f63ea051e664f0688dda0dc5ecc2f9))


### Documentation

* **adr-0010:** revise to pure boundary mapping — wire is 100% camelCase, no exceptions ([#1683](https://github.com/MustardSeedNetworks/seed/issues/1683)) ([0259e3d](https://github.com/MustardSeedNetworks/seed/commit/0259e3d39c94e60887446307f33487e02fc6cde1))
* **ws-b:** close B3 — identity is the documented ADR-0024 exception ([#1679](https://github.com/MustardSeedNetworks/seed/issues/1679)) ([2bd9ab6](https://github.com/MustardSeedNetworks/seed/commit/2bd9ab6ccde0436c13ec38c172e5f3525723cb9f))


### Tests

* make Darwin cache test lint-clean ([#1805](https://github.com/MustardSeedNetworks/seed/issues/1805)) ([74297a2](https://github.com/MustardSeedNetworks/seed/commit/74297a2704a5c2576ca9b4b5c9def4a0c4fbe231))
* make end-to-end validation hermetic ([#1786](https://github.com/MustardSeedNetworks/seed/issues/1786)) ([fa4c15b](https://github.com/MustardSeedNetworks/seed/commit/fa4c15b6ae211a4d1a7b4a7b88eb63be3fcb2c1e))
* **ui:** gate Storybook interactions and accessibility ([#1802](https://github.com/MustardSeedNetworks/seed/issues/1802)) ([22addfb](https://github.com/MustardSeedNetworks/seed/commit/22addfb0ba66cc8150fc66b07f18ed99d519904a))


### Continuous Integration

* enforce banned vocabulary policy ([#1801](https://github.com/MustardSeedNetworks/seed/issues/1801)) ([071b260](https://github.com/MustardSeedNetworks/seed/commit/071b260e08d0f289b1169125ebf1b91b9dea857c))
* **governance:** exempt release-please PRs from human PR-body template ([#1741](https://github.com/MustardSeedNetworks/seed/issues/1741)) ([e083074](https://github.com/MustardSeedNetworks/seed/commit/e083074a5012085b678f451f8fa5a8c55dfbf5d0))
* **license-check:** consume the fleet-shared reusable workflow ([#1758](https://github.com/MustardSeedNetworks/seed/issues/1758)) ([2f2cdfe](https://github.com/MustardSeedNetworks/seed/commit/2f2cdfe7c18f8e6e4abc5647f783c0c2f6e9abc0))
* **perf:** build UI once, add job timeouts, least-privilege ([#1733](https://github.com/MustardSeedNetworks/seed/issues/1733)) ([02e2a17](https://github.com/MustardSeedNetworks/seed/commit/02e2a17bc8015357b8a61ad16e8242ca17c8526f))
* **release:** use msn-ci-bot App token for release-please ([#1745](https://github.com/MustardSeedNetworks/seed/issues/1745)) ([515b298](https://github.com/MustardSeedNetworks/seed/commit/515b298f7c21e49b850f617576d15f805d38bc9f))
* **release:** verify primary artifact coverage ([#1800](https://github.com/MustardSeedNetworks/seed/issues/1800)) ([fd52fc6](https://github.com/MustardSeedNetworks/seed/commit/fd52fc6662938054ee8af13605982df783cb20a6))
* **security:** add Semgrep SAST gate, pin curl|sh install ([#1737](https://github.com/MustardSeedNetworks/seed/issues/1737)) ([ecbc6ec](https://github.com/MustardSeedNetworks/seed/commit/ecbc6ec12d383d3e34fe9408f7d387d90308af9b)), closes [#1736](https://github.com/MustardSeedNetworks/seed/issues/1736)
* **semgrep:** consume the fleet-shared reusable Semgrep workflow ([#1760](https://github.com/MustardSeedNetworks/seed/issues/1760)) ([a88ef4d](https://github.com/MustardSeedNetworks/seed/commit/a88ef4d32e4e379ef0077f5781bd866850aa0c43))


### Miscellaneous

* **build:** remove Docker/GHCR publishing ([#1729](https://github.com/MustardSeedNetworks/seed/issues/1729)) ([513e1e0](https://github.com/MustardSeedNetworks/seed/commit/513e1e01d4291e13fa4e32231371644a26778e2e))
* delete dead subsystems (config-migration, wake-on-LAN, iperf auto-installer) ([#1762](https://github.com/MustardSeedNetworks/seed/issues/1762)) ([d34e6ce](https://github.com/MustardSeedNetworks/seed/commit/d34e6cecbacb7bafb322d2a699d8a8b9224292b0))
* **deps:** always-latest toolchain sweep (lockstep with stem) ([#1687](https://github.com/MustardSeedNetworks/seed/issues/1687)) ([60b54c4](https://github.com/MustardSeedNetworks/seed/commit/60b54c415ab2a120a5c39a98b0a2838859086868))
* **deps:** refresh go module graph ([#1690](https://github.com/MustardSeedNetworks/seed/issues/1690)) ([82144b5](https://github.com/MustardSeedNetworks/seed/commit/82144b5e5c5c479289df6b759aaedb3119509b06))
* **github:** standardize repo governance ([#1689](https://github.com/MustardSeedNetworks/seed/issues/1689)) ([f409599](https://github.com/MustardSeedNetworks/seed/commit/f40959915e63522e426b5c60236794122085977e))
* **license:** add license-key-circumvention clause to BUSL Additional Use Grant ([#1735](https://github.com/MustardSeedNetworks/seed/issues/1735)) ([e961114](https://github.com/MustardSeedNetworks/seed/commit/e96111461834308534e4712e5a5b4dd1586e9d23))
* **main:** release 0.211.0 ([#1698](https://github.com/MustardSeedNetworks/seed/issues/1698)) ([db007d9](https://github.com/MustardSeedNetworks/seed/commit/db007d91703bdb4ca3d92eb9ee5d64bda72154cd))
* **main:** release 0.212.0 ([#1749](https://github.com/MustardSeedNetworks/seed/issues/1749)) ([cb85fb5](https://github.com/MustardSeedNetworks/seed/commit/cb85fb5305eebc8da4824c8ceae8b5babce3d645))
* **main:** release 0.212.1 ([#1674](https://github.com/MustardSeedNetworks/seed/issues/1674)) ([33e4388](https://github.com/MustardSeedNetworks/seed/commit/33e4388b4faa927b72e176803f0271576c9fd848))
* onboard Renovate via shared org preset ([#1750](https://github.com/MustardSeedNetworks/seed/issues/1750)) ([b98dfdc](https://github.com/MustardSeedNetworks/seed/commit/b98dfdcb2a9109a5c80398a78ea7c56c54ddc0fe))
* **release:** expose release train metadata ([#1693](https://github.com/MustardSeedNetworks/seed/issues/1693)) ([58c79fc](https://github.com/MustardSeedNetworks/seed/commit/58c79fc26b96d19c213afc4773dcf54e65d7212e))
* **ui:** replace eslint references with biome ([#1695](https://github.com/MustardSeedNetworks/seed/issues/1695)) ([139b5d5](https://github.com/MustardSeedNetworks/seed/commit/139b5d5a552aeda50cc6ec4721736ef7f835d297))

## [0.212.0](https://github.com/MustardSeedNetworks/seed/compare/v0.211.0...v0.212.0) (2026-07-08)


### ⚠ BREAKING CHANGES

* **probe:** the health-check run/settings/anomalies endpoints moved from /telemetry/health-checks/* to /telemetry/probes/*. Pre-alpha, no compat.
* **probe:** /run response drops the unrendered CustomTestResult fields (security headers, redirect chains, body-match, per-endpoint vertical detail that was never populated). Pre-alpha, no compat.
* **health:** delete the dead health_check_results read-path (ADR-0026) ([#1636](https://github.com/MustardSeedNetworks/seed/issues/1636))

### Features

* **anomaly:** add error severity to the four-level ladder (ADR-0021 ph5) ([#1637](https://github.com/MustardSeedNetworks/seed/issues/1637)) ([5127ca0](https://github.com/MustardSeedNetworks/seed/commit/5127ca0fe53a59bd36e1b86aab2f2a544f872bc1))
* **anomaly:** daily-rollup census of the anomaly store (ADR-0028) ([#1650](https://github.com/MustardSeedNetworks/seed/issues/1650)) ([97aab3e](https://github.com/MustardSeedNetworks/seed/commit/97aab3ec2598a3ceb9d8976746dcf7483b058041))
* **anomaly:** persist the anomaly engine in SQL (ADR-0021 phase 1) ([#1629](https://github.com/MustardSeedNetworks/seed/issues/1629)) ([83f2ce4](https://github.com/MustardSeedNetworks/seed/commit/83f2ce46eeebf55335c3de548117dc0ca7e71527))
* **anomaly:** persist the Wi-Fi anomaly stream + load-on-start (ADR-0021 phase 3) ([#1632](https://github.com/MustardSeedNetworks/seed/issues/1632)) ([d24f896](https://github.com/MustardSeedNetworks/seed/commit/d24f896670a1c62cd4d19e1522cf3e14af3f9978))
* **anomaly:** probe is the active-monitoring anomaly producer (ADR-0025) ([#1635](https://github.com/MustardSeedNetworks/seed/issues/1635)) ([500c890](https://github.com/MustardSeedNetworks/seed/commit/500c8902a7ecfeb51ea49a71e8b8b8665d9507d7))
* **anomaly:** re-derive catalog-static impact/followUps on store read (ADR-0029) ([#1653](https://github.com/MustardSeedNetworks/seed/issues/1653)) ([be81b0d](https://github.com/MustardSeedNetworks/seed/commit/be81b0db76e31a7fccc749b0cc9b58a0f5bf3e56))
* **anomaly:** resolve probe anomalies immediately on a clean result ([#1638](https://github.com/MustardSeedNetworks/seed/issues/1638)) ([1365e9a](https://github.com/MustardSeedNetworks/seed/commit/1365e9a1b154a45c59ea1f9036e2fdf7af115957))
* **anomaly:** TTL-purge resolved anomalies in retention (ADR-0021 phase 2) ([#1630](https://github.com/MustardSeedNetworks/seed/issues/1630)) ([0ca47d7](https://github.com/MustardSeedNetworks/seed/commit/0ca47d7566e978a421b7189b3341a458dfea089e))
* **probe:** add eight health-check vertical checkers (ADR-0027 P1) ([#1641](https://github.com/MustardSeedNetworks/seed/issues/1641)) ([99649e5](https://github.com/MustardSeedNetworks/seed/commit/99649e5c55185c3ea511677401edc01a480afb7a))
* **probe:** detect TLS certificate expiry as a probe anomaly ([#1639](https://github.com/MustardSeedNetworks/seed/issues/1639)) ([a93cac5](https://github.com/MustardSeedNetworks/seed/commit/a93cac5fd260315b5fdd6009dcf7f95cfb0cff20))
* **probe:** enrich HTTP/HTTPS checker with per-phase timings and cert summary (ADR-0027 P3a) ([#1643](https://github.com/MustardSeedNetworks/seed/issues/1643)) ([94c47ab](https://github.com/MustardSeedNetworks/seed/commit/94c47ab9a20b901c8e02d5270a980f218d6f7977))
* **probe:** make the probes table store-of-record for health-check settings (ADR-0027 P2) ([#1642](https://github.com/MustardSeedNetworks/seed/issues/1642)) ([b37fe80](https://github.com/MustardSeedNetworks/seed/commit/b37fe8024b75c2398652c2376d78daeafae67de2))
* **probe:** rename health-check transport to /telemetry/probes/* (ADR-0027 P5) ([#1645](https://github.com/MustardSeedNetworks/seed/issues/1645)) ([8296cc2](https://github.com/MustardSeedNetworks/seed/commit/8296cc2a64645ae94cf38859b701da4270d6a873))
* **probe:** run health checks through the probe engine; delete the legacy stack (ADR-0027 P3+P4) ([#1644](https://github.com/MustardSeedNetworks/seed/issues/1644)) ([15bf342](https://github.com/MustardSeedNetworks/seed/commit/15bf3423b4bcb45636578cae5f291d5449d5efd8))
* **ui:** add RequireRole/RequireAdmin gate + hide user mgmt from non-admins ([#1254](https://github.com/MustardSeedNetworks/seed/issues/1254)) ([#1586](https://github.com/MustardSeedNetworks/seed/issues/1586)) ([0ee5694](https://github.com/MustardSeedNetworks/seed/commit/0ee569495c9e1ea2efbd41a5c6b7ff3207d24cde))
* **ui:** per-item module accent colour in the sidebar (M1 follow-up) ([#1682](https://github.com/MustardSeedNetworks/seed/issues/1682)) ([2aa1f1c](https://github.com/MustardSeedNetworks/seed/commit/2aa1f1c730fd9da38315f6417dff246eee0d9558))


### Bug Fixes

* **api:** require operator role on persistent-write routes ([#1631](https://github.com/MustardSeedNetworks/seed/issues/1631)) ([fc7e3f5](https://github.com/MustardSeedNetworks/seed/commit/fc7e3f5e2bb3d6af97652e74f45198c2509546ae))
* **build:** honest quality gates — propagate exit codes, fix hidden lint/race findings ([#1723](https://github.com/MustardSeedNetworks/seed/issues/1723)) ([474fcaf](https://github.com/MustardSeedNetworks/seed/commit/474fcafc4a5d974981dd6392035db2f89e67dc65))
* **database:** de-collide anomaly census test fixtures (flaky) ([#1670](https://github.com/MustardSeedNetworks/seed/issues/1670)) ([c6414cd](https://github.com/MustardSeedNetworks/seed/commit/c6414cd709511719595dce59560ccca8cffddd7b))
* **probe:** seed factory health-check targets on first run ([#1646](https://github.com/MustardSeedNetworks/seed/issues/1646)) ([4283b6e](https://github.com/MustardSeedNetworks/seed/commit/4283b6e228851c4e201db7799c556be1c4de49a1))
* **security:** bump Go 1.26.5 and gate Trivy SARIF ([#1747](https://github.com/MustardSeedNetworks/seed/issues/1747)) ([77378ed](https://github.com/MustardSeedNetworks/seed/commit/77378ed496106a14aa0565d121afde790530dd68))
* **security:** resolve CodeQL alerts (allocation caps, TLS inspect, secure RNG) ([#1728](https://github.com/MustardSeedNetworks/seed/issues/1728)) ([2a39e22](https://github.com/MustardSeedNetworks/seed/commit/2a39e22a6ae93d6e00843689e9bd1cdd4d887092))
* **ui:** close SEED_UI_ARCH_PLAN D-batch — distinct brand green (L1) + profile entry points (VL1) + on-brand AA (VL2) ([#1680](https://github.com/MustardSeedNetworks/seed/issues/1680)) ([af1e7d1](https://github.com/MustardSeedNetworks/seed/commit/af1e7d1496b965f3c507df7fa9cd6c8af3d7b35e))


### Code Refactoring

* **alerts:** relocate Alert/Rule/ListenerEvent rows to domain pkgs, make persistence-free (WS-B) ([#1676](https://github.com/MustardSeedNetworks/seed/issues/1676)) ([0323857](https://github.com/MustardSeedNetworks/seed/commit/0323857e9d56274430e766e76e3790415ed93b4a))
* **anomaly:** converge producers onto one server-owned engine (ADR-0029 P2+P3) ([#1652](https://github.com/MustardSeedNetworks/seed/issues/1652)) ([58007b4](https://github.com/MustardSeedNetworks/seed/commit/58007b4c36b3bbb011712901f52ed178b2aac6c8))
* **anomaly:** delete bespoke health detector, read unified store (ADR-0021 phase 4) ([#1633](https://github.com/MustardSeedNetworks/seed/issues/1633)) ([438e3d1](https://github.com/MustardSeedNetworks/seed/commit/438e3d1b7dbb8256ce2cf31c60fde2485d9ca627))
* **anomaly:** source on detection + source-scoped prune (ADR-0029 P1) ([#1651](https://github.com/MustardSeedNetworks/seed/issues/1651)) ([21dc576](https://github.com/MustardSeedNetworks/seed/commit/21dc576d5cdc573ce64128230615175278bff469))
* **api:** clean-hexagonal alerts retrofit (ADR-0020) ([#1612](https://github.com/MustardSeedNetworks/seed/issues/1612)) ([412e2b9](https://github.com/MustardSeedNetworks/seed/commit/412e2b984f715a861dd992b5bc99ea551235b4cc))
* **api:** clean-hexagonal discovery retrofit (ADR-0020) ([#1616](https://github.com/MustardSeedNetworks/seed/issues/1616)) ([6d0c035](https://github.com/MustardSeedNetworks/seed/commit/6d0c0352bfc159ade52df7b2a2c19244e6864cb7))
* **api:** clean-hexagonal health-monitoring retrofit (ADR-0020) ([#1618](https://github.com/MustardSeedNetworks/seed/issues/1618)) ([4678697](https://github.com/MustardSeedNetworks/seed/commit/46786970f84b34681749630cd53373144ead5258))
* **api:** clean-hexagonal network exemplar + ADR-0020 ([#1611](https://github.com/MustardSeedNetworks/seed/issues/1611)) ([dbaf8dc](https://github.com/MustardSeedNetworks/seed/commit/dbaf8dcac55b6805b00fa5695293ef8bac1005f4))
* **api:** clean-hexagonal profiles retrofit (ADR-0020) ([#1614](https://github.com/MustardSeedNetworks/seed/issues/1614)) ([b5516b2](https://github.com/MustardSeedNetworks/seed/commit/b5516b2267fce5a8a8508ea9a6059c4b3c62767b))
* **api:** clean-hexagonal settings retrofit (ADR-0020) ([#1613](https://github.com/MustardSeedNetworks/seed/issues/1613)) ([5d98097](https://github.com/MustardSeedNetworks/seed/commit/5d980976222a0ed2aad5ce459c7eb85cb4ec70b6))
* **api:** clean-hexagonal wifi retrofit (ADR-0020) ([#1615](https://github.com/MustardSeedNetworks/seed/issues/1615)) ([e903744](https://github.com/MustardSeedNetworks/seed/commit/e90374445d7b1eb66b2d1cb91cc40e9b6319973d))
* **api:** delete ServiceContainer, flatten services onto Server (D1) ([#1627](https://github.com/MustardSeedNetworks/seed/issues/1627)) ([feb198e](https://github.com/MustardSeedNetworks/seed/commit/feb198e3df9db18ebb83aa497535f550f8d18fc2))
* **api:** extract drainJobSubstrate + test the shutdown drain ([#1628](https://github.com/MustardSeedNetworks/seed/issues/1628)) ([1b8d2cf](https://github.com/MustardSeedNetworks/seed/commit/1b8d2cf5c69833b0152a44b734b41b8aeb73c213))
* **api:** finish the profiles handler strangle (ADR-0020, WS-A11c) ([#1669](https://github.com/MustardSeedNetworks/seed/issues/1669)) ([c92f82b](https://github.com/MustardSeedNetworks/seed/commit/c92f82b865c1881b4dda210bef3ab07c0f81e657))
* **api:** route handler license reads through s.licenseManager() (D1 prep) ([#1626](https://github.com/MustardSeedNetworks/seed/issues/1626)) ([54318de](https://github.com/MustardSeedNetworks/seed/commit/54318de5ef736edddf58b1776f736ca05d14f787))
* **api:** route the config-read/write residuals through use-cases (ADR-0020, WS-A11a) ([#1665](https://github.com/MustardSeedNetworks/seed/issues/1665)) ([50d92e5](https://github.com/MustardSeedNetworks/seed/commit/50d92e5a89cc4ba7ef10db8691edbd3876bcf462))
* **api:** strangle alert inbox into a use-case (ADR-0020, WS-A8) ([#1662](https://github.com/MustardSeedNetworks/seed/issues/1662)) ([c25d4cf](https://github.com/MustardSeedNetworks/seed/commit/c25d4cf20fe0f1db1dfe8c8e443f42996e5a209d))
* **api:** strangle config backup/restore into a use-case (ADR-0020, WS-A9) ([#1663](https://github.com/MustardSeedNetworks/seed/issues/1663)) ([5460177](https://github.com/MustardSeedNetworks/seed/commit/54601779883293399a0472a98be3e78463b0f799))
* **api:** strangle device-discovery settings into a use-case (ADR-0020, WS-A1) ([#1655](https://github.com/MustardSeedNetworks/seed/issues/1655)) ([f39f91e](https://github.com/MustardSeedNetworks/seed/commit/f39f91e45d81fd13f26758f52d0a366edd1e3220))
* **api:** strangle diagnostic export into a use-case (ADR-0020, WS-A10) ([#1664](https://github.com/MustardSeedNetworks/seed/issues/1664)) ([b1774de](https://github.com/MustardSeedNetworks/seed/commit/b1774de2d1963efa22e21511783cc7d617808ac8))
* **api:** strangle engine-status into a use-case over the engine registry (ADR-0020) ([#1625](https://github.com/MustardSeedNetworks/seed/issues/1625)) ([1f2c242](https://github.com/MustardSeedNetworks/seed/commit/1f2c242a6af96780696974bf9dd2e212fff5265b))
* **api:** strangle health-checks settings into a use-case (ADR-0020, WS-A4) ([#1658](https://github.com/MustardSeedNetworks/seed/issues/1658)) ([6967217](https://github.com/MustardSeedNetworks/seed/commit/696721726d007f55b6a596b86e7f2ecfbc581a4e))
* **api:** strangle identity (users/oauth/tokens) into use-cases over repository ports (C4) ([#1624](https://github.com/MustardSeedNetworks/seed/issues/1624)) ([8c335f8](https://github.com/MustardSeedNetworks/seed/commit/8c335f82a8396e0ed5242f873a8a37abc6d0db58))
* **api:** strangle main settings into a use-case (ADR-0020, WS-A2) ([#1656](https://github.com/MustardSeedNetworks/seed/issues/1656)) ([c9f16ce](https://github.com/MustardSeedNetworks/seed/commit/c9f16cefacc7415254b49c5c3894e3a0b8ce950e))
* **api:** strangle polling-targets CRUD into a use-case (ADR-0020, WS-A7) ([#1661](https://github.com/MustardSeedNetworks/seed/issues/1661)) ([3640352](https://github.com/MustardSeedNetworks/seed/commit/364035220a43b59357a5716e12bf514aae3f9a6a))
* **api:** strangle security settings into a use-case + fix SNMP deadlock (ADR-0020, WS-A3) ([#1657](https://github.com/MustardSeedNetworks/seed/issues/1657)) ([7146550](https://github.com/MustardSeedNetworks/seed/commit/714655090163ec6fe53fd43e9c97cc94ff479929))
* **api:** strangle the last residual handler config/db reaches (ADR-0020, WS-A) ([#1671](https://github.com/MustardSeedNetworks/seed/issues/1671)) ([ba8d183](https://github.com/MustardSeedNetworks/seed/commit/ba8d183ba3d0adcdae92314900526dc763c4d39e))
* **api:** strangle the log query into a use-case (ADR-0020, WS-A11b) ([#1668](https://github.com/MustardSeedNetworks/seed/issues/1668)) ([9e22a05](https://github.com/MustardSeedNetworks/seed/commit/9e22a0516eea99a31ed116b895b4824f9f3f7f18))
* **api:** strangle topology read endpoints into a use-case (ADR-0020, WS-A6) ([#1660](https://github.com/MustardSeedNetworks/seed/issues/1660)) ([b306028](https://github.com/MustardSeedNetworks/seed/commit/b30602829e09df79d561e39fedb0a3dffd7abade))
* **api:** strangle update handlers into internal/update/lifecycle (C3) ([#1620](https://github.com/MustardSeedNetworks/seed/issues/1620)) ([decba0c](https://github.com/MustardSeedNetworks/seed/commit/decba0c78f65dc2522fcfe6e5148cefe386304fa))
* **api:** strangle vulnerability-scanner settings into the security use-case (ADR-0020, WS-A5) ([#1659](https://github.com/MustardSeedNetworks/seed/issues/1659)) ([1d0d33e](https://github.com/MustardSeedNetworks/seed/commit/1d0d33edf428950482953cec542b4bb9b7c7299f))
* **ci:** add domain-purity depguard rules for six capability packages (WS-C4) ([#1697](https://github.com/MustardSeedNetworks/seed/issues/1697)) ([134e9eb](https://github.com/MustardSeedNetworks/seed/commit/134e9ebd57c7fb54c09081c2fd0bed7a774deb4d))
* **ci:** empty the json-casing baseline; exempt external adapters by marker (ADR-0010 revised) ([#1684](https://github.com/MustardSeedNetworks/seed/issues/1684)) ([1df12dd](https://github.com/MustardSeedNetworks/seed/commit/1df12dd6c5a80fd8a9ade6b746aea8aabbe0be66))
* **config:** reject plaintext credentials, delete legacy v0/JWT path ([#1623](https://github.com/MustardSeedNetworks/seed/issues/1623)) ([05350bf](https://github.com/MustardSeedNetworks/seed/commit/05350bfbece01b143d7f7fde135fa52249d6bc6e))
* **database:** decompose the discovery repository god-file by entity (WS-D) ([#1702](https://github.com/MustardSeedNetworks/seed/issues/1702)) ([57e1fbe](https://github.com/MustardSeedNetworks/seed/commit/57e1fbe40c031584e0d32cac9eb6c381416aa473))
* decompose four WS-D god-files by role (arp, users, snmp, engine) ([#1709](https://github.com/MustardSeedNetworks/seed/issues/1709)) ([cb8e937](https://github.com/MustardSeedNetworks/seed/commit/cb8e9375184d9e8881a1bb8465abcd020d19b12f))
* decompose seven more WS-D god-files by role (database, alerts, registry, netif, iperf installer) ([#1710](https://github.com/MustardSeedNetworks/seed/issues/1710)) ([ec72335](https://github.com/MustardSeedNetworks/seed/commit/ec723354cd0271bb704730784cce1556aa1c7ba3))
* **dhcp:** split lease-file parsing out of the dhcp god-file (WS-D) ([#1700](https://github.com/MustardSeedNetworks/seed/issues/1700)) ([d4c7680](https://github.com/MustardSeedNetworks/seed/commit/d4c76808c3e747ea8e94cc1a0726211d1acbe15e))
* **discovery:** decompose the fingerprint god-file by role (WS-D) ([#1708](https://github.com/MustardSeedNetworks/seed/issues/1708)) ([b90b0e2](https://github.com/MustardSeedNetworks/seed/commit/b90b0e26bed21434497c3edaea1a6bb77edef93e))
* **discovery:** decompose the traceroute god-file by protocol (WS-D) ([#1705](https://github.com/MustardSeedNetworks/seed/issues/1705)) ([05fccd7](https://github.com/MustardSeedNetworks/seed/commit/05fccd76ef4b15dcef67b17fe1bb4d481f0b5b9f))
* **health:** delete the dead health_check_results read-path (ADR-0026) ([#1636](https://github.com/MustardSeedNetworks/seed/issues/1636)) ([ded0f2b](https://github.com/MustardSeedNetworks/seed/commit/ded0f2bd43ee077374a9bb5322a4f4413782bf37))
* **health:** make health surface persistence-free (WS-B) ([#1678](https://github.com/MustardSeedNetworks/seed/issues/1678)) ([3215b97](https://github.com/MustardSeedNetworks/seed/commit/3215b97f13640d680083445865add9d5b27fcd16))
* **iperf:** decompose the iperf god-file by role (WS-D) ([#1706](https://github.com/MustardSeedNetworks/seed/issues/1706)) ([62db623](https://github.com/MustardSeedNetworks/seed/commit/62db62346e008343e74d02b7849c1c82013b1aa8))
* **paths:** drop legacy config-name fallback and dead DetectLegacyConfig ([#1609](https://github.com/MustardSeedNetworks/seed/issues/1609)) ([7b0c6f9](https://github.com/MustardSeedNetworks/seed/commit/7b0c6f926416c2dd9ee054497d6ee64785b0c609))
* **polling:** relocate PollingTarget to domain pkg + orchestrator ports, make persistence-free (WS-B) ([#1675](https://github.com/MustardSeedNetworks/seed/issues/1675)) ([74c4162](https://github.com/MustardSeedNetworks/seed/commit/74c4162e637d459ad2be551b13e49e0489796b2c))
* **probe:** narrow persistence port to domain types (WS-B1) ([#1672](https://github.com/MustardSeedNetworks/seed/issues/1672)) ([e07d928](https://github.com/MustardSeedNetworks/seed/commit/e07d9285c75763bdd12bc59b3c59f10a06b6a4bb))
* **retention:** relocate rollup SQL adapters to internal/database (WS-B5) ([#1677](https://github.com/MustardSeedNetworks/seed/issues/1677)) ([7370c51](https://github.com/MustardSeedNetworks/seed/commit/7370c515dfd01046fd9c666a4a9a2c73c1edca34))
* **settings:** decompose the management god-file by role (WS-D) ([#1699](https://github.com/MustardSeedNetworks/seed/issues/1699)) ([54b3399](https://github.com/MustardSeedNetworks/seed/commit/54b33993b3a6553c525855a26abce9fe2474cd78))
* **snmp:** decompose the interface god-file by query area (WS-D) ([#1701](https://github.com/MustardSeedNetworks/seed/issues/1701)) ([cdbbc18](https://github.com/MustardSeedNetworks/seed/commit/cdbbc1828be47086d87f569761a7397402ec273d))
* **survey:** decompose the report god-file by role (WS-D) ([#1703](https://github.com/MustardSeedNetworks/seed/issues/1703)) ([b19db25](https://github.com/MustardSeedNetworks/seed/commit/b19db25c085437971690df31a103daa5f1869926))
* **survey:** decompose the survey manager god-file by role (WS-D) ([#1704](https://github.com/MustardSeedNetworks/seed/issues/1704)) ([c06374a](https://github.com/MustardSeedNetworks/seed/commit/c06374a2641f881582e3245220dba2df66b9aaf6))
* **topology:** relocate row types to domain pkgs, make persistence-free (WS-B) ([#1673](https://github.com/MustardSeedNetworks/seed/issues/1673)) ([e768fcc](https://github.com/MustardSeedNetworks/seed/commit/e768fcc7b884fd387b87faf46a799ca4740f7256))
* **ui:** function-first sidebar nav IA, retire botanical metaphor (M1, [#1452](https://github.com/MustardSeedNetworks/seed/issues/1452)) ([#1681](https://github.com/MustardSeedNetworks/seed/issues/1681)) ([546d937](https://github.com/MustardSeedNetworks/seed/commit/546d93724dd5f26a7ba984b32a0923bc61fe1ff2))
* **vuln:** decompose the scanner god-file by role (WS-D) ([#1707](https://github.com/MustardSeedNetworks/seed/issues/1707)) ([2701fef](https://github.com/MustardSeedNetworks/seed/commit/2701fefb81f63ea051e664f0688dda0dc5ecc2f9))


### Documentation

* **adr-0010:** revise to pure boundary mapping — wire is 100% camelCase, no exceptions ([#1683](https://github.com/MustardSeedNetworks/seed/issues/1683)) ([0259e3d](https://github.com/MustardSeedNetworks/seed/commit/0259e3d39c94e60887446307f33487e02fc6cde1))
* **adr:** ADR-0021 persist anomaly engine in SQL + converge all sources ([#1619](https://github.com/MustardSeedNetworks/seed/issues/1619)) ([d2d367a](https://github.com/MustardSeedNetworks/seed/commit/d2d367a67f0e95302d593ad65e0da8b4c3f7628f))
* **adr:** design daily rollups for the anomaly store (ADR-0028) ([#1649](https://github.com/MustardSeedNetworks/seed/issues/1649)) ([1150b81](https://github.com/MustardSeedNetworks/seed/commit/1150b8144ff471aa7534648dbf8172c3ba5baea6))
* **adr:** hygiene pass — honest statuses + two as-built ADRs ([#1622](https://github.com/MustardSeedNetworks/seed/issues/1622)) ([db669f7](https://github.com/MustardSeedNetworks/seed/commit/db669f7615fe7082a8d2b9de875475178d8ee492))
* **adr:** probe is the active-monitoring anomaly source (ADR-0025) ([#1634](https://github.com/MustardSeedNetworks/seed/issues/1634)) ([58370b8](https://github.com/MustardSeedNetworks/seed/commit/58370b8128df232344cd26beb435f06645160c9e))
* **adr:** scope migrating health-checks onto probe, then renaming (ADR-0027) ([#1640](https://github.com/MustardSeedNetworks/seed/issues/1640)) ([a7da0da](https://github.com/MustardSeedNetworks/seed/commit/a7da0da7a4522dd7b37e2f18c41a754b3c57026b))
* **architecture:** add the architecture completion plan (no-shortcuts strangle finish) ([#1654](https://github.com/MustardSeedNetworks/seed/issues/1654)) ([4b189f1](https://github.com/MustardSeedNetworks/seed/commit/4b189f12644392cad33446086d1854d55eed326a))
* **ws-b:** close B3 — identity is the documented ADR-0024 exception ([#1679](https://github.com/MustardSeedNetworks/seed/issues/1679)) ([2bd9ab6](https://github.com/MustardSeedNetworks/seed/commit/2bd9ab6ccde0436c13ec38c172e5f3525723cb9f))


### Continuous Integration

* **governance:** exempt release-please PRs from human PR-body template ([#1741](https://github.com/MustardSeedNetworks/seed/issues/1741)) ([e083074](https://github.com/MustardSeedNetworks/seed/commit/e083074a5012085b678f451f8fa5a8c55dfbf5d0))
* **perf:** build UI once, add job timeouts, least-privilege ([#1733](https://github.com/MustardSeedNetworks/seed/issues/1733)) ([02e2a17](https://github.com/MustardSeedNetworks/seed/commit/02e2a17bc8015357b8a61ad16e8242ca17c8526f))
* **release:** use msn-ci-bot App token for release-please ([#1745](https://github.com/MustardSeedNetworks/seed/issues/1745)) ([515b298](https://github.com/MustardSeedNetworks/seed/commit/515b298f7c21e49b850f617576d15f805d38bc9f))
* **security:** add Semgrep SAST gate, pin curl|sh install ([#1737](https://github.com/MustardSeedNetworks/seed/issues/1737)) ([ecbc6ec](https://github.com/MustardSeedNetworks/seed/commit/ecbc6ec12d383d3e34fe9408f7d387d90308af9b)), closes [#1736](https://github.com/MustardSeedNetworks/seed/issues/1736)


### Miscellaneous

* **build:** remove Docker/GHCR publishing ([#1729](https://github.com/MustardSeedNetworks/seed/issues/1729)) ([513e1e0](https://github.com/MustardSeedNetworks/seed/commit/513e1e01d4291e13fa4e32231371644a26778e2e))
* **ci:** add filename-policy gate for decomposed packages ([#1617](https://github.com/MustardSeedNetworks/seed/issues/1617)) ([cd8642c](https://github.com/MustardSeedNetworks/seed/commit/cd8642ccea29b2324573d43447f2a2a2d8b8772b))
* **cleanup:** remove dead HTTP-redirector remnants ([#1621](https://github.com/MustardSeedNetworks/seed/issues/1621)) ([1fc1275](https://github.com/MustardSeedNetworks/seed/commit/1fc12751a4c548be59b7f21bc20708f31f95f22b))
* **deps:** always-latest toolchain sweep (lockstep with stem) ([#1687](https://github.com/MustardSeedNetworks/seed/issues/1687)) ([60b54c4](https://github.com/MustardSeedNetworks/seed/commit/60b54c415ab2a120a5c39a98b0a2838859086868))
* **deps:** refresh go module graph ([#1690](https://github.com/MustardSeedNetworks/seed/issues/1690)) ([82144b5](https://github.com/MustardSeedNetworks/seed/commit/82144b5e5c5c479289df6b759aaedb3119509b06))
* **deps:** update frontend deps to latest + clear esbuild advisory ([#1666](https://github.com/MustardSeedNetworks/seed/issues/1666)) ([9d669bf](https://github.com/MustardSeedNetworks/seed/commit/9d669bfbc22b448c26986835c8f4e05f29ea2408))
* **deps:** update Go modules to latest ([#1667](https://github.com/MustardSeedNetworks/seed/issues/1667)) ([2153916](https://github.com/MustardSeedNetworks/seed/commit/21539163bc5ea64f432991a9b2c2b0d3b96f3477))
* **github:** standardize repo governance ([#1689](https://github.com/MustardSeedNetworks/seed/issues/1689)) ([f409599](https://github.com/MustardSeedNetworks/seed/commit/f40959915e63522e426b5c60236794122085977e))
* **license:** add license-key-circumvention clause to BUSL Additional Use Grant ([#1735](https://github.com/MustardSeedNetworks/seed/issues/1735)) ([e961114](https://github.com/MustardSeedNetworks/seed/commit/e96111461834308534e4712e5a5b4dd1586e9d23))
* **main:** release 0.211.0 ([#1698](https://github.com/MustardSeedNetworks/seed/issues/1698)) ([db007d9](https://github.com/MustardSeedNetworks/seed/commit/db007d91703bdb4ca3d92eb9ee5d64bda72154cd))
* **main:** release 0.212.0 ([#1610](https://github.com/MustardSeedNetworks/seed/issues/1610)) ([8124e7a](https://github.com/MustardSeedNetworks/seed/commit/8124e7a537eb0acb7c2ccddd18bd44a19efba990))
* **main:** release 0.212.1 ([#1674](https://github.com/MustardSeedNetworks/seed/issues/1674)) ([33e4388](https://github.com/MustardSeedNetworks/seed/commit/33e4388b4faa927b72e176803f0271576c9fd848))
* **release:** expose release train metadata ([#1693](https://github.com/MustardSeedNetworks/seed/issues/1693)) ([58c79fc](https://github.com/MustardSeedNetworks/seed/commit/58c79fc26b96d19c213afc4773dcf54e65d7212e))
* **ui:** replace eslint references with biome ([#1695](https://github.com/MustardSeedNetworks/seed/issues/1695)) ([139b5d5](https://github.com/MustardSeedNetworks/seed/commit/139b5d5a552aeda50cc6ec4721736ef7f835d297))

## [0.211.0](https://github.com/MustardSeedNetworks/seed/compare/v0.212.1...v0.211.0) (2026-07-06)


### ⚠ BREAKING CHANGES

* **probe:** the health-check run/settings/anomalies endpoints moved from /telemetry/health-checks/* to /telemetry/probes/*. Pre-alpha, no compat.
* **probe:** /run response drops the unrendered CustomTestResult fields (security headers, redirect chains, body-match, per-endpoint vertical detail that was never populated). Pre-alpha, no compat.
* **health:** delete the dead health_check_results read-path (ADR-0026) ([#1636](https://github.com/MustardSeedNetworks/seed/issues/1636))
* the SSO config API (/api/v1 sso update) and the CVE-database file format now use camelCase keys (clientId, clientSecret, redirectUrl, tenantId, cvssScore, lastUpdated) instead of snake_case. Pre-1.0, no external consumers.
* **api:** the MFA (/api/v1 mfa) and auth-login responses now use camelCase json keys (totpEnabled, provisioningUri, qrCodePngBase64, mfaRequired, mfaToken, etc.) instead of snake_case. Pre-1.0, no external consumers; the bundled UI is updated in lockstep.
* **api:** the logs (/api logs query + stats), profiles (/api profiles), and config-version/restore APIs now use camelCase json keys (requestId, durationMs, totalCount, byLevel, isDefault, needsMigration, backupName, etc.) instead of snake_case. Pre-1.0, no external consumers; the bundled UI is updated in lockstep.
* **discovery:** the network problem-detection API (/api problem + network-problems responses) now uses camelCase json keys (deviceId, bySeverity, signalDbm, interfaceErrors, scanDurationMs, etc.) instead of snake_case. Pre-1.0, no external consumers; the bundled UI is updated in lockstep via regenerated types.
* **discovery:** the wi-fi discovery API (/api/v1/wifi discovery responses) now uses camelCase json keys (isHidden, securityType, frequencyMhz, signalDbm, etc.) instead of snake_case. Pre-1.0, no external consumers; the bundled UI is updated in lockstep via regenerated types.

### Features

* **a11y:** axe-core test harness + DiscoveryModal clear-button label ([#1272](https://github.com/MustardSeedNetworks/seed/issues/1272)) ([8d8913d](https://github.com/MustardSeedNetworks/seed/commit/8d8913d46f15c37f0964c030f33cadc84007bb8d))
* **alerts:** listener-event alert pipeline (stage a4.5) ([#1354](https://github.com/MustardSeedNetworks/seed/issues/1354)) ([5c906ac](https://github.com/MustardSeedNetworks/seed/commit/5c906acca94287a3fb5cd226aec221c8b7db916b))
* **alerts:** load listener rules from db with hot reload ([#1386](https://github.com/MustardSeedNetworks/seed/issues/1386)) ([2b0082c](https://github.com/MustardSeedNetworks/seed/commit/2b0082c4691e324f6ea71a33a8490d8d947c6e3a))
* **alerts:** observation-delta alert pipeline (stage a4.6) ([#1355](https://github.com/MustardSeedNetworks/seed/issues/1355)) ([cea75e1](https://github.com/MustardSeedNetworks/seed/commit/cea75e1de817097006b864eaa592a6a32e68f897))
* **alerts:** replay observation state on startup ([#1399](https://github.com/MustardSeedNetworks/seed/issues/1399)) ([06600ee](https://github.com/MustardSeedNetworks/seed/commit/06600eec96ce89966942c815666b7e19ff1e2431))
* **alerts:** time-windowed rule thresholds ([#1398](https://github.com/MustardSeedNetworks/seed/issues/1398)) ([ee2a991](https://github.com/MustardSeedNetworks/seed/commit/ee2a991abe77bcb0ef901c72ccfcbb4e57ec1031))
* **anomaly:** add error severity to the four-level ladder (ADR-0021 ph5) ([#1637](https://github.com/MustardSeedNetworks/seed/issues/1637)) ([5127ca0](https://github.com/MustardSeedNetworks/seed/commit/5127ca0fe53a59bd36e1b86aab2f2a544f872bc1))
* **anomaly:** daily-rollup census of the anomaly store (ADR-0028) ([#1650](https://github.com/MustardSeedNetworks/seed/issues/1650)) ([97aab3e](https://github.com/MustardSeedNetworks/seed/commit/97aab3ec2598a3ceb9d8976746dcf7483b058041))
* **anomaly:** general network-anomaly engine + data-driven catalog (W4a) ([#1531](https://github.com/MustardSeedNetworks/seed/issues/1531)) ([37ebd31](https://github.com/MustardSeedNetworks/seed/commit/37ebd31ce9bfb45ab34fc62df1aa46cd3193473b))
* **anomaly:** persist the anomaly engine in SQL (ADR-0021 phase 1) ([#1629](https://github.com/MustardSeedNetworks/seed/issues/1629)) ([83f2ce4](https://github.com/MustardSeedNetworks/seed/commit/83f2ce46eeebf55335c3de548117dc0ca7e71527))
* **anomaly:** persist the Wi-Fi anomaly stream + load-on-start (ADR-0021 phase 3) ([#1632](https://github.com/MustardSeedNetworks/seed/issues/1632)) ([d24f896](https://github.com/MustardSeedNetworks/seed/commit/d24f896670a1c62cd4d19e1522cf3e14af3f9978))
* **anomaly:** probe is the active-monitoring anomaly producer (ADR-0025) ([#1635](https://github.com/MustardSeedNetworks/seed/issues/1635)) ([500c890](https://github.com/MustardSeedNetworks/seed/commit/500c8902a7ecfeb51ea49a71e8b8b8665d9507d7))
* **anomaly:** re-derive catalog-static impact/followUps on store read (ADR-0029) ([#1653](https://github.com/MustardSeedNetworks/seed/issues/1653)) ([be81b0d](https://github.com/MustardSeedNetworks/seed/commit/be81b0db76e31a7fccc749b0cc9b58a0f5bf3e56))
* **anomaly:** resolve probe anomalies immediately on a clean result ([#1638](https://github.com/MustardSeedNetworks/seed/issues/1638)) ([1365e9a](https://github.com/MustardSeedNetworks/seed/commit/1365e9a1b154a45c59ea1f9036e2fdf7af115957))
* **anomaly:** TTL-purge resolved anomalies in retention (ADR-0021 phase 2) ([#1630](https://github.com/MustardSeedNetworks/seed/issues/1630)) ([0ca47d7](https://github.com/MustardSeedNetworks/seed/commit/0ca47d7566e978a421b7189b3341a458dfea089e))
* **api:** /api/v1/engines admin endpoint (stage a5.8, item 3) ([#1364](https://github.com/MustardSeedNetworks/seed/issues/1364)) ([ef4fa05](https://github.com/MustardSeedNetworks/seed/commit/ef4fa05f9f191db98659fbede66c225bf03db0a8))
* **api:** add unified job runner HTTP surface (ADR-0005) ([#1468](https://github.com/MustardSeedNetworks/seed/issues/1468)) ([5c36baf](https://github.com/MustardSeedNetworks/seed/commit/5c36baf1f8bc7359b5551843a5ec79400d409ca6))
* **api:** alert-rule editor (stage a5.10, item 5) ([#1366](https://github.com/MustardSeedNetworks/seed/issues/1366)) ([871414c](https://github.com/MustardSeedNetworks/seed/commit/871414c866de7c626ec906c1af2922bf9b3a8722))
* **api:** alerts list + acknowledge/resolve endpoints (stage a5.2) ([#1358](https://github.com/MustardSeedNetworks/seed/issues/1358)) ([b66dc9b](https://github.com/MustardSeedNetworks/seed/commit/b66dc9b6f076192779c1ee772da22efae7a6a503))
* **api:** capability manifest + /__capabilities + route-policy CI gate ([#1412](https://github.com/MustardSeedNetworks/seed/issues/1412)) ([80202b2](https://github.com/MustardSeedNetworks/seed/commit/80202b23a26262c4fe508c03a9f65ff48ef896a0))
* **api:** capability registry + convert Canopy routes (Phase 1) ([#1406](https://github.com/MustardSeedNetworks/seed/issues/1406)) ([7b5dfc9](https://github.com/MustardSeedNetworks/seed/commit/7b5dfc990e223a785c9c2b46c141cfbb94f05ef0))
* **api:** convert Roots/Harvest/Topology/API-token routes to registry ([#1407](https://github.com/MustardSeedNetworks/seed/issues/1407)) ([d9a0dda](https://github.com/MustardSeedNetworks/seed/commit/d9a0dda9048470955af04021c039c656caedff60))
* **api:** convert SAP + Shell routes to registry ([#1410](https://github.com/MustardSeedNetworks/seed/issues/1410)) ([c4cbd36](https://github.com/MustardSeedNetworks/seed/commit/c4cbd36fee35b4b9c47baecdcc8901c1f2cbfd52))
* **api:** convert Update routes to registry ([#1411](https://github.com/MustardSeedNetworks/seed/issues/1411)) ([c2d89a7](https://github.com/MustardSeedNetworks/seed/commit/c2d89a77bae589c1ee79b308ffcc4d22a596d188))
* **api:** enforce viewer read-only via writeGated route wrapper ([#1226](https://github.com/MustardSeedNetworks/seed/issues/1226)) ([#1265](https://github.com/MustardSeedNetworks/seed/issues/1265)) ([b99dae0](https://github.com/MustardSeedNetworks/seed/commit/b99dae0409e68379185b3a9fddd16d49c661c041))
* **api:** expose IncludeNameRes + IncludeProfiling on EngineScanRequest (P7 S4.1c) ([#1502](https://github.com/MustardSeedNetworks/seed/issues/1502)) ([dd7cc62](https://github.com/MustardSeedNetworks/seed/commit/dd7cc624af7f9d3c157d26cab12248936501cb81))
* **api:** gate /events SSE on live_telemetry — close Pro revenue leak ([#1278](https://github.com/MustardSeedNetworks/seed/issues/1278)) ([f08b8ef](https://github.com/MustardSeedNetworks/seed/commit/f08b8ef17b999c88af341ccd5d382d1d1f5a09aa))
* **api:** gate wifi_roam_analysis on survey response — close revenue leak ([#1280](https://github.com/MustardSeedNetworks/seed/issues/1280)) ([bdec2ff](https://github.com/MustardSeedNetworks/seed/commit/bdec2ff922725d83ca737e97a876415a87ebe016))
* **api:** list arp bindings via /topology/arp ([#1387](https://github.com/MustardSeedNetworks/seed/issues/1387)) ([9840f66](https://github.com/MustardSeedNetworks/seed/commit/9840f66448cc33fff56e4a95b5dd1e157b3398eb)), closes [#1382](https://github.com/MustardSeedNetworks/seed/issues/1382) [#1367](https://github.com/MustardSeedNetworks/seed/issues/1367)
* **api:** make HTTP method + body-limit authoritative in the route registry ([#1530](https://github.com/MustardSeedNetworks/seed/issues/1530)) ([8314d24](https://github.com/MustardSeedNetworks/seed/commit/8314d246c114096fc32d16e5ed42d0e139f3c28e))
* **api:** migrate bluetooth/wifi-discovery/device scans to job kinds (ADR-0005) ([#1474](https://github.com/MustardSeedNetworks/seed/issues/1474)) ([f28a890](https://github.com/MustardSeedNetworks/seed/commit/f28a890ac48d3a8ed331a0de2cc3299487cba8db))
* **api:** migrate discovery engine scan to a job kind (ADR-0005) ([#1473](https://github.com/MustardSeedNetworks/seed/issues/1473)) ([857eb64](https://github.com/MustardSeedNetworks/seed/commit/857eb645018e4bed9d68898bdcac9042a44fc83f))
* **api:** migrate iperf3 client test to a job kind (ADR-0005) ([#1471](https://github.com/MustardSeedNetworks/seed/issues/1471)) ([0fc9f2a](https://github.com/MustardSeedNetworks/seed/commit/0fc9f2a2023e6b0dd4ce7542e76556823eb62db0))
* **api:** migrate speedtest to a unified job kind (ADR-0005) ([#1470](https://github.com/MustardSeedNetworks/seed/issues/1470)) ([5a447f3](https://github.com/MustardSeedNetworks/seed/commit/5a447f312408c2737cbda5887f188c53260193e6))
* **api:** migrate vulnerability scan to a job kind (ADR-0005) ([#1472](https://github.com/MustardSeedNetworks/seed/issues/1472)) ([f937e86](https://github.com/MustardSeedNetworks/seed/commit/f937e8608487942e924fd2fc485a85d55662183d))
* **api:** per-tier engine gating (stage a5.9, item 4) ([#1365](https://github.com/MustardSeedNetworks/seed/issues/1365)) ([8f628dc](https://github.com/MustardSeedNetworks/seed/commit/8f628dce708278cd7d0f72a8cb7b73e42c9994cf))
* **api:** per-token role scope for personal-access tokens ([#1255](https://github.com/MustardSeedNetworks/seed/issues/1255)) ([#1268](https://github.com/MustardSeedNetworks/seed/issues/1268)) ([a5078dd](https://github.com/MustardSeedNetworks/seed/commit/a5078dd8ec126d3163225fe8f08becd231be72e4))
* **api:** polling targets crud (stage a5.3) ([#1359](https://github.com/MustardSeedNetworks/seed/issues/1359)) ([fb279f0](https://github.com/MustardSeedNetworks/seed/commit/fb279f0874b68310fe3a490d1b8c4313b2d80a22))
* **api:** read-only topology endpoints (stage a5.1) ([#1357](https://github.com/MustardSeedNetworks/seed/issues/1357)) ([22f5169](https://github.com/MustardSeedNetworks/seed/commit/22f5169e10e1baff003bcdce2ca46905f58cb499))
* **api:** register stage a4 engines via services.engines (wire-up) ([#1356](https://github.com/MustardSeedNetworks/seed/issues/1356)) ([2df506e](https://github.com/MustardSeedNetworks/seed/commit/2df506ed9b310232e95ed0997601ae9117dfba76))
* **api:** server lifecycle via engine.registry (stage a3.5d) ([#1345](https://github.com/MustardSeedNetworks/seed/issues/1345)) ([8930e5b](https://github.com/MustardSeedNetworks/seed/commit/8930e5b32b6a7443a5a22098ae006effeac75fed))
* **api:** structured audit log for authz denials ([#1257](https://github.com/MustardSeedNetworks/seed/issues/1257)) ([#1271](https://github.com/MustardSeedNetworks/seed/issues/1271)) ([47bdac6](https://github.com/MustardSeedNetworks/seed/commit/47bdac6ff46e188ee60c748b682a5a26d379a737))
* **api:** wire probe engine into server lifecycle (Stage A1.8) ([#1326](https://github.com/MustardSeedNetworks/seed/issues/1326)) ([cbdacac](https://github.com/MustardSeedNetworks/seed/commit/cbdacacc3fd7fb52a48543980b53226e0484fd3d))
* **api:** wire snmp poller into services.engines (stage a5.4) ([#1360](https://github.com/MustardSeedNetworks/seed/issues/1360)) ([58b52b1](https://github.com/MustardSeedNetworks/seed/commit/58b52b1aeb6bb25dd785917a8ffbbfaa7a58c2c8))
* **api:** wire syslog + snmp trap listeners via engine.registry (stage a3.5e-4) ([#1349](https://github.com/MustardSeedNetworks/seed/issues/1349)) ([d34f2fd](https://github.com/MustardSeedNetworks/seed/commit/d34f2fdb5e9d43b2d351518c9fb830dd5786ac26))
* **bluetooth:** Bluetooth visibility UI (card + full-screen device modal) ([#1520](https://github.com/MustardSeedNetworks/seed/issues/1520)) ([8177448](https://github.com/MustardSeedNetworks/seed/commit/8177448148bc5b3528da09da1663c5055288fe4b))
* **bluetooth:** decode manufacturer ID, GATT services, and BLE appearance ([#1517](https://github.com/MustardSeedNetworks/seed/issues/1517)) ([67b2168](https://github.com/MustardSeedNetworks/seed/commit/67b21682ee6d0757086d2dbe16be67b9947c7781))
* **cli:** example blocks on all commands + help completeness test ([#1273](https://github.com/MustardSeedNetworks/seed/issues/1273)) ([8cadbb1](https://github.com/MustardSeedNetworks/seed/commit/8cadbb109307a2e41755efa591695b8e9285472e))
* **config:** refuse CORS `*` origin at startup ([#1256](https://github.com/MustardSeedNetworks/seed/issues/1256)) ([#1269](https://github.com/MustardSeedNetworks/seed/issues/1269)) ([32cd690](https://github.com/MustardSeedNetworks/seed/commit/32cd690973cb4009f8e7bfbf22b554846d27b6df))
* **config:** separate credential encryption key from JWTSecret (ADR-0015) ([#1549](https://github.com/MustardSeedNetworks/seed/issues/1549)) ([69e19c5](https://github.com/MustardSeedNetworks/seed/commit/69e19c549fbde40559939e2c8b43f58bb781ce18))
* **contract:** code-first contract decision (ADR-0003 amended) + widen DTO coverage ([#1413](https://github.com/MustardSeedNetworks/seed/issues/1413)) ([d8d70ba](https://github.com/MustardSeedNetworks/seed/commit/d8d70ba5785cde926a8c833bc37917497f7aa33e))
* **contract:** widen DTO coverage batch 2 (7 DTOs); flag nested-defs blocker ([#1414](https://github.com/MustardSeedNetworks/seed/issues/1414)) ([35111cb](https://github.com/MustardSeedNetworks/seed/commit/35111cb1678c6eb750c457478ce8c55583924a61))
* **contract:** widen DTO coverage batch 3 (+9 SAP/network/discovery) ([#1416](https://github.com/MustardSeedNetworks/seed/issues/1416)) ([902eb32](https://github.com/MustardSeedNetworks/seed/commit/902eb32b1d2a888054bda65cbea6282d9bcec268))
* **contract:** widen DTO coverage batch 4 (+10 SAP/network settings) ([#1418](https://github.com/MustardSeedNetworks/seed/issues/1418)) ([140beba](https://github.com/MustardSeedNetworks/seed/commit/140beba6e85c50804e9595f2ba040b43637b20be))
* **contract:** widen DTO coverage batch 5 (+22 health-check DTOs) ([#1419](https://github.com/MustardSeedNetworks/seed/issues/1419)) ([b786040](https://github.com/MustardSeedNetworks/seed/commit/b786040e20f2f4e2a1e75e76f591c2cac2e7ebed))
* **contract:** widen DTO coverage batch 6 (+11 iperf/tools/dns/engine) ([#1420](https://github.com/MustardSeedNetworks/seed/issues/1420)) ([40beeda](https://github.com/MustardSeedNetworks/seed/commit/40beeda8271986723fcaa6d9970c746d50a1067e))
* **contract:** widen DTO coverage batch 7 (+16 users/tokens/update/sso/logs) ([#1421](https://github.com/MustardSeedNetworks/seed/issues/1421)) ([b31b585](https://github.com/MustardSeedNetworks/seed/commit/b31b58572ba90c5bff35a5fa6ee5e1cb7f69c62a))
* **contract:** widen DTO coverage batch 8 (+10 survey) — self-contained set complete ([#1422](https://github.com/MustardSeedNetworks/seed/issues/1422)) ([f50d8f6](https://github.com/MustardSeedNetworks/seed/commit/f50d8f6919b6db406c84a0fbf6bac747320639c0))
* **database:** add proven goose schema baseline (ADR-0006, Phase 5b-1) ([#1477](https://github.com/MustardSeedNetworks/seed/issues/1477)) ([f2d41b0](https://github.com/MustardSeedNetworks/seed/commit/f2d41b0aac898f4ef6e3a508efe97f587b9e700f))
* **database:** durable jobs table + repository (ADR-0005, Phase 5c-1) ([#1481](https://github.com/MustardSeedNetworks/seed/issues/1481)) ([86e71b3](https://github.com/MustardSeedNetworks/seed/commit/86e71b3e82699976c1f2ac217e2c07e3b4860f26))
* **database:** swap migration runner to goose (ADR-0006, Phase 5b-2) ([#1478](https://github.com/MustardSeedNetworks/seed/issues/1478)) ([21ee962](https://github.com/MustardSeedNetworks/seed/commit/21ee9625a04f16cb7cfc6cff1c7a2e77cf4d3ea2))
* **db:** drop superseded dns_monitors / ssl_monitors / cert_observations (Stage A1.9) ([#1327](https://github.com/MustardSeedNetworks/seed/issues/1327)) ([0fd3e11](https://github.com/MustardSeedNetworks/seed/commit/0fd3e11c6249be7bf428d84e5b4cdaa08fd0c9f4))
* **discovery:** embed ieee oui registry as single source, drop hardcoded maps ([#1591](https://github.com/MustardSeedNetworks/seed/issues/1591)) ([18cc72b](https://github.com/MustardSeedNetworks/seed/commit/18cc72b7b54adfa27b523d4b4d584a4d2d05118c))
* **discovery:** emit phase-grained scan progress from the engine (P7 S4.2) ([#1501](https://github.com/MustardSeedNetworks/seed/issues/1501)) ([95254e7](https://github.com/MustardSeedNetworks/seed/commit/95254e73711e34e82091343af2084444c027dc12))
* **discovery:** fold port-scan intensity + timing into the engine (P7 S4.1) ([#1500](https://github.com/MustardSeedNetworks/seed/issues/1500)) ([14839df](https://github.com/MustardSeedNetworks/seed/commit/14839df85faa606f94270dfb67c0c2b4c186ad86))
* **engine:** minimal engine interface + lifecycle registry (stage a3.5a) ([#1343](https://github.com/MustardSeedNetworks/seed/issues/1343)) ([7230f08](https://github.com/MustardSeedNetworks/seed/commit/7230f08080f57d9a6f0ffe437fa71a01b3da5005))
* **engine:** optional reporter interface + /engines status surface ([#1389](https://github.com/MustardSeedNetworks/seed/issues/1389)) ([19c117e](https://github.com/MustardSeedNetworks/seed/commit/19c117e6f5cd4fda799f8a43ba1a86735cb53705))
* **forms:** adopt react-hook-form + valibot ([#1201](https://github.com/MustardSeedNetworks/seed/issues/1201)) ([#1209](https://github.com/MustardSeedNetworks/seed/issues/1209)) ([07b6ac2](https://github.com/MustardSeedNetworks/seed/commit/07b6ac22e3e6e5b33acbd8beed2ef73af5505b94))
* **help:** add path/reports/logs sections + route-coverage test ([#1274](https://github.com/MustardSeedNetworks/seed/issues/1274)) ([5fc112f](https://github.com/MustardSeedNetworks/seed/commit/5fc112fb779f503039032bf3e4835b18e4a7605c))
* **i18n:** add errors.license.* keys for tier-gating UI ([#1160](https://github.com/MustardSeedNetworks/seed/issues/1160)) ([7392e31](https://github.com/MustardSeedNetworks/seed/commit/7392e3179538406e62ef95ce25f9cb95a7cd9e2e))
* **i18n:** add per-repo dynamic-prefixes allowlist for check-keys.py ([#1216](https://github.com/MustardSeedNetworks/seed/issues/1216)) ([991c591](https://github.com/MustardSeedNetworks/seed/commit/991c5914c2bee6eb991f83231b5aa3b1551e9eb5))
* **i18n:** add useLocale hook + migrate VulnerabilitySettings plural ([#1200](https://github.com/MustardSeedNetworks/seed/issues/1200)) ([f8ad517](https://github.com/MustardSeedNetworks/seed/commit/f8ad517ab43c6ce7c6bd582dc3cb735ec6f65eeb))
* **i18n:** en/es key parity + DNT compliance test ([#1276](https://github.com/MustardSeedNetworks/seed/issues/1276)) ([7c0dc9f](https://github.com/MustardSeedNetworks/seed/commit/7c0dc9f6fc54a81bded5c26745138099680382bb))
* **i18n:** port shared validator + check-keys + add phase 6 i18n tests ([#1203](https://github.com/MustardSeedNetworks/seed/issues/1203)) ([46379ff](https://github.com/MustardSeedNetworks/seed/commit/46379ffa7d55021d5b4eabec04557e75cd3e59fe))
* **interfaces:** settings ui for multi_interface ([#1210](https://github.com/MustardSeedNetworks/seed/issues/1210)) ([4e6de69](https://github.com/MustardSeedNetworks/seed/commit/4e6de694eb96dbfcf514346af74db12cb86c445d))
* **jobs:** add durable Store seam to the runner (ADR-0005, Phase 5c-2) ([#1482](https://github.com/MustardSeedNetworks/seed/issues/1482)) ([4b2dfdb](https://github.com/MustardSeedNetworks/seed/commit/4b2dfdb95a2435742e3880096d53771bd24e413f))
* **jobs:** durable Idempotency-Key store for POST /jobs (ADR-0005, Phase 5c-4) ([#1484](https://github.com/MustardSeedNetworks/seed/issues/1484)) ([e55f2db](https://github.com/MustardSeedNetworks/seed/commit/e55f2db8a8b18f9afd08e2cbae65fd9755b53066))
* **jobs:** wire durable SQLite store into the runner + boot recovery (ADR-0005, Phase 5c-3) ([#1483](https://github.com/MustardSeedNetworks/seed/issues/1483)) ([8ca9eb9](https://github.com/MustardSeedNetworks/seed/commit/8ca9eb9de6df5f03bae50c2a5a1ce3f252612894))
* **jobs:** wire jobs retention into the maintenance loop (ADR-0005, Phase 5c) ([#1485](https://github.com/MustardSeedNetworks/seed/issues/1485)) ([a2e0593](https://github.com/MustardSeedNetworks/seed/commit/a2e059368869e5fc439e54d60d985218dc7b03bb))
* **license:** add feature-gating framework ([#1153](https://github.com/MustardSeedNetworks/seed/issues/1153)) ([cc6a1fa](https://github.com/MustardSeedNetworks/seed/commit/cc6a1fa9298ff24c17f93e1f4252ce2da863d19a))
* **license:** gate /harvest/export and ReportsPage on export_csv_json (PR-B2) ([#1156](https://github.com/MustardSeedNetworks/seed/issues/1156)) ([a41567a](https://github.com/MustardSeedNetworks/seed/commit/a41567a5eb2c99f1bbad92eb91da86ded2880109))
* **license:** gate /sap/health-checks/anomalies on anomaly_detection (PR-B3) ([#1158](https://github.com/MustardSeedNetworks/seed/issues/1158)) ([dff5269](https://github.com/MustardSeedNetworks/seed/commit/dff5269ea815128b809b2175ebb9845cb9cccce1))
* **license:** gate AirMapper baseline-diff import behind Pro tier (PR-B1) ([#1157](https://github.com/MustardSeedNetworks/seed/issues/1157)) ([05ef48b](https://github.com/MustardSeedNetworks/seed/commit/05ef48b931d46d957231d9fb4957db80084c1c4a))
* **license:** gate path_analysis (Roots) behind Pro tier (PR-B5) ([#1155](https://github.com/MustardSeedNetworks/seed/issues/1155)) ([550f088](https://github.com/MustardSeedNetworks/seed/commit/550f0882923117fa6eae050e4b9d713b6ffacff9))
* **license:** gate shell active-scan endpoints on compliance_advanced (PR-B4) ([#1159](https://github.com/MustardSeedNetworks/seed/issues/1159)) ([e43dab1](https://github.com/MustardSeedNetworks/seed/commit/e43dab17e516f3193862fe0924f652840a5353ed))
* **license:** mirror keygen v2.2.0 — add sso + drop legacy multi_site/starter multi_interface ([#1197](https://github.com/MustardSeedNetworks/seed/issues/1197)) ([726668e](https://github.com/MustardSeedNetworks/seed/commit/726668e19ef0bb6b3ba74ad7b7a32b1f68077ee0))
* **license:** replace forgeable rotor cipher with Ed25519-signed tokens ([#1575](https://github.com/MustardSeedNetworks/seed/issues/1575)) ([bb70f10](https://github.com/MustardSeedNetworks/seed/commit/bb70f10f3d0450c72cc13cd94b6538224ec19ad7))
* **listener:** snmpv2c trap listener (stage a3.5e-2) ([#1348](https://github.com/MustardSeedNetworks/seed/issues/1348)) ([a1d6c5d](https://github.com/MustardSeedNetworks/seed/commit/a1d6c5d534adac4dd0e46652188ffb37f55e98b2))
* **listener:** syslog udp listener + listener_events persistence (stage a3.5e-1) ([#1347](https://github.com/MustardSeedNetworks/seed/issues/1347)) ([a80136d](https://github.com/MustardSeedNetworks/seed/commit/a80136db157a9a3f1120b5a1ddfe5ab932d48f91))
* **netif:** linkmonitor pool for multi_interface fan-out ([#1219](https://github.com/MustardSeedNetworks/seed/issues/1219)) ([b2df3fb](https://github.com/MustardSeedNetworks/seed/commit/b2df3fb53dfa74098a5ac936d70e45db07fe8252))
* **outbox:** transactional outbox relay for durable event delivery (ADR-0017) ([#1562](https://github.com/MustardSeedNetworks/seed/issues/1562)) ([958b6e8](https://github.com/MustardSeedNetworks/seed/commit/958b6e84f55d8cf421a2a34547051ec1fe11845c))
* **path:** unify L2+L3 path discovery into one ordered timeline ([#1436](https://github.com/MustardSeedNetworks/seed/issues/1436)) ([5f070bd](https://github.com/MustardSeedNetworks/seed/commit/5f070bd45e2f48e4c7ef0c7938556cacbbc186e4))
* **platform:** add in-process domain event bus (ADR-0004) ([#1466](https://github.com/MustardSeedNetworks/seed/issues/1466)) ([8831d8f](https://github.com/MustardSeedNetworks/seed/commit/8831d8f12cc769f0f27e74900fd362f77279b7c4))
* **platform:** add unified async job runner core (ADR-0005) ([#1467](https://github.com/MustardSeedNetworks/seed/issues/1467)) ([1194daf](https://github.com/MustardSeedNetworks/seed/commit/1194daffcf823a4d3b743a143fadfd7ff61cce4b))
* **probe:** add eight health-check vertical checkers (ADR-0027 P1) ([#1641](https://github.com/MustardSeedNetworks/seed/issues/1641)) ([99649e5](https://github.com/MustardSeedNetworks/seed/commit/99649e5c55185c3ea511677401edc01a480afb7a))
* **probe:** detect TLS certificate expiry as a probe anomaly ([#1639](https://github.com/MustardSeedNetworks/seed/issues/1639)) ([a93cac5](https://github.com/MustardSeedNetworks/seed/commit/a93cac5fd260315b5fdd6009dcf7f95cfb0cff20))
* **probe:** engine lifecycle - storage + scheduler + RunNow (Stage A1.3b) ([#1325](https://github.com/MustardSeedNetworks/seed/issues/1325)) ([0ed1138](https://github.com/MustardSeedNetworks/seed/commit/0ed113883b274fbb00e467a794086123c06929a6))
* **probe:** enrich HTTP/HTTPS checker with per-phase timings and cert summary (ADR-0027 P3a) ([#1643](https://github.com/MustardSeedNetworks/seed/issues/1643)) ([94c47ab](https://github.com/MustardSeedNetworks/seed/commit/94c47ab9a20b901c8e02d5270a980f218d6f7977))
* **probe:** make the probes table store-of-record for health-check settings (ADR-0027 P2) ([#1642](https://github.com/MustardSeedNetworks/seed/issues/1642)) ([b37fe80](https://github.com/MustardSeedNetworks/seed/commit/b37fe8024b75c2398652c2376d78daeafae67de2))
* **probe:** ping checker via TCP fallback (Stage A1.7 - 1 of N) ([#1328](https://github.com/MustardSeedNetworks/seed/issues/1328)) ([93590dc](https://github.com/MustardSeedNetworks/seed/commit/93590dc1be42295aa6fa6f30e4fa94781eb1c382))
* **probe:** rename health-check transport to /telemetry/probes/* (ADR-0027 P5) ([#1645](https://github.com/MustardSeedNetworks/seed/issues/1645)) ([8296cc2](https://github.com/MustardSeedNetworks/seed/commit/8296cc2a64645ae94cf38859b701da4270d6a873))
* **probe:** run health checks through the probe engine; delete the legacy stack (ADR-0027 P3+P4) ([#1644](https://github.com/MustardSeedNetworks/seed/issues/1644)) ([15bf342](https://github.com/MustardSeedNetworks/seed/commit/15bf3423b4bcb45636578cae5f291d5449d5efd8))
* **probe:** tcp, udp, http, https, rtsp, dicom checkers (Stage A1.7) ([#1329](https://github.com/MustardSeedNetworks/seed/issues/1329)) ([3a15010](https://github.com/MustardSeedNetworks/seed/commit/3a15010d7dd95871b011c8b0051934f851c52613))
* **profiles:** optimistic concurrency via ETag / If-Match (Phase 5) ([#1559](https://github.com/MustardSeedNetworks/seed/issues/1559)) ([558e923](https://github.com/MustardSeedNetworks/seed/commit/558e9239d63fbd9da050f398246fe17ae75cd0dc))
* **profiles:** row_version optimistic-concurrency token (Phase 5 hardening) ([#1561](https://github.com/MustardSeedNetworks/seed/issues/1561)) ([4a1c9d7](https://github.com/MustardSeedNetworks/seed/commit/4a1c9d7be9be9132bf5f3b66258676ad3e4c8407))
* **retention:** unified tier-aware retention engine (Stage A2) ([#1330](https://github.com/MustardSeedNetworks/seed/issues/1330)) ([20fd168](https://github.com/MustardSeedNetworks/seed/commit/20fd168d7267b629b01adb3b7b1af70a06b68491))
* **schema:** publish profile envelope schemas (config as opaque JSON) ([#1465](https://github.com/MustardSeedNetworks/seed/issues/1465)) ([2bd1639](https://github.com/MustardSeedNetworks/seed/commit/2bd16394b2d8db7ffb43c9f580d9caf51ec0ac08))
* **schema:** publish self-contained composer DTO schemas ([#1454](https://github.com/MustardSeedNetworks/seed/issues/1454)) ([902ce46](https://github.com/MustardSeedNetworks/seed/commit/902ce46c90ab934881f2b3f7c358d30604258916))
* **schema:** register config.Config as the code-first Config type (P7 S6.1) ([#1510](https://github.com/MustardSeedNetworks/seed/issues/1510)) ([d9b6008](https://github.com/MustardSeedNetworks/seed/commit/d9b600810c80bdffbc9276d7025cff900f5a4248))
* **schema:** register EngineDiscoveryResponse via the ADR-0008 pure-data exception (P7 S2) ([#1499](https://github.com/MustardSeedNetworks/seed/issues/1499)) ([c8599af](https://github.com/MustardSeedNetworks/seed/commit/c8599af9799014023a63fc8f989bfc7415abc563))
* **schema:** register JobResponse + CreateJobRequest DTOs (P7 S1a) ([#1497](https://github.com/MustardSeedNetworks/seed/issues/1497)) ([49ee599](https://github.com/MustardSeedNetworks/seed/commit/49ee599b674379e08158faa95b9e801f736e153e))
* **seed#1191:** multi_user CRUD + schema hardening + SSO columns ([#1204](https://github.com/MustardSeedNetworks/seed/issues/1204)) ([5c3c6b9](https://github.com/MustardSeedNetworks/seed/commit/5c3c6b9f23060b2e1b29d9f0684ffedc98aa1e37))
* **seed#1192:** multi_interface gate + Ethernet[] / WiFiList[] config ([#1206](https://github.com/MustardSeedNetworks/seed/issues/1206)) ([59fd51d](https://github.com/MustardSeedNetworks/seed/commit/59fd51d6b775c2c74b7c38b34ad41ac9f6b9ff73))
* **seed#1196:** wire multi_client gate on profile-create paths ([#1205](https://github.com/MustardSeedNetworks/seed/issues/1205)) ([2a1b2e7](https://github.com/MustardSeedNetworks/seed/commit/2a1b2e7ae1ab56fa48103abde0406731379e24aa))
* **settings:** optimistic concurrency via ETag / If-Match (Phase 5) ([#1560](https://github.com/MustardSeedNetworks/seed/issues/1560)) ([e15bfb9](https://github.com/MustardSeedNetworks/seed/commit/e15bfb906a74f7d8958f684f4f3909011263918f))
* **snmp:** arp collector (stage a3.5) ([#1338](https://github.com/MustardSeedNetworks/seed/issues/1338)) ([03efa76](https://github.com/MustardSeedNetworks/seed/commit/03efa76199c75f89a72735ef54681b6438c6ade8))
* **snmp:** bgp4_mib peer collector (stage a3.9) ([#1342](https://github.com/MustardSeedNetworks/seed/issues/1342)) ([1af3269](https://github.com/MustardSeedNetworks/seed/commit/1af32694490987e4a5af860a130b26617820aa94))
* **snmp:** cdp neighbor collector (stage a3.4b) ([#1336](https://github.com/MustardSeedNetworks/seed/issues/1336)) ([2ca9a89](https://github.com/MustardSeedNetworks/seed/commit/2ca9a89b61654f30c112801c73e9e6167c80aa4d))
* **snmp:** collector-chain poller scaffold (stage a3.1) ([#1332](https://github.com/MustardSeedNetworks/seed/issues/1332)) ([e740ae0](https://github.com/MustardSeedNetworks/seed/commit/e740ae018061ff2021ce5b7c3ebbb01ba1ffa41e))
* **snmp:** fdb collector (stage a3.6) ([#1339](https://github.com/MustardSeedNetworks/seed/issues/1339)) ([b5092e5](https://github.com/MustardSeedNetworks/seed/commit/b5092e53bfe72f91079873016fd367f281aeb71d))
* **snmp:** fdp neighbor collector via cdp wrapper (stage a3.4c) ([#1337](https://github.com/MustardSeedNetworks/seed/issues/1337)) ([cfcd0d4](https://github.com/MustardSeedNetworks/seed/commit/cfcd0d4943410d8c85e540da35759c412a9d7b7e))
* **snmp:** gosnmp-backed client factory (stage a3.5c) ([#1346](https://github.com/MustardSeedNetworks/seed/issues/1346)) ([970bd2c](https://github.com/MustardSeedNetworks/seed/commit/970bd2c18d05360624558e05b359e1d2448194dc))
* **snmp:** host_resources collector (stage a3.8) ([#1341](https://github.com/MustardSeedNetworks/seed/issues/1341)) ([7ae707f](https://github.com/MustardSeedNetworks/seed/commit/7ae707fc45285ce8d699ee6ac0c191dd4b04adad))
* **snmp:** if_table collector (stage a3.3) ([#1334](https://github.com/MustardSeedNetworks/seed/issues/1334)) ([d361c53](https://github.com/MustardSeedNetworks/seed/commit/d361c53060ce42e11595b9faa7bc4f7155f37e3d))
* **snmp:** lldp neighbor collector (stage a3.4) ([#1335](https://github.com/MustardSeedNetworks/seed/issues/1335)) ([8ed4f01](https://github.com/MustardSeedNetworks/seed/commit/8ed4f0193452d6fc864b1ddefb615a981016c04e))
* **snmp:** orchestrator + observation persistence (stage a3.5b) ([#1344](https://github.com/MustardSeedNetworks/seed/issues/1344)) ([a6c7cf1](https://github.com/MustardSeedNetworks/seed/commit/a6c7cf1854279428a24321b977aaacc0584cdd15))
* **snmp:** routing collector (stage a3.7) ([#1340](https://github.com/MustardSeedNetworks/seed/issues/1340)) ([129e105](https://github.com/MustardSeedNetworks/seed/commit/129e105a4b9a3d7c98da8db700997d206bb2b201))
* **snmp:** sys_info collector (stage a3.2) ([#1333](https://github.com/MustardSeedNetworks/seed/issues/1333)) ([7acf498](https://github.com/MustardSeedNetworks/seed/commit/7acf4988f27e72fe46d0ed3dac26fbd4fc8a93db))
* **sso:** gate settings PUT and sync IdP users on callback ([#1207](https://github.com/MustardSeedNetworks/seed/issues/1207)) ([2427d4c](https://github.com/MustardSeedNetworks/seed/commit/2427d4c27faf7d67e0a63ffd3d6ed2f744e19190))
* **sso:** settings ui for provider admin ([#1213](https://github.com/MustardSeedNetworks/seed/issues/1213)) ([c586f2f](https://github.com/MustardSeedNetworks/seed/commit/c586f2ff8c176e904097b4f0ccdd07780bba2583))
* **topology:** arp reconciler + ip/mac bindings (stage a4.4) ([#1353](https://github.com/MustardSeedNetworks/seed/issues/1353)) ([4349f3d](https://github.com/MustardSeedNetworks/seed/commit/4349f3d0017c7b0c6b76ae847ab7709815d6c01b))
* **topology:** edge reconciler upserts topology_links (stage a4.3) ([#1352](https://github.com/MustardSeedNetworks/seed/issues/1352)) ([fde7075](https://github.com/MustardSeedNetworks/seed/commit/fde70751e0311de0b7a109c9ff7e7380d134ada8))
* **topology:** if_table reconciler attaches interfaces to nodes (stage a4.2) ([#1351](https://github.com/MustardSeedNetworks/seed/issues/1351)) ([4c045b7](https://github.com/MustardSeedNetworks/seed/commit/4c045b7bfaa4444cdd05e34aa9f7ddb110b31302))
* **topology:** sys_info reconciler upserts topology_nodes (stage a4.1) ([#1350](https://github.com/MustardSeedNetworks/seed/issues/1350)) ([5d1b8df](https://github.com/MustardSeedNetworks/seed/commit/5d1b8df76a622dac647cfea7166157a28ef33819))
* **ui:** add jobs client + job event stream hook (P7 S1b) ([#1498](https://github.com/MustardSeedNetworks/seed/issues/1498)) ([9937a20](https://github.com/MustardSeedNetworks/seed/commit/9937a2037e9e9e48d5b83f496616d450179b2668))
* **ui:** add RequireRole/RequireAdmin gate + hide user mgmt from non-admins ([#1254](https://github.com/MustardSeedNetworks/seed/issues/1254)) ([#1586](https://github.com/MustardSeedNetworks/seed/issues/1586)) ([0ee5694](https://github.com/MustardSeedNetworks/seed/commit/0ee569495c9e1ea2efbd41a5c6b7ff3207d24cde))
* **ui:** add useEnginePhase hook tracking the current scan phase (P7 S3.2a) ([#1504](https://github.com/MustardSeedNetworks/seed/issues/1504)) ([79aca7e](https://github.com/MustardSeedNetworks/seed/commit/79aca7e49f157b93bb39933a43836f3396d7f653))
* **ui:** add useEngineScan hook driving discovery via the jobs spine (P7 S3.1) ([#1503](https://github.com/MustardSeedNetworks/seed/issues/1503)) ([83ea33a](https://github.com/MustardSeedNetworks/seed/commit/83ea33a5a8d4c221bb0b57c1343ef29fa183cdb3))
* **ui:** alert-rules editor page ([#1397](https://github.com/MustardSeedNetworks/seed/issues/1397)) ([af4f85c](https://github.com/MustardSeedNetworks/seed/commit/af4f85c610e0e721ebdcc5e1d87be6499b6552f3))
* **ui:** alerts page — list + ack + resolve (stage a5.7) ([#1363](https://github.com/MustardSeedNetworks/seed/issues/1363)) ([6e2d3e7](https://github.com/MustardSeedNetworks/seed/commit/6e2d3e70774d9ab874b5eade3d4c9663ca8ec448))
* **ui:** converge settings drawer shell — focus trap + slide-in (Phase 3c) ([#1236](https://github.com/MustardSeedNetworks/seed/issues/1236)) ([50da4e0](https://github.com/MustardSeedNetworks/seed/commit/50da4e0ae736920786b4193a755161f5d115f131))
* **ui:** converge Tooltip to the shared text/side design (Phase 3a) ([#1235](https://github.com/MustardSeedNetworks/seed/issues/1235)) ([f04b9c4](https://github.com/MustardSeedNetworks/seed/commit/f04b9c4cfe1610559e60806623b7db980691bf39))
* **ui:** establish semantic design-token foundation ([#1246](https://github.com/MustardSeedNetworks/seed/issues/1246)) ([49013de](https://github.com/MustardSeedNetworks/seed/commit/49013de44b56ee8bd161035b6d21edca5aef6e89))
* **ui:** migrate discovery card onto the engine-scan job (P7 S3.2b) ([#1505](https://github.com/MustardSeedNetworks/seed/issues/1505)) ([cbd634b](https://github.com/MustardSeedNetworks/seed/commit/cbd634b631200d8869dd4b837f68cb66a0258e93))
* **ui:** per-item module accent colour in the sidebar (M1 follow-up) ([#1682](https://github.com/MustardSeedNetworks/seed/issues/1682)) ([2aa1f1c](https://github.com/MustardSeedNetworks/seed/commit/2aa1f1c730fd9da38315f6417dff246eee0d9558))
* **ui:** polling targets crud page (stage a5.5) ([#1361](https://github.com/MustardSeedNetworks/seed/issues/1361)) ([c3734af](https://github.com/MustardSeedNetworks/seed/commit/c3734af7cd8aa1228707f04ca8f42c92dbb90eb0))
* **ui:** role-based write gating with RoleContext + WriteGate ([#1254](https://github.com/MustardSeedNetworks/seed/issues/1254)) ([#1267](https://github.com/MustardSeedNetworks/seed/issues/1267)) ([04a4b35](https://github.com/MustardSeedNetworks/seed/commit/04a4b35c311710d5008627181f541c940701c933))
* **ui:** sync canonical shell from stem (Phase 1) ([#1222](https://github.com/MustardSeedNetworks/seed/issues/1222)) ([271f5f4](https://github.com/MustardSeedNetworks/seed/commit/271f5f4068a77f99998575f5d727bfbf45a47f44))
* **ui:** topology page — nodes list + node detail (stage a5.6) ([#1362](https://github.com/MustardSeedNetworks/seed/issues/1362)) ([55de802](https://github.com/MustardSeedNetworks/seed/commit/55de80281393a59a1defd4e2077f8eb2d4a36904))
* **ui:** wrap SettingsDrawer in ReadOnlyView for viewer role ([#1254](https://github.com/MustardSeedNetworks/seed/issues/1254) follow-up) ([#1270](https://github.com/MustardSeedNetworks/seed/issues/1270)) ([2aa3b6d](https://github.com/MustardSeedNetworks/seed/commit/2aa3b6d97d304e544bf745fabbdc371d78f24eb0))
* **users:** settings ui for multi_user crud ([#1208](https://github.com/MustardSeedNetworks/seed/issues/1208)) ([2e3af3d](https://github.com/MustardSeedNetworks/seed/commit/2e3af3d464aae6d26fec0ddfd9eb4d675658c701))
* V1.0 unified architecture foundation (Stage A0 + A1.1-A1.5) ([#1323](https://github.com/MustardSeedNetworks/seed/issues/1323)) ([6004b02](https://github.com/MustardSeedNetworks/seed/commit/6004b02d3e1ae4e8237014206151259c5de83ea0))
* **wifi:** 802.11 decoder + airspace model foundation (W1+W2) ([#1526](https://github.com/MustardSeedNetworks/seed/issues/1526)) ([6245b89](https://github.com/MustardSeedNetworks/seed/commit/6245b89de14d3aea7b48cd175fb76be1be3ced0c))
* **wifi:** add deauth/disassoc-flood anomaly rule (w4e) ([#1544](https://github.com/MustardSeedNetworks/seed/issues/1544)) ([ff4ee00](https://github.com/MustardSeedNetworks/seed/commit/ff4ee0028cedfe3ce2f90f2583e787650682fe6e))
* **wifi:** add rogue-ap-on-lan cross-reference rule (w7) ([#1545](https://github.com/MustardSeedNetworks/seed/issues/1545)) ([acf6814](https://github.com/MustardSeedNetworks/seed/commit/acf68142d4c060d20c8d7764188a7d7d2770b28f))
* **wifi:** airspace tree + anomaly stream UI (W6) ([#1539](https://github.com/MustardSeedNetworks/seed/issues/1539)) ([5535585](https://github.com/MustardSeedNetworks/seed/commit/5535585104111e880772782a5dcc03385416561d))
* **wifi:** airspace visibility service (W5a) ([#1533](https://github.com/MustardSeedNetworks/seed/issues/1533)) ([691f644](https://github.com/MustardSeedNetworks/seed/commit/691f6446667c8c949253db102aa8e524421c11a8))
* **wifi:** anomaly catalog + airspace rules (W4b) ([#1532](https://github.com/MustardSeedNetworks/seed/issues/1532)) ([66e0a0f](https://github.com/MustardSeedNetworks/seed/commit/66e0a0fb365bfb88043c6941c405e23f3fe5be6d))
* **wifi:** bss-load + channel-width anomaly rules (W4d) ([#1540](https://github.com/MustardSeedNetworks/seed/issues/1540)) ([2b8d729](https://github.com/MustardSeedNetworks/seed/commit/2b8d72982ffe1b47a648c7d43f792e367e0cf910))
* **wifi:** four buildable-now anomaly rules (W4c) ([#1535](https://github.com/MustardSeedNetworks/seed/issues/1535)) ([988147f](https://github.com/MustardSeedNetworks/seed/commit/988147f4a44b00b06e96ce3fe91347d3af658e0f))
* **wifi:** monitor-mode auto-enablement (W3 follow-up) ([#1538](https://github.com/MustardSeedNetworks/seed/issues/1538)) ([996bc66](https://github.com/MustardSeedNetworks/seed/commit/996bc66806a226aed6417ea8e688eb5555a2bbd6))
* **wifi:** monitor-mode capture producer (W3) ([#1536](https://github.com/MustardSeedNetworks/seed/issues/1536)) ([25f6ea7](https://github.com/MustardSeedNetworks/seed/commit/25f6ea763d316feac4e3c41590de773bf0f4071d))
* **wifi:** pro-gated airspace + anomaly read API (W5b) ([#1534](https://github.com/MustardSeedNetworks/seed/issues/1534)) ([05a5bbf](https://github.com/MustardSeedNetworks/seed/commit/05a5bbfdb787cb055935ebcf4c91f2714e97dbf3))
* **wifi:** regulatory-violation rule (802.11d, 2.4 GHz) ([#1537](https://github.com/MustardSeedNetworks/seed/issues/1537)) ([336a0fa](https://github.com/MustardSeedNetworks/seed/commit/336a0fa3a73f617e6824fa8d8dc1586ddae3f004))
* **wifi:** run the anomaly engine over wi-fi survey samples ([#1543](https://github.com/MustardSeedNetworks/seed/issues/1543)) ([dcee9c8](https://github.com/MustardSeedNetworks/seed/commit/dcee9c821470377c605cd46752c3f2b97fc78851))


### Bug Fixes

* **api:** require operator role on persistent-write routes ([#1631](https://github.com/MustardSeedNetworks/seed/issues/1631)) ([fc7e3f5](https://github.com/MustardSeedNetworks/seed/commit/fc7e3f5e2bb3d6af97652e74f45198c2509546ae))
* **auth:** stop a late mount /status probe from clobbering a completed login ([#1598](https://github.com/MustardSeedNetworks/seed/issues/1598)) ([cc43f4a](https://github.com/MustardSeedNetworks/seed/commit/cc43f4a86933502bcaa71a1eedb40be194f1761b))
* **build:** honest quality gates — propagate exit codes, fix hidden lint/race findings ([#1723](https://github.com/MustardSeedNetworks/seed/issues/1723)) ([474fcaf](https://github.com/MustardSeedNetworks/seed/commit/474fcafc4a5d974981dd6392035db2f89e67dc65))
* **config:** unify on-disk config format to JSON (seed.json) ([#1528](https://github.com/MustardSeedNetworks/seed/issues/1528)) ([3663f25](https://github.com/MustardSeedNetworks/seed/commit/3663f25090d7801d568bcf8c6e18787902c7dc2f))
* **contract:** gen-types handles nested-type DTOs (unblocks bulk rollout) ([#1415](https://github.com/MustardSeedNetworks/seed/issues/1415)) ([323e0fd](https://github.com/MustardSeedNetworks/seed/commit/323e0fd3f34f4b1697f85079bc9ab78b68cd8e18))
* **database:** de-collide anomaly census test fixtures (flaky) ([#1670](https://github.com/MustardSeedNetworks/seed/issues/1670)) ([c6414cd](https://github.com/MustardSeedNetworks/seed/commit/c6414cd709511719595dce59560ccca8cffddd7b))
* **database:** enforce owner-only (0600) permissions on the database file ([#1546](https://github.com/MustardSeedNetworks/seed/issues/1546)) ([1f4c989](https://github.com/MustardSeedNetworks/seed/commit/1f4c9892511978ff0a87196fc02e5480ce78e1b6))
* **deploy:** do not auto-open the firewall on install; require opt-in ([#1529](https://github.com/MustardSeedNetworks/seed/issues/1529)) ([437cb00](https://github.com/MustardSeedNetworks/seed/commit/437cb0066ce1e69992e9e786975deddc0a7c1d4e))
* **e2e,test:** repair v1-API URL drift + EventSource polyfill ([#1146](https://github.com/MustardSeedNetworks/seed/issues/1146)) ([972f1e3](https://github.com/MustardSeedNetworks/seed/commit/972f1e3fec89e74ed1dabd16eeb7fc214ec8b478))
* **e2e:** add header-logout testid + retire SVG class fallback in auth-complete ([#1177](https://github.com/MustardSeedNetworks/seed/issues/1177)) ([1a6c9ef](https://github.com/MustardSeedNetworks/seed/commit/1a6c9ef729dd5e3ecfeb44b0cffc9ed1b3fb2701))
* **e2e:** bulk-replace brittle heading regexes with getByTestId ([#1162](https://github.com/MustardSeedNetworks/seed/issues/1162)) ([22f9c06](https://github.com/MustardSeedNetworks/seed/commit/22f9c067623f14036b90f2a24704456ec9efbb74))
* **e2e:** de-flake gateway IPv6/IPv4 test on WebKit ([#1518](https://github.com/MustardSeedNetworks/seed/issues/1518)) ([2a4029e](https://github.com/MustardSeedNetworks/seed/commit/2a4029e682897096dc26aa77deab74bcde856bb0))
* **e2e:** drop .or() in 401 test - strict-mode violation when both render ([#1322](https://github.com/MustardSeedNetworks/seed/issues/1322)) ([d00719b](https://github.com/MustardSeedNetworks/seed/commit/d00719b357740e719e3643ed679426d0b571d44e))
* **e2e:** drop top-level theme-toggle smoke test from seed ([#1297](https://github.com/MustardSeedNetworks/seed/issues/1297)) ([f4d44ea](https://github.com/MustardSeedNetworks/seed/commit/f4d44ea25b55571f06e33d758cab00279a0f4b14))
* **e2e:** isolate responsive logout tests so they don't poison shared storageState ([#1176](https://github.com/MustardSeedNetworks/seed/issues/1176)) ([935f318](https://github.com/MustardSeedNetworks/seed/commit/935f318a96add5823e80e275088412d9e4a2c51d))
* **e2e:** mock /api/v1/sap/gateway + add NetworkPage help + race-free FAB keyboard test ([#1321](https://github.com/MustardSeedNetworks/seed/issues/1321)) ([3443d99](https://github.com/MustardSeedNetworks/seed/commit/3443d9974481f6309ddd1f92f91a263e188692f2))
* **e2e:** override storageState for 4 login-form error-scenarios ([#1316](https://github.com/MustardSeedNetworks/seed/issues/1316)) ([26451bb](https://github.com/MustardSeedNetworks/seed/commit/26451bbed33f5121d6dc2311cfb4948f35bf5373))
* **e2e:** remove garbage JS in theme-and-help.spec.ts (closes [#1169](https://github.com/MustardSeedNetworks/seed/issues/1169)) ([#1171](https://github.com/MustardSeedNetworks/seed/issues/1171)) ([7cd0d74](https://github.com/MustardSeedNetworks/seed/commit/7cd0d74ca7910fe3e078583cb0b7eead63fe3ee3))
* **e2e:** replace brittle text regexes with stable id selectors (Category B) ([#1174](https://github.com/MustardSeedNetworks/seed/issues/1174)) ([879ee93](https://github.com/MustardSeedNetworks/seed/commit/879ee93fd7fef910620307b9d2526a37e533dc48))
* **e2e:** replace per-page H1 heading regexes with getByTestId (Category C) ([#1173](https://github.com/MustardSeedNetworks/seed/issues/1173)) ([484f6a4](https://github.com/MustardSeedNetworks/seed/commit/484f6a49c6d786d2099aa50afc2ee847e23647ad))
* **e2e:** replace remaining /login/ heading regex in auth.spec.ts ([#1163](https://github.com/MustardSeedNetworks/seed/issues/1163)) ([e2e9af4](https://github.com/MustardSeedNetworks/seed/commit/e2e9af4d0adb5c26400ccaf7584471bafa590e2b))
* **e2e:** replace remaining settings-drawer text regexes with testid ([#1179](https://github.com/MustardSeedNetworks/seed/issues/1179)) ([bb36015](https://github.com/MustardSeedNetworks/seed/commit/bb36015816d9e410609b1fb12f2999b7a46ff5ff))
* **e2e:** repoint seed specs to sidebar Settings/Help after Phase 2 ([#1231](https://github.com/MustardSeedNetworks/seed/issues/1231)) ([1c4299d](https://github.com/MustardSeedNetworks/seed/commit/1c4299d8c75353c584f282df9b027309bbe4d2de))
* **e2e:** rewrite global-setup to run login in a single chromium context ([#1172](https://github.com/MustardSeedNetworks/seed/issues/1172)) ([385220c](https://github.com/MustardSeedNetworks/seed/commit/385220cc6bd90f0619f1624fb1d43dd93b1773d6))
* **e2e:** rewrite system-theme test with colorScheme emulation (real assertion) ([#1189](https://github.com/MustardSeedNetworks/seed/issues/1189)) ([6dc2c80](https://github.com/MustardSeedNetworks/seed/commit/6dc2c80c67330db25d820142e7b6f39e0fd1515d))
* **e2e:** route gateway + dashboard card tests to the right pages ([#1315](https://github.com/MustardSeedNetworks/seed/issues/1315)) ([0484ce2](https://github.com/MustardSeedNetworks/seed/commit/0484ce2655684ae6e88cf612fee81b0b58bfebaf))
* **e2e:** sync FAB tests on data-running attribute, not animate-spin (Category D) ([#1175](https://github.com/MustardSeedNetworks/seed/issues/1175)) ([e939137](https://github.com/MustardSeedNetworks/seed/commit/e9391370833c5267dddc8e6b3bf3a3991818bb43))
* **e2e:** use #profile-modal-title id for profile-management modal assertion ([#1178](https://github.com/MustardSeedNetworks/seed/issues/1178)) ([e7fec8d](https://github.com/MustardSeedNetworks/seed/commit/e7fec8d734cd3a3ccc7247f4b050928af28889be))
* **e2e:** use data-testid for auth login + page header selectors ([#1161](https://github.com/MustardSeedNetworks/seed/issues/1161)) ([f6a0848](https://github.com/MustardSeedNetworks/seed/commit/f6a0848a48cdae0a6d5a28f55f04290cc3972c50))
* **help-modal:** add esc handler + testid; fix e2e selectors ([#1228](https://github.com/MustardSeedNetworks/seed/issues/1228)) ([e1a74a5](https://github.com/MustardSeedNetworks/seed/commit/e1a74a536206bee7ec5ad3d929fd84387aeeaa26))
* **help:** add HelpDrawer sections for alerts, polling-targets, topology ([#1425](https://github.com/MustardSeedNetworks/seed/issues/1425)) ([758f485](https://github.com/MustardSeedNetworks/seed/commit/758f4857de6d36db7fd3af5b6e5fc7c6aa4d74af))
* **i18n:** replace banned 'open source' with 'source-available' per CLAUDE.md ([#1184](https://github.com/MustardSeedNetworks/seed/issues/1184)) ([ac207b3](https://github.com/MustardSeedNetworks/seed/commit/ac207b3ced7c666f7c4293256505e70dbe0868f8))
* **i18n:** resolve 329 t() calls referencing missing EN locale keys ([#1211](https://github.com/MustardSeedNetworks/seed/issues/1211)) ([3502972](https://github.com/MustardSeedNetworks/seed/commit/3502972c501803da7adee56a74833abb9279d279))
* **i18n:** update document.lang on locale change for a11y ([#1186](https://github.com/MustardSeedNetworks/seed/issues/1186)) ([c357405](https://github.com/MustardSeedNetworks/seed/commit/c3574055da426d1b55244c23c69a59385614daa8))
* **license:** add RWMutex to Manager for safe concurrent access ([#1152](https://github.com/MustardSeedNetworks/seed/issues/1152)) ([810cfd9](https://github.com/MustardSeedNetworks/seed/commit/810cfd9ead9d4d13b17e1a43909f4e5798f0bfcc))
* **license:** memoize device fingerprint to stop spurious invalidation ([#1523](https://github.com/MustardSeedNetworks/seed/issues/1523)) ([50aabc8](https://github.com/MustardSeedNetworks/seed/commit/50aabc896da33b9603cc13f9e2fb2533fe767981))
* **probe:** seed factory health-check targets on first run ([#1646](https://github.com/MustardSeedNetworks/seed/issues/1646)) ([4283b6e](https://github.com/MustardSeedNetworks/seed/commit/4283b6e228851c4e201db7799c556be1c4de49a1))
* **scripts:** clean up all shellcheck warnings + pin severity=warning ([#1144](https://github.com/MustardSeedNetworks/seed/issues/1144)) ([0be82a4](https://github.com/MustardSeedNetworks/seed/commit/0be82a4e80f9f9683debf69563e5baea7a0ab500))
* **security:** rate-limit /auth/refresh ([#1224](https://github.com/MustardSeedNetworks/seed/issues/1224)) ([#1243](https://github.com/MustardSeedNetworks/seed/issues/1243)) ([641b79d](https://github.com/MustardSeedNetworks/seed/commit/641b79dd4e9f73a956cd01033c5c62414ae156ed))
* **security:** resolve CodeQL alerts (allocation caps, TLS inspect, secure RNG) ([#1728](https://github.com/MustardSeedNetworks/seed/issues/1728)) ([2a39e22](https://github.com/MustardSeedNetworks/seed/commit/2a39e22a6ae93d6e00843689e9bd1cdd4d887092))
* **test:** serialize snmptrap tests to avoid upstream gosnmp race ([#1388](https://github.com/MustardSeedNetworks/seed/issues/1388)) ([6280df4](https://github.com/MustardSeedNetworks/seed/commit/6280df4ca4d8fde3a33043b839e8f722da0d1ab5))
* **ui:** Add data-testid card + update e2e selector (kill last pre-existing E2E flake) ([#1154](https://github.com/MustardSeedNetworks/seed/issues/1154)) ([e5293dc](https://github.com/MustardSeedNetworks/seed/commit/e5293dcd871907ca3b9a4f3c25fca640a88cdbb2))
* **ui:** bring AlertsPage onto semantic design tokens (P7 S0) ([#1492](https://github.com/MustardSeedNetworks/seed/issues/1492)) ([5826432](https://github.com/MustardSeedNetworks/seed/commit/5826432c45cd61557657fded5e8e87889ba08b19))
* **ui:** bring PollingTargetsPage onto semantic design tokens (P7 S0) ([#1493](https://github.com/MustardSeedNetworks/seed/issues/1493)) ([a295e3e](https://github.com/MustardSeedNetworks/seed/commit/a295e3e8f17ab3d76bf898a4352200e3d1ec3475))
* **ui:** bring TopologyPage onto semantic design tokens (P7 S0) ([#1491](https://github.com/MustardSeedNetworks/seed/issues/1491)) ([36e61e5](https://github.com/MustardSeedNetworks/seed/commit/36e61e5b6a801d545627e7d2bc57dc38ea482227))
* **ui:** close SEED_UI_ARCH_PLAN D-batch — distinct brand green (L1) + profile entry points (VL1) + on-brand AA (VL2) ([#1680](https://github.com/MustardSeedNetworks/seed/issues/1680)) ([af1e7d1](https://github.com/MustardSeedNetworks/seed/commit/af1e7d1496b965f3c507df7fa9cd6c8af3d7b35e))
* **ui:** fix stale brand green in canvas/PDF; wire canvas markers to tokens ([#1249](https://github.com/MustardSeedNetworks/seed/issues/1249)) ([055c923](https://github.com/MustardSeedNetworks/seed/commit/055c9237d0d9aa7901672852defca1ce1c4a589d))
* **ui:** logout synchronously clears legacy localStorage keys ([#1317](https://github.com/MustardSeedNetworks/seed/issues/1317)) ([7233b8c](https://github.com/MustardSeedNetworks/seed/commit/7233b8c5d6ba2a6aec1a0c601f6affc19f835291))
* **ui:** re-sync shell from stem — sidebar shows the product name ([#1233](https://github.com/MustardSeedNetworks/seed/issues/1233)) ([2a4641b](https://github.com/MustardSeedNetworks/seed/commit/2a4641b367ccf17e38c57d8e67ed28f01dca054f))
* **ui:** re-sync shell from stem to pull page-header-title testid ([#1230](https://github.com/MustardSeedNetworks/seed/issues/1230)) ([7092857](https://github.com/MustardSeedNetworks/seed/commit/7092857d3c7f20eec93300628d7bf847fe91af3e))
* **ui:** rename customer-facing 'Wi-Fi Planning Mode' to 'Wi-Fi Survey Mode' (l3) ([#1569](https://github.com/MustardSeedNetworks/seed/issues/1569)) ([bad996e](https://github.com/MustardSeedNetworks/seed/commit/bad996e1e45aa13004fa3d2f64de55db09eae992))
* **ui:** repair token-discipline guard and close remaining color leaks ([#1251](https://github.com/MustardSeedNetworks/seed/issues/1251)) ([6dd96c0](https://github.com/MustardSeedNetworks/seed/commit/6dd96c0c55a7f1c02cb460a6d9db08bca4cc6c7b))
* **ui:** replace broken help modal with data-driven help drawer ([#43](https://github.com/MustardSeedNetworks/seed/issues/43)) ([#1239](https://github.com/MustardSeedNetworks/seed/issues/1239)) ([11384f3](https://github.com/MustardSeedNetworks/seed/commit/11384f3ad2aa3a3bf73065d5648e45300bb0686e))
* **ui:** replace more undefined color tokens ([#1263](https://github.com/MustardSeedNetworks/seed/issues/1263)) ([7e9cc0e](https://github.com/MustardSeedNetworks/seed/commit/7e9cc0edffcb13244be23c28a37e84b56a3f38ab))
* **ui:** replace undefined bg-surface-secondary token in SLA card ([#1253](https://github.com/MustardSeedNetworks/seed/issues/1253)) ([5d4a922](https://github.com/MustardSeedNetworks/seed/commit/5d4a9228c80546501eadcd678869721a99fd524d))
* **ui:** replace undefined hover:bg-brand-primary-hover token ([#1262](https://github.com/MustardSeedNetworks/seed/issues/1262)) ([19dc373](https://github.com/MustardSeedNetworks/seed/commit/19dc373ca593b26399e83aa82b68b01ad938c28e))
* **ui:** settings drawer focus trap — drop stopPropagation that defeated it ([#1240](https://github.com/MustardSeedNetworks/seed/issues/1240)) ([41c2cbd](https://github.com/MustardSeedNetworks/seed/commit/41c2cbd6498cb98118af85537d20fa18c2e10e0c))
* **ui:** stop the FAB presenting a partial run as complete (c2) ([#1568](https://github.com/MustardSeedNetworks/seed/issues/1568)) ([07cecb7](https://github.com/MustardSeedNetworks/seed/commit/07cecb7411a16a3596139ba8935ba8f801a480f1))
* **ui:** suppress node dep0205 build warning ([e3dadd7](https://github.com/MustardSeedNetworks/seed/commit/e3dadd74db4be15a8228a5129e0a8d7c535e04df))
* **ui:** surface NMS pages in the sidebar + guard nav/route parity (h3) ([#1565](https://github.com/MustardSeedNetworks/seed/issues/1565)) ([c02d9a2](https://github.com/MustardSeedNetworks/seed/commit/c02d9a2637a3c3276d9bc1947a36a8427192b960))
* **ui:** unblock frontend gate (@storybook/react devDep) + repair reports-page e2e ([#1566](https://github.com/MustardSeedNetworks/seed/issues/1566)) ([9dfe217](https://github.com/MustardSeedNetworks/seed/commit/9dfe21748e5e24f8385137d1442fcad5b310f772))
* **ui:** users settings TS error blocking CI strict tsc check ([#1229](https://github.com/MustardSeedNetworks/seed/issues/1229)) ([a0b5ced](https://github.com/MustardSeedNetworks/seed/commit/a0b5ced1aa69fef87e33c411783b140ca2a59d37))


### Performance Improvements

* **ui:** split vendor chunks, add modern build target + analyzer ([#1599](https://github.com/MustardSeedNetworks/seed/issues/1599)) ([f451a70](https://github.com/MustardSeedNetworks/seed/commit/f451a70af38814f805ed97f7dc6846887fb3f723))


### Code Refactoring

* **alerts:** relocate Alert/Rule/ListenerEvent rows to domain pkgs, make persistence-free (WS-B) ([#1676](https://github.com/MustardSeedNetworks/seed/issues/1676)) ([0323857](https://github.com/MustardSeedNetworks/seed/commit/0323857e9d56274430e766e76e3790415ed93b4a))
* **anomaly:** converge producers onto one server-owned engine (ADR-0029 P2+P3) ([#1652](https://github.com/MustardSeedNetworks/seed/issues/1652)) ([58007b4](https://github.com/MustardSeedNetworks/seed/commit/58007b4c36b3bbb011712901f52ed178b2aac6c8))
* **anomaly:** delete bespoke health detector, read unified store (ADR-0021 phase 4) ([#1633](https://github.com/MustardSeedNetworks/seed/issues/1633)) ([438e3d1](https://github.com/MustardSeedNetworks/seed/commit/438e3d1b7dbb8256ce2cf31c60fde2485d9ca627))
* **anomaly:** source on detection + source-scoped prune (ADR-0029 P1) ([#1651](https://github.com/MustardSeedNetworks/seed/issues/1651)) ([21dc576](https://github.com/MustardSeedNetworks/seed/commit/21dc576d5cdc573ce64128230615175278bff469))
* **api,config:** remove HTTP→HTTPS redirector — HTTPS-only ([#1277](https://github.com/MustardSeedNetworks/seed/issues/1277)) ([ab9d4f3](https://github.com/MustardSeedNetworks/seed/commit/ab9d4f3a59397b90db145b593b7ea1b0351b526c))
* **api:** add flat transport mirrors for 4 domain-nested DTOs ([#1461](https://github.com/MustardSeedNetworks/seed/issues/1461)) ([de7244e](https://github.com/MustardSeedNetworks/seed/commit/de7244ef1644ebf59435bf61f1fdb3516651fc7e))
* **api:** add flat transport mirrors for the Bluetooth DTOs ([#1462](https://github.com/MustardSeedNetworks/seed/issues/1462)) ([6258793](https://github.com/MustardSeedNetworks/seed/commit/62587937a8cd2022af9152f01e2c01543826442c))
* **api:** add flat transport mirrors for the Wi-Fi discovery DTOs ([#1463](https://github.com/MustardSeedNetworks/seed/issues/1463)) ([d2491ea](https://github.com/MustardSeedNetworks/seed/commit/d2491ea3fbcead79466f0191ad1dfbc25865a38e))
* **api:** camelCase the bluetooth DTO JSON tags (P8.2 pilot, ADR-0010) ([#1516](https://github.com/MustardSeedNetworks/seed/issues/1516)) ([962af1c](https://github.com/MustardSeedNetworks/seed/commit/962af1c52471239d1b616f9234f4ef9e5862dfc5))
* **api:** camelcase the logs, profiles, and config-version apis (no grandfathering) ([#1606](https://github.com/MustardSeedNetworks/seed/issues/1606)) ([865fc78](https://github.com/MustardSeedNetworks/seed/commit/865fc78f29faa791ec92a20510a803f67a1a9c2f))
* **api:** camelcase the mfa + auth-login apis (no grandfathering) ([#1607](https://github.com/MustardSeedNetworks/seed/issues/1607)) ([bac77f5](https://github.com/MustardSeedNetworks/seed/commit/bac77f5c947b772755fb373b79aea948dcd21462))
* **api:** clean-hexagonal alerts retrofit (ADR-0020) ([#1612](https://github.com/MustardSeedNetworks/seed/issues/1612)) ([412e2b9](https://github.com/MustardSeedNetworks/seed/commit/412e2b984f715a861dd992b5bc99ea551235b4cc))
* **api:** clean-hexagonal discovery retrofit (ADR-0020) ([#1616](https://github.com/MustardSeedNetworks/seed/issues/1616)) ([6d0c035](https://github.com/MustardSeedNetworks/seed/commit/6d0c0352bfc159ade52df7b2a2c19244e6864cb7))
* **api:** clean-hexagonal health-monitoring retrofit (ADR-0020) ([#1618](https://github.com/MustardSeedNetworks/seed/issues/1618)) ([4678697](https://github.com/MustardSeedNetworks/seed/commit/46786970f84b34681749630cd53373144ead5258))
* **api:** clean-hexagonal network exemplar + ADR-0020 ([#1611](https://github.com/MustardSeedNetworks/seed/issues/1611)) ([dbaf8dc](https://github.com/MustardSeedNetworks/seed/commit/dbaf8dcac55b6805b00fa5695293ef8bac1005f4))
* **api:** clean-hexagonal profiles retrofit (ADR-0020) ([#1614](https://github.com/MustardSeedNetworks/seed/issues/1614)) ([b5516b2](https://github.com/MustardSeedNetworks/seed/commit/b5516b2267fce5a8a8508ea9a6059c4b3c62767b))
* **api:** clean-hexagonal settings retrofit (ADR-0020) ([#1613](https://github.com/MustardSeedNetworks/seed/issues/1613)) ([5d98097](https://github.com/MustardSeedNetworks/seed/commit/5d980976222a0ed2aad5ce459c7eb85cb4ec70b6))
* **api:** clean-hexagonal wifi retrofit (ADR-0020) ([#1615](https://github.com/MustardSeedNetworks/seed/issues/1615)) ([e903744](https://github.com/MustardSeedNetworks/seed/commit/e90374445d7b1eb66b2d1cb91cc40e9b6319973d))
* **api:** delete ServiceContainer, flatten services onto Server (D1) ([#1627](https://github.com/MustardSeedNetworks/seed/issues/1627)) ([feb198e](https://github.com/MustardSeedNetworks/seed/commit/feb198e3df9db18ebb83aa497535f550f8d18fc2))
* **api:** drop redundant in-handler method guards (registry is authoritative) ([#1597](https://github.com/MustardSeedNetworks/seed/issues/1597)) ([2abceb3](https://github.com/MustardSeedNetworks/seed/commit/2abceb33e30ec293007039cae1b14c4c159d9059))
* **api:** extract composed middleware chain into Server.Handler() ([#1404](https://github.com/MustardSeedNetworks/seed/issues/1404)) ([dd5e572](https://github.com/MustardSeedNetworks/seed/commit/dd5e572d6e04fb9dc9b7790dde5e4b3707556d00))
* **api:** extract drainJobSubstrate + test the shutdown drain ([#1628](https://github.com/MustardSeedNetworks/seed/issues/1628)) ([1b8d2cf](https://github.com/MustardSeedNetworks/seed/commit/1b8d2cf5c69833b0152a44b734b41b8aeb73c213))
* **api:** finish the profiles handler strangle (ADR-0020, WS-A11c) ([#1669](https://github.com/MustardSeedNetworks/seed/issues/1669)) ([c92f82b](https://github.com/MustardSeedNetworks/seed/commit/c92f82b865c1881b4dda210bef3ab07c0f81e657))
* **api:** flat transport mirrors for PathResponse + survey-import request ([#1464](https://github.com/MustardSeedNetworks/seed/issues/1464)) ([24a2b85](https://github.com/MustardSeedNetworks/seed/commit/24a2b85b6a3446e836f64c6d39aa4dc3562be6d1))
* **api:** retire the /security/pipeline/* endpoints (P7 retirement R1) ([#1507](https://github.com/MustardSeedNetworks/seed/issues/1507)) ([1ca258d](https://github.com/MustardSeedNetworks/seed/commit/1ca258dadb3dc232242018738e7901f877acebe9))
* **api:** route handler license reads through s.licenseManager() (D1 prep) ([#1626](https://github.com/MustardSeedNetworks/seed/issues/1626)) ([54318de](https://github.com/MustardSeedNetworks/seed/commit/54318de5ef736edddf58b1776f736ca05d14f787))
* **api:** route the config-read/write residuals through use-cases (ADR-0020, WS-A11a) ([#1665](https://github.com/MustardSeedNetworks/seed/issues/1665)) ([50d92e5](https://github.com/MustardSeedNetworks/seed/commit/50d92e5a89cc4ba7ef10db8691edbd3876bcf462))
* **api:** split GatewayResponse into a flat non-recursive transport DTO ([#1460](https://github.com/MustardSeedNetworks/seed/issues/1460)) ([9069be5](https://github.com/MustardSeedNetworks/seed/commit/9069be5ae6f1ba2cd74042d3eda44689c5d7f2e5))
* **api:** strangle alert inbox into a use-case (ADR-0020, WS-A8) ([#1662](https://github.com/MustardSeedNetworks/seed/issues/1662)) ([c25d4cf](https://github.com/MustardSeedNetworks/seed/commit/c25d4cf20fe0f1db1dfe8c8e443f42996e5a209d))
* **api:** strangle alert-rule handlers into alertsapp use-case (ADR-0016) ([#1557](https://github.com/MustardSeedNetworks/seed/issues/1557)) ([e08efba](https://github.com/MustardSeedNetworks/seed/commit/e08efba8530622279692898ddaba98e808c54743))
* **api:** strangle config backup/restore into a use-case (ADR-0020, WS-A9) ([#1663](https://github.com/MustardSeedNetworks/seed/issues/1663)) ([5460177](https://github.com/MustardSeedNetworks/seed/commit/54601779883293399a0472a98be3e78463b0f799))
* **api:** strangle device-discovery settings into a use-case (ADR-0020, WS-A1) ([#1655](https://github.com/MustardSeedNetworks/seed/issues/1655)) ([f39f91e](https://github.com/MustardSeedNetworks/seed/commit/f39f91e45d81fd13f26758f52d0a366edd1e3220))
* **api:** strangle diagnostic export into a use-case (ADR-0020, WS-A10) ([#1664](https://github.com/MustardSeedNetworks/seed/issues/1664)) ([b1774de](https://github.com/MustardSeedNetworks/seed/commit/b1774de2d1963efa22e21511783cc7d617808ac8))
* **api:** strangle engine-status into a use-case over the engine registry (ADR-0020) ([#1625](https://github.com/MustardSeedNetworks/seed/issues/1625)) ([1f2c242](https://github.com/MustardSeedNetworks/seed/commit/1f2c242a6af96780696974bf9dd2e212fff5265b))
* **api:** strangle health-checks settings into a use-case (ADR-0020, WS-A4) ([#1658](https://github.com/MustardSeedNetworks/seed/issues/1658)) ([6967217](https://github.com/MustardSeedNetworks/seed/commit/696721726d007f55b6a596b86e7f2ecfbc581a4e))
* **api:** strangle identity (users/oauth/tokens) into use-cases over repository ports (C4) ([#1624](https://github.com/MustardSeedNetworks/seed/issues/1624)) ([8c335f8](https://github.com/MustardSeedNetworks/seed/commit/8c335f82a8396e0ed5242f873a8a37abc6d0db58))
* **api:** strangle internal/api — ADR-0016 + Wi-Fi visibility phase 1 ([#1541](https://github.com/MustardSeedNetworks/seed/issues/1541)) ([a9c6fd0](https://github.com/MustardSeedNetworks/seed/commit/a9c6fd003acf303c50f505cb2195eee5fb781940))
* **api:** strangle ip-config + mtu handlers into networkapp (ADR-0016) ([#1555](https://github.com/MustardSeedNetworks/seed/issues/1555)) ([d9ae260](https://github.com/MustardSeedNetworks/seed/commit/d9ae2608964a93482d9410725f74c938a67c7273))
* **api:** strangle main settings into a use-case (ADR-0020, WS-A2) ([#1656](https://github.com/MustardSeedNetworks/seed/issues/1656)) ([c9f16ce](https://github.com/MustardSeedNetworks/seed/commit/c9f16cefacc7415254b49c5c3894e3a0b8ce950e))
* **api:** strangle polling-targets CRUD into a use-case (ADR-0020, WS-A7) ([#1661](https://github.com/MustardSeedNetworks/seed/issues/1661)) ([3640352](https://github.com/MustardSeedNetworks/seed/commit/364035220a43b59357a5716e12bf514aae3f9a6a))
* **api:** strangle profile handlers into profilesapp use-case (ADR-0016) ([#1554](https://github.com/MustardSeedNetworks/seed/issues/1554)) ([3236034](https://github.com/MustardSeedNetworks/seed/commit/323603425f0ac09841c33044498d0e55a7658300))
* **api:** strangle security settings into a use-case + fix SNMP deadlock (ADR-0020, WS-A3) ([#1657](https://github.com/MustardSeedNetworks/seed/issues/1657)) ([7146550](https://github.com/MustardSeedNetworks/seed/commit/714655090163ec6fe53fd43e9c97cc94ff479929))
* **api:** strangle settings handlers into settingsapp use-case (ADR-0016) ([#1550](https://github.com/MustardSeedNetworks/seed/issues/1550)) ([be4a415](https://github.com/MustardSeedNetworks/seed/commit/be4a415fdda3aa3e023997f4c48aec31365441ab))
* **api:** strangle the last residual handler config/db reaches (ADR-0020, WS-A) ([#1671](https://github.com/MustardSeedNetworks/seed/issues/1671)) ([ba8d183](https://github.com/MustardSeedNetworks/seed/commit/ba8d183ba3d0adcdae92314900526dc763c4d39e))
* **api:** strangle the log query into a use-case (ADR-0020, WS-A11b) ([#1668](https://github.com/MustardSeedNetworks/seed/issues/1668)) ([9e22a05](https://github.com/MustardSeedNetworks/seed/commit/9e22a0516eea99a31ed116b895b4824f9f3f7f18))
* **api:** strangle topology read endpoints into a use-case (ADR-0020, WS-A6) ([#1660](https://github.com/MustardSeedNetworks/seed/issues/1660)) ([b306028](https://github.com/MustardSeedNetworks/seed/commit/b30602829e09df79d561e39fedb0a3dffd7abade))
* **api:** strangle update handlers into internal/update/lifecycle (C3) ([#1620](https://github.com/MustardSeedNetworks/seed/issues/1620)) ([decba0c](https://github.com/MustardSeedNetworks/seed/commit/decba0c78f65dc2522fcfe6e5148cefe386304fa))
* **api:** strangle vulnerability-scanner settings into the security use-case (ADR-0020, WS-A5) ([#1659](https://github.com/MustardSeedNetworks/seed/issues/1659)) ([1d0d33e](https://github.com/MustardSeedNetworks/seed/commit/1d0d33edf428950482953cec542b4bb9b7c7299f))
* **api:** strangle wi-fi management + discovery handlers into wifiapp use-cases ([#1542](https://github.com/MustardSeedNetworks/seed/issues/1542)) ([469bb3a](https://github.com/MustardSeedNetworks/seed/commit/469bb3ab6f3e52d04af71895ec2b513ba687e674))
* **api:** use HasFeature("rest_api") for PAT minting gate ([#1281](https://github.com/MustardSeedNetworks/seed/issues/1281)) ([6135c25](https://github.com/MustardSeedNetworks/seed/commit/6135c25fb4962a3dda5bd0923274a3a0cb8214c1))
* **arch:** drop dead health_report.go, cutting harvest-&gt;health coupling (Phase 3 1b-ii) ([#1428](https://github.com/MustardSeedNetworks/seed/issues/1428)) ([6e311cd](https://github.com/MustardSeedNetworks/seed/commit/6e311cda7930a4470c0cf95ec2dc6914d1455c65))
* **arch:** relocate harvest to internal/modules/harvest (Phase 3 pilot 1a) ([#1427](https://github.com/MustardSeedNetworks/seed/issues/1427)) ([bd2e91e](https://github.com/MustardSeedNetworks/seed/commit/bd2e91eb2166efc6b8cfb38f0fb6dec3918e6579))
* camelcase the sso-config + cve-db apis; finish OUR de-grandfathering ([#1608](https://github.com/MustardSeedNetworks/seed/issues/1608)) ([e072884](https://github.com/MustardSeedNetworks/seed/commit/e0728848990f356d9a4c085113cf6df677adb746))
* **capture:** add Capture port + pcap/nullcapture adapters (Phase 6 S1a) ([#1487](https://github.com/MustardSeedNetworks/seed/issues/1487)) ([bf6506f](https://github.com/MustardSeedNetworks/seed/commit/bf6506f74a67c97e087a871e60a509d1f078b573))
* **capture:** route discovery/dhcp/vlan capture through the port (Phase 6 S1b) ([#1488](https://github.com/MustardSeedNetworks/seed/issues/1488)) ([f101c00](https://github.com/MustardSeedNetworks/seed/commit/f101c0037aca2e1e262e363d22042407e9dfe66b))
* **ci:** add domain-purity depguard rules for six capability packages (WS-C4) ([#1697](https://github.com/MustardSeedNetworks/seed/issues/1697)) ([134e9eb](https://github.com/MustardSeedNetworks/seed/commit/134e9ebd57c7fb54c09081c2fd0bed7a774deb4d))
* **ci:** empty the json-casing baseline; exempt external adapters by marker (ADR-0010 revised) ([#1684](https://github.com/MustardSeedNetworks/seed/issues/1684)) ([1df12dd](https://github.com/MustardSeedNetworks/seed/commit/1df12dd6c5a80fd8a9ade6b746aea8aabbe0be66))
* **config:** drop the unused pipeline config (P7 retirement follow-up) ([#1509](https://github.com/MustardSeedNetworks/seed/issues/1509)) ([506f455](https://github.com/MustardSeedNetworks/seed/commit/506f455727aaaa79ddad8e883e5cfcab7c2c86db))
* **config:** reject plaintext credentials, delete legacy v0/JWT path ([#1623](https://github.com/MustardSeedNetworks/seed/issues/1623)) ([05350bf](https://github.com/MustardSeedNetworks/seed/commit/05350bfbece01b143d7f7fde135fa52249d6bc6e))
* **contract:** retire hand-maintained TS twins for 6 generated DTOs ([#1424](https://github.com/MustardSeedNetworks/seed/issues/1424)) ([7929ef4](https://github.com/MustardSeedNetworks/seed/commit/7929ef40f888e6cd95dd00346ad11b485b4c4516))
* **database:** add boolean CHECK + credentials_id FK hardening (ADR-0006, Phase 5b-4) ([#1480](https://github.com/MustardSeedNetworks/seed/issues/1480)) ([bf71ff8](https://github.com/MustardSeedNetworks/seed/commit/bf71ff8926a831a3d44c60d985b2ef96abdca5e3))
* **database:** convert goose baseline to STRICT tables (ADR-0006, Phase 5b-3) ([#1479](https://github.com/MustardSeedNetworks/seed/issues/1479)) ([d12a5fc](https://github.com/MustardSeedNetworks/seed/commit/d12a5fc3c11261f1dc267ebf67200c48df9bf3d1))
* **database:** decompose the discovery repository god-file by entity (WS-D) ([#1702](https://github.com/MustardSeedNetworks/seed/issues/1702)) ([57e1fbe](https://github.com/MustardSeedNetworks/seed/commit/57e1fbe40c031584e0d32cac9eb6c381416aa473))
* decompose four WS-D god-files by role (arp, users, snmp, engine) ([#1709](https://github.com/MustardSeedNetworks/seed/issues/1709)) ([cb8e937](https://github.com/MustardSeedNetworks/seed/commit/cb8e9375184d9e8881a1bb8465abcd020d19b12f))
* decompose seven more WS-D god-files by role (database, alerts, registry, netif, iperf installer) ([#1710](https://github.com/MustardSeedNetworks/seed/issues/1710)) ([ec72335](https://github.com/MustardSeedNetworks/seed/commit/ec723354cd0271bb704730784cce1556aa1c7ba3))
* **dhcp:** split lease-file parsing out of the dhcp god-file (WS-D) ([#1700](https://github.com/MustardSeedNetworks/seed/issues/1700)) ([d4c7680](https://github.com/MustardSeedNetworks/seed/commit/d4c76808c3e747ea8e94cc1a0726211d1acbe15e))
* **discovery:** camelcase the problem-detection api (no grandfathering) ([#1605](https://github.com/MustardSeedNetworks/seed/issues/1605)) ([5aff651](https://github.com/MustardSeedNetworks/seed/commit/5aff651975a10b85fec246dbf2a7820a0966ba16))
* **discovery:** decompose the fingerprint god-file by role (WS-D) ([#1708](https://github.com/MustardSeedNetworks/seed/issues/1708)) ([b90b0e2](https://github.com/MustardSeedNetworks/seed/commit/b90b0e26bed21434497c3edaea1a6bb77edef93e))
* **discovery:** decompose the traceroute god-file by protocol (WS-D) ([#1705](https://github.com/MustardSeedNetworks/seed/issues/1705)) ([05fccd7](https://github.com/MustardSeedNetworks/seed/commit/05fccd76ef4b15dcef67b17fe1bb4d481f0b5b9f))
* **discovery:** delete the pipeline orchestrator (P7 retirement R2) ([#1508](https://github.com/MustardSeedNetworks/seed/issues/1508)) ([4a15394](https://github.com/MustardSeedNetworks/seed/commit/4a1539463eb1ce0abc1795c9281a2ac0f062eeee))
* **discovery:** drive Engine collectors through ports (adr-0018) ([#1592](https://github.com/MustardSeedNetworks/seed/issues/1592)) ([9edde7b](https://github.com/MustardSeedNetworks/seed/commit/9edde7b6031041b66b59506e56cfe25870684934))
* **discovery:** express scan as stage ports (adr-0018, phase 6) ([#1564](https://github.com/MustardSeedNetworks/seed/issues/1564)) ([0757fde](https://github.com/MustardSeedNetworks/seed/commit/0757fdeeec4bcc78a0d63d94e65c18c8321ff1b5))
* **discovery:** extract name/identity resolution to resolve leaf (adr-0018) ([#1589](https://github.com/MustardSeedNetworks/seed/issues/1589)) ([a874afd](https://github.com/MustardSeedNetworks/seed/commit/a874afd779d066ecf507d0130db5d2f8da3e3eba))
* **discovery:** relocate bluetooth collector to enumerate stage (adr-0018) ([#1595](https://github.com/MustardSeedNetworks/seed/issues/1595)) ([4933b79](https://github.com/MustardSeedNetworks/seed/commit/4933b79351872eba417584471a1e8c6d73b37ddb))
* **discovery:** relocate internal/services/discovery -&gt; internal/discovery (Phase 6 S3) ([#1489](https://github.com/MustardSeedNetworks/seed/issues/1489)) ([bbe72a6](https://github.com/MustardSeedNetworks/seed/commit/bbe72a62dbb9c2992ee56ae03b1fd9a8f063dd2e))
* **discovery:** relocate port-scan leaf to fingerprint stage (adr-0018) ([#1587](https://github.com/MustardSeedNetworks/seed/issues/1587)) ([e598809](https://github.com/MustardSeedNetworks/seed/commit/e5988094e2e6458f276cda5bcb172a8a63ccacc7))
* **discovery:** relocate the vuln assessment stage into a subpackage (adr-0018) ([#1570](https://github.com/MustardSeedNetworks/seed/issues/1570)) ([cefe714](https://github.com/MustardSeedNetworks/seed/commit/cefe714738a892acc98d2319418dd2f1670583e9))
* **discovery:** relocate wi-fi collector to enumerate + camelcase the wi-fi api (adr-0018) ([#1596](https://github.com/MustardSeedNetworks/seed/issues/1596)) ([6d2f3d8](https://github.com/MustardSeedNetworks/seed/commit/6d2f3d8a83198da719bdd4d4f6b6339cb8ea3398))
* **discovery:** relocate wired+service collector to enumerate stage (adr-0018) ([#1600](https://github.com/MustardSeedNetworks/seed/issues/1600)) ([9139cbe](https://github.com/MustardSeedNetworks/seed/commit/9139cbe31b746e1cc01d445263cec29284bf8737))
* drop dead Module vestige (api.Modules→BackgroundComponents, reporting.Module→Service) ([#1450](https://github.com/MustardSeedNetworks/seed/issues/1450)) ([a8d0e6d](https://github.com/MustardSeedNetworks/seed/commit/a8d0e6d1c13b443bba616d2a97e54dfd86a91ccd))
* **e2e:** rename mockAuthenticated -&gt; skipSetupWizard ([#1149](https://github.com/MustardSeedNetworks/seed/issues/1149)) ([4189799](https://github.com/MustardSeedNetworks/seed/commit/4189799c06b085ea625a8b551b4ff6b7f12e1ede))
* **harvest:** extract ReportRepo port into adapters/store ring ([#1431](https://github.com/MustardSeedNetworks/seed/issues/1431)) ([62302ee](https://github.com/MustardSeedNetworks/seed/commit/62302ee474538f519a66e57b26e228e2310f2437))
* **harvest:** extract Schedule/Metrics/Export repos; module is persistence-free ([#1432](https://github.com/MustardSeedNetworks/seed/issues/1432)) ([aab9220](https://github.com/MustardSeedNetworks/seed/commit/aab9220a01b5901685fbdab9b4c26b10c144daef))
* **health:** delete the dead health_check_results read-path (ADR-0026) ([#1636](https://github.com/MustardSeedNetworks/seed/issues/1636)) ([ded0f2b](https://github.com/MustardSeedNetworks/seed/commit/ded0f2bd43ee077374a9bb5322a4f4413782bf37))
* **health:** make health surface persistence-free (WS-B) ([#1678](https://github.com/MustardSeedNetworks/seed/issues/1678)) ([3215b97](https://github.com/MustardSeedNetworks/seed/commit/3215b97f13640d680083445865add9d5b27fcd16))
* **iperf:** decompose the iperf god-file by role (WS-D) ([#1706](https://github.com/MustardSeedNetworks/seed/issues/1706)) ([62db623](https://github.com/MustardSeedNetworks/seed/commit/62db62346e008343e74d02b7849c1c82013b1aa8))
* **paths:** drop legacy config-name fallback and dead DetectLegacyConfig ([#1609](https://github.com/MustardSeedNetworks/seed/issues/1609)) ([7b0c6f9](https://github.com/MustardSeedNetworks/seed/commit/7b0c6f926416c2dd9ee054497d6ee64785b0c609))
* **polling:** relocate PollingTarget to domain pkg + orchestrator ports, make persistence-free (WS-B) ([#1675](https://github.com/MustardSeedNetworks/seed/issues/1675)) ([74c4162](https://github.com/MustardSeedNetworks/seed/commit/74c4162e637d459ad2be551b13e49e0489796b2c))
* **probe:** narrow persistence port to domain types (WS-B1) ([#1672](https://github.com/MustardSeedNetworks/seed/issues/1672)) ([e07d928](https://github.com/MustardSeedNetworks/seed/commit/e07d9285c75763bdd12bc59b3c59f10a06b6a4bb))
* **r4a:** consolidate wifi under internal/wifi (was internal/canopy) ([#1444](https://github.com/MustardSeedNetworks/seed/issues/1444)) ([fd1fe13](https://github.com/MustardSeedNetworks/seed/commit/fd1fe13b721d0d58ae797a1fe73a3522ef48a740))
* **r4a:** descriptive api service groupings (Sap/Canopy/Roots -&gt; Diagnostics/Wireless) ([52a8371](https://github.com/MustardSeedNetworks/seed/commit/52a8371826c37580fa79ef21cb3aae9becb4828f))
* **r4a:** group network diagnostics under internal/diagnostics ([#1445](https://github.com/MustardSeedNetworks/seed/issues/1445)) ([7cc3b15](https://github.com/MustardSeedNetworks/seed/commit/7cc3b15a9c0656d730c8fed544b43d283807acde))
* **r4a:** move guestaudit to internal/security (was services/shell) ([#1446](https://github.com/MustardSeedNetworks/seed/issues/1446)) ([94edf22](https://github.com/MustardSeedNetworks/seed/commit/94edf2241de30451c06fa47f90857ef8ad3f0f21))
* **r4a:** rename harvest module to reporting (internal) ([#1448](https://github.com/MustardSeedNetworks/seed/issues/1448)) ([3ad9d58](https://github.com/MustardSeedNetworks/seed/commit/3ad9d581ce7777a09d9247bde715ae70abf3023b))
* **reconcile:** delete the dead canopy module facade (R1) ([#1442](https://github.com/MustardSeedNetworks/seed/issues/1442)) ([55f8a27](https://github.com/MustardSeedNetworks/seed/commit/55f8a275e458112132811374a0f17756d252c481))
* **reconcile:** delete the dead sap module facade (R2) ([#1443](https://github.com/MustardSeedNetworks/seed/issues/1443)) ([b1895d4](https://github.com/MustardSeedNetworks/seed/commit/b1895d427ed1e467ac3638d3dee2c4e57496c55b))
* **reconcile:** delete the dead shell module facade (R1) ([#1441](https://github.com/MustardSeedNetworks/seed/issues/1441)) ([b75c884](https://github.com/MustardSeedNetworks/seed/commit/b75c8847fded262f01755850b95e827034a34445))
* rename botanical API routes + internal identifiers to meaningful names ([#1451](https://github.com/MustardSeedNetworks/seed/issues/1451)) ([bf1b061](https://github.com/MustardSeedNetworks/seed/commit/bf1b06120402e6db60df99ae1feda6296fdcfc56))
* **retention:** relocate rollup SQL adapters to internal/database (WS-B5) ([#1677](https://github.com/MustardSeedNetworks/seed/issues/1677)) ([7370c51](https://github.com/MustardSeedNetworks/seed/commit/7370c515dfd01046fd9c666a4a9a2c73c1edca34))
* **roots:** delete the production-dead roots module skeleton ([#1439](https://github.com/MustardSeedNetworks/seed/issues/1439)) ([47a8952](https://github.com/MustardSeedNetworks/seed/commit/47a8952ba6e83f66b1316158b9c033cd23132905))
* **roots:** drop dead subpackages and unused db; module persistence-free ([#1438](https://github.com/MustardSeedNetworks/seed/issues/1438)) ([f3195b0](https://github.com/MustardSeedNetworks/seed/commit/f3195b0993d0c6e4b3706c2397b15f2ffcf7dee7))
* **roots:** relocate internal/pipeline to internal/modules/roots ([#1435](https://github.com/MustardSeedNetworks/seed/issues/1435)) ([307d447](https://github.com/MustardSeedNetworks/seed/commit/307d447ca31e6a14995fedaa2a3007807ca204a2))
* **settings:** decompose the management god-file by role (WS-D) ([#1699](https://github.com/MustardSeedNetworks/seed/issues/1699)) ([54b3399](https://github.com/MustardSeedNetworks/seed/commit/54b33993b3a6553c525855a26abce9fe2474cd78))
* **snmp:** decompose the interface god-file by query area (WS-D) ([#1701](https://github.com/MustardSeedNetworks/seed/issues/1701)) ([cdbbc18](https://github.com/MustardSeedNetworks/seed/commit/cdbbc1828be47086d87f569761a7397402ec273d))
* **survey:** decompose the report god-file by role (WS-D) ([#1703](https://github.com/MustardSeedNetworks/seed/issues/1703)) ([b19db25](https://github.com/MustardSeedNetworks/seed/commit/b19db25c085437971690df31a103daa5f1869926))
* **survey:** decompose the survey manager god-file by role (WS-D) ([#1704](https://github.com/MustardSeedNetworks/seed/issues/1704)) ([c06374a](https://github.com/MustardSeedNetworks/seed/commit/c06374a2641f881582e3245220dba2df66b9aaf6))
* **topology:** relocate row types to domain pkgs, make persistence-free (WS-B) ([#1673](https://github.com/MustardSeedNetworks/seed/issues/1673)) ([e768fcc](https://github.com/MustardSeedNetworks/seed/commit/e768fcc7b884fd387b87faf46a799ca4740f7256))
* **ui:** a11y brand-fill text, tokenize inset shadow, guard rgb/hsl ([#1258](https://github.com/MustardSeedNetworks/seed/issues/1258)) ([e63d1ff](https://github.com/MustardSeedNetworks/seed/commit/e63d1ff9e9a44bd3937d1ddecd7dfba6b9de8345))
* **ui:** decompose app.tsx god component into AppShell + useAppOrchestration (b1) ([#1578](https://github.com/MustardSeedNetworks/seed/issues/1578)) ([c58408d](https://github.com/MustardSeedNetworks/seed/commit/c58408d817a8cd8d2496b57e4159dc6b3a9fd58c))
* **ui:** function-first sidebar nav IA, retire botanical metaphor (M1, [#1452](https://github.com/MustardSeedNetworks/seed/issues/1452)) ([#1681](https://github.com/MustardSeedNetworks/seed/issues/1681)) ([546d937](https://github.com/MustardSeedNetworks/seed/commit/546d93724dd5f26a7ba984b32a0923bc61fe1ff2))
* **ui:** isolate T568B wire palette; tokenize stray headings + z-index ([#1250](https://github.com/MustardSeedNetworks/seed/issues/1250)) ([624ed74](https://github.com/MustardSeedNetworks/seed/commit/624ed74473879e7f89a88320acad3ed8b780d19e))
* **ui:** migrate component palette leaks to semantic/log/cat tokens ([#1248](https://github.com/MustardSeedNetworks/seed/issues/1248)) ([9599127](https://github.com/MustardSeedNetworks/seed/commit/9599127a9fae97777540817ee6fde32678889aef))
* **ui:** remove dead class-helpers; move design-system colors onto tokens ([#1247](https://github.com/MustardSeedNetworks/seed/issues/1247)) ([25a9b11](https://github.com/MustardSeedNetworks/seed/commit/25a9b11a4e4d8ab1aafe099b75703e008e1c169d))
* **ui:** remove the dead pipeline UI now that discovery rides jobs (P7 S3.3) ([#1506](https://github.com/MustardSeedNetworks/seed/issues/1506)) ([21e90e7](https://github.com/MustardSeedNetworks/seed/commit/21e90e79b3a5e0e56c2f52a8338c9620143068f1))
* **ui:** replace HeaderBar inline SVGs with lucide icons (m2) ([#1582](https://github.com/MustardSeedNetworks/seed/issues/1582)) ([3932184](https://github.com/MustardSeedNetworks/seed/commit/39321848d97bb60858b001c31b05e7a4e19e2f67))
* **ui:** replace window event test-orchestration bus with a zustand store ([#1574](https://github.com/MustardSeedNetworks/seed/issues/1574)) ([ea5b59e](https://github.com/MustardSeedNetworks/seed/commit/ea5b59e7a6682434691c50dde92773f784c80ff7))
* **ui:** single canonical SeedLogo brand mark (m3) ([#1580](https://github.com/MustardSeedNetworks/seed/issues/1580)) ([90663a6](https://github.com/MustardSeedNetworks/seed/commit/90663a68e3e35ef516aa645ca25ad61d963cfb46))
* **ui:** slim HeaderBar — kill sidebar duplicates, logout to profile menu (Phase 2) ([#1227](https://github.com/MustardSeedNetworks/seed/issues/1227)) ([8dfd648](https://github.com/MustardSeedNetworks/seed/commit/8dfd64865592d9fa3d38b018ac6d78b1136759da))
* **ui:** tokenize all design primitives (Phase 0 harmonization) ([#1221](https://github.com/MustardSeedNetworks/seed/issues/1221)) ([24fb448](https://github.com/MustardSeedNetworks/seed/commit/24fb4488a9111da49400bffd41f849c70115e9cd))
* **ui:** write heatmap legend gradient in hex, not rgb ([#1260](https://github.com/MustardSeedNetworks/seed/issues/1260)) ([f873261](https://github.com/MustardSeedNetworks/seed/commit/f87326152abc5e3effb8ff0faa18c7e078726494))
* **vuln:** decompose the scanner god-file by role (WS-D) ([#1707](https://github.com/MustardSeedNetworks/seed/issues/1707)) ([2701fef](https://github.com/MustardSeedNetworks/seed/commit/2701fefb81f63ea051e664f0688dda0dc5ecc2f9))


### Documentation

* **adr-0010:** revise to pure boundary mapping — wire is 100% camelCase, no exceptions ([#1683](https://github.com/MustardSeedNetworks/seed/issues/1683)) ([0259e3d](https://github.com/MustardSeedNetworks/seed/commit/0259e3d39c94e60887446307f33487e02fc6cde1))
* **adr:** add discovery pipeline stage split design (adr-0018, phase 6) ([#1563](https://github.com/MustardSeedNetworks/seed/issues/1563)) ([9a7de5a](https://github.com/MustardSeedNetworks/seed/commit/9a7de5ab0878b2f0077ee65ed78d8afd141ac92f))
* **adr:** ADR-0021 persist anomaly engine in SQL + converge all sources ([#1619](https://github.com/MustardSeedNetworks/seed/issues/1619)) ([d2d367a](https://github.com/MustardSeedNetworks/seed/commit/d2d367a67f0e95302d593ad65e0da8b4c3f7628f))
* **adr:** bluetooth live-scan capture port (ADR-0013) ([#1522](https://github.com/MustardSeedNetworks/seed/issues/1522)) ([1554100](https://github.com/MustardSeedNetworks/seed/commit/15541009de31d15ab7f5e6ec1babddf73d374b41))
* **adr:** broaden ADR-0010 into the full naming standard (file/dir audit) ([#1515](https://github.com/MustardSeedNetworks/seed/issues/1515)) ([899ac20](https://github.com/MustardSeedNetworks/seed/commit/899ac2060f226c280171de1fb61890713ecba259))
* **adr:** config validation schema is a constraints validator (ADR-0014) ([#1525](https://github.com/MustardSeedNetworks/seed/issues/1525)) ([0f6c99b](https://github.com/MustardSeedNetworks/seed/commit/0f6c99b43a9fc27849c6fd1e364cae1cfb2142a1))
* **adr:** design daily rollups for the anomaly store (ADR-0028) ([#1649](https://github.com/MustardSeedNetworks/seed/issues/1649)) ([1150b81](https://github.com/MustardSeedNetworks/seed/commit/1150b8144ff471aa7534648dbf8172c3ba5baea6))
* **adr:** establish identifier casing conventions (P8.0) ([#1513](https://github.com/MustardSeedNetworks/seed/issues/1513)) ([dc96fdb](https://github.com/MustardSeedNetworks/seed/commit/dc96fdbc942abb48c11798ffd54890da82848a74))
* **adr:** hygiene pass — honest statuses + two as-built ADRs ([#1622](https://github.com/MustardSeedNetworks/seed/issues/1622)) ([db669f7](https://github.com/MustardSeedNetworks/seed/commit/db669f7615fe7082a8d2b9de875475178d8ee492))
* **adr:** network anomaly engine + Wi-Fi monitor capture/decode ([#1521](https://github.com/MustardSeedNetworks/seed/issues/1521)) ([995c65e](https://github.com/MustardSeedNetworks/seed/commit/995c65edf79bde4bf39cabaa852bcd2d74616a3e))
* **adr:** probe is the active-monitoring anomaly source (ADR-0025) ([#1634](https://github.com/MustardSeedNetworks/seed/issues/1634)) ([58370b8](https://github.com/MustardSeedNetworks/seed/commit/58370b8128df232344cd26beb435f06645160c9e))
* **adr:** record discovery orchestrator convergence, defer fold+schema to P7 (Phase 6 S4) ([#1490](https://github.com/MustardSeedNetworks/seed/issues/1490)) ([f7aaba8](https://github.com/MustardSeedNetworks/seed/commit/f7aaba83d3a81d57c4a73a986244bdf564112e35))
* **adr:** record outbox deferral with a trigger; mark jobs durability done (ADR-0004/0005) ([#1486](https://github.com/MustardSeedNetworks/seed/issues/1486)) ([55fb3ef](https://github.com/MustardSeedNetworks/seed/commit/55fb3ef8bc7a6347d2f15c7c63b1249c52c62e34))
* **adr:** record that profile/settings UI types are a curated view (P7 S6 close) ([#1511](https://github.com/MustardSeedNetworks/seed/issues/1511)) ([a9f6a04](https://github.com/MustardSeedNetworks/seed/commit/a9f6a040273de0d34a7d6c1a5268f497643957d3))
* **adr:** record the fingerprint-stage relocation design (adr-0018, phase 6) ([#1571](https://github.com/MustardSeedNetworks/seed/issues/1571)) ([53b5884](https://github.com/MustardSeedNetworks/seed/commit/53b58841d43b3bed3c80e919e77b4154dda16052))
* **adr:** scope migrating health-checks onto probe, then renaming (ADR-0027) ([#1640](https://github.com/MustardSeedNetworks/seed/issues/1640)) ([a7da0da](https://github.com/MustardSeedNetworks/seed/commit/a7da0da7a4522dd7b37e2f18c41a754b3c57026b))
* **adr:** separate credential data-encryption key from auth jwt secret (ADR-0015) ([#1527](https://github.com/MustardSeedNetworks/seed/issues/1527)) ([3bdde72](https://github.com/MustardSeedNetworks/seed/commit/3bdde726e3109f467b866b883baf165d7c293831))
* align makefile local build scope ([bdb4875](https://github.com/MustardSeedNetworks/seed/commit/bdb4875218731c32c26a8cea0388a51291966cc7))
* **arch:** correct Phase 5 persistence status (5b schema work is done) ([#1558](https://github.com/MustardSeedNetworks/seed/issues/1558)) ([723d470](https://github.com/MustardSeedNetworks/seed/commit/723d470b6c9914a2f4739129e44ca93ccb4cc1ad))
* **architecture:** add generated route inventory (183 routes) ([#1403](https://github.com/MustardSeedNetworks/seed/issues/1403)) ([2718a78](https://github.com/MustardSeedNetworks/seed/commit/2718a781539964298d754e830492fcd5cf962e57))
* **architecture:** add Phase 3 hexagon extraction plan (harvest pilot) ([#1426](https://github.com/MustardSeedNetworks/seed/issues/1426)) ([966e20d](https://github.com/MustardSeedNetworks/seed/commit/966e20d10d92c2f384e5a06c43d89200d423b770))
* **architecture:** add re-architecture blueprint and ADRs 0001-0006 ([#1401](https://github.com/MustardSeedNetworks/seed/issues/1401)) ([3268b1f](https://github.com/MustardSeedNetworks/seed/commit/3268b1f6d512d42d4d64d8efdb5eca734fda7b21))
* **architecture:** add the architecture completion plan (no-shortcuts strangle finish) ([#1654](https://github.com/MustardSeedNetworks/seed/issues/1654)) ([4b189f1](https://github.com/MustardSeedNetworks/seed/commit/4b189f12644392cad33446086d1854d55eed326a))
* **architecture:** reconcile ADR-0005 + blueprint §8 to implemented jobs state ([#1475](https://github.com/MustardSeedNetworks/seed/issues/1475)) ([5ff8d9d](https://github.com/MustardSeedNetworks/seed/commit/5ff8d9d144b13fae0add3bf5e65a2237aa092ace))
* **architecture:** record Phase 3 harvest status + ReportRepo execution recipe ([#1430](https://github.com/MustardSeedNetworks/seed/issues/1430)) ([56775bb](https://github.com/MustardSeedNetworks/seed/commit/56775bb2b74efe2d592c183caf3ffbca890e5863))
* **arch:** mark harvest→reporting done in reconcile §0 ([#1448](https://github.com/MustardSeedNetworks/seed/issues/1448)) ([#1449](https://github.com/MustardSeedNetworks/seed/issues/1449)) ([7c30658](https://github.com/MustardSeedNetworks/seed/commit/7c30658829bbb0a3b4de60464a0c9d6249185205))
* **arch:** phase 3 reconcile proposal — one composition root + descriptive names ([#1440](https://github.com/MustardSeedNetworks/seed/issues/1440)) ([738288c](https://github.com/MustardSeedNetworks/seed/commit/738288ce1dc82e673408c3b4622f767131c1efa3))
* **arch:** record R4b code-vs-brand split ([#1450](https://github.com/MustardSeedNetworks/seed/issues/1450), [#1451](https://github.com/MustardSeedNetworks/seed/issues/1451)) + defer Phase 3.x IA redesign ([#1452](https://github.com/MustardSeedNetworks/seed/issues/1452)) ([5aecbc9](https://github.com/MustardSeedNetworks/seed/commit/5aecbc9da489e13ed7766db6ddbaa158c47acc46))
* **arch:** record reconcile execution status (checkpoint) ([#1447](https://github.com/MustardSeedNetworks/seed/issues/1447)) ([e26385d](https://github.com/MustardSeedNetworks/seed/commit/e26385d26959e20b587a17aba63d57d044f7dd2a))
* **phase3:** mark harvest pilot complete — §7 docs gate closed ([#1434](https://github.com/MustardSeedNetworks/seed/issues/1434)) ([458ae8b](https://github.com/MustardSeedNetworks/seed/commit/458ae8b37af7bd5c5eaa4dba53b767e7cc4b85e8))
* reconcile stale botanical routes + ring paths after the rename ([#1453](https://github.com/MustardSeedNetworks/seed/issues/1453)) ([16d22d3](https://github.com/MustardSeedNetworks/seed/commit/16d22d3e97c781cc33bc2a75a60910ec1669c7bb))
* **ui:** add STATE.md state-placement guide (l2) ([#1583](https://github.com/MustardSeedNetworks/seed/issues/1583)) ([02f995c](https://github.com/MustardSeedNetworks/seed/commit/02f995c5b0ad82ea767ffce85abc684099ad863d))
* **ws-b:** close B3 — identity is the documented ADR-0024 exception ([#1679](https://github.com/MustardSeedNetworks/seed/issues/1679)) ([2bd9ab6](https://github.com/MustardSeedNetworks/seed/commit/2bd9ab6ccde0436c13ec38c172e5f3525723cb9f))


### Tests

* **api:** add golden HTTP characterization harness (public routes) ([#1402](https://github.com/MustardSeedNetworks/seed/issues/1402)) ([a9a50c2](https://github.com/MustardSeedNetworks/seed/commit/a9a50c2a61dd0cc8d56ab116e17037ef9f240bb8))
* **api:** full-chain auth-boundary golden harness (Phase 0) ([#1405](https://github.com/MustardSeedNetworks/seed/issues/1405)) ([ec8d40c](https://github.com/MustardSeedNetworks/seed/commit/ec8d40cf2a56df11d2b337bc0993ad647be3f42b))
* **auth:** lock the first-run no-usable-default-password invariant ([#1242](https://github.com/MustardSeedNetworks/seed/issues/1242)) ([#1585](https://github.com/MustardSeedNetworks/seed/issues/1585)) ([322c902](https://github.com/MustardSeedNetworks/seed/commit/322c902a9ea9877b7865ca2cfdac926665d6ec2b))
* **auth:** pin the CSRF exempt-list with a coverage gate ([#1223](https://github.com/MustardSeedNetworks/seed/issues/1223)) ([#1584](https://github.com/MustardSeedNetworks/seed/issues/1584)) ([ddb05de](https://github.com/MustardSeedNetworks/seed/commit/ddb05dece69e7e5c63a0fc7ef6ea6282412eb19c))
* **config:** cover the profile save-&gt;apply round-trip (close ADR-0009 [#1](https://github.com/MustardSeedNetworks/seed/issues/1)) ([#1512](https://github.com/MustardSeedNetworks/seed/issues/1512)) ([39664c2](https://github.com/MustardSeedNetworks/seed/commit/39664c270df34236e669b4d3af7ecda8d4b7bedd))
* **contract:** enforce the non-recursive transport policy in CI ([#1423](https://github.com/MustardSeedNetworks/seed/issues/1423)) ([8a74a5e](https://github.com/MustardSeedNetworks/seed/commit/8a74a5ed2542c51616c0dc5b5f34afd4855d055b))
* **contract:** round-trip guardrail — covered DTOs validate against their schema ([#1417](https://github.com/MustardSeedNetworks/seed/issues/1417)) ([869de92](https://github.com/MustardSeedNetworks/seed/commit/869de920fa71458eae1560abdaf724e34d9fa4c5))
* **database:** add schema-snapshot regression gate (ADR-0006, Phase 5a) ([#1476](https://github.com/MustardSeedNetworks/seed/issues/1476)) ([48cff18](https://github.com/MustardSeedNetworks/seed/commit/48cff18f086be1985040fed998900eea800bba70))
* **e2e:** 4 coverage-gaps testids - profile/logs/discovery modals ([#1311](https://github.com/MustardSeedNetworks/seed/issues/1311)) ([0887695](https://github.com/MustardSeedNetworks/seed/commit/0887695b139b189bffdafcf6ce5cc87edcd5d872))
* **e2e:** appearance-settings-section testid, migrate 3 sites ([#1303](https://github.com/MustardSeedNetworks/seed/issues/1303)) ([3076866](https://github.com/MustardSeedNetworks/seed/commit/307686653526ef683ea936902705bac38786262a))
* **e2e:** collapse 3 OR-fallback empty-state regexes + drop auto-save junk ([#1313](https://github.com/MustardSeedNetworks/seed/issues/1313)) ([d00da8a](https://github.com/MustardSeedNetworks/seed/commit/d00da8accc50260e71eb441606c3b8ccef1a6a59))
* **e2e:** delete 21 advisory-tier-perma-failing tests across 9 files ([#1318](https://github.com/MustardSeedNetworks/seed/issues/1318)) ([b04ba10](https://github.com/MustardSeedNetworks/seed/commit/b04ba10813dfaf963235d8a21dc2ae020627fd10))
* **e2e:** delete 3 remaining failing tests - threshold precondition + 2 logout ([#1319](https://github.com/MustardSeedNetworks/seed/issues/1319)) ([370b056](https://github.com/MustardSeedNetworks/seed/commit/370b0561d54fca99382070fca864751de09187cf))
* **e2e:** delete 5 gated error-scenarios tests, migrate retry to login-submit ([#1312](https://github.com/MustardSeedNetworks/seed/issues/1312)) ([a035841](https://github.com/MustardSeedNetworks/seed/commit/a03584161dce36b78feacc12247f74b73aa0029d))
* **e2e:** delete 8 junk tests from error-scenarios.spec.ts ([#1294](https://github.com/MustardSeedNetworks/seed/issues/1294)) ([8784fe3](https://github.com/MustardSeedNetworks/seed/commit/8784fe33051d8a02b3e96353465614a18d3c1bb4))
* **e2e:** delete coverage-gaps.spec.ts - tests non-existent UI ([#1314](https://github.com/MustardSeedNetworks/seed/issues/1314)) ([99e8a0b](https://github.com/MustardSeedNetworks/seed/commit/99e8a0bb434aa642b44ac41cc1613f4aa14ea89e))
* **e2e:** delete responsive.spec.ts wholesale ([#1296](https://github.com/MustardSeedNetworks/seed/issues/1296)) ([7ee0de8](https://github.com/MustardSeedNetworks/seed/commit/7ee0de8645c70519e903f574790156b270d8bae9))
* **e2e:** delete setup-wizard.spec.ts wholesale ([#1305](https://github.com/MustardSeedNetworks/seed/issues/1305)) ([7513300](https://github.com/MustardSeedNetworks/seed/commit/7513300823bd3edaf767665fe1e2a78c29873cb2))
* **e2e:** drop tautological scroll test + finish responsive [class*=card] migration ([#1292](https://github.com/MustardSeedNetworks/seed/issues/1292)) ([cf047ca](https://github.com/MustardSeedNetworks/seed/commit/cf047cae204ca0e368df1a1a6a1bd829080822fd))
* **e2e:** explicit toBeVisible wait between profile and logout click ([#1288](https://github.com/MustardSeedNetworks/seed/issues/1288)) ([bd8eec6](https://github.com/MustardSeedNetworks/seed/commit/bd8eec6f8148eb6ca1f45c89270989b482f8ba19))
* **e2e:** fix error-scenarios stale mock prefixes + tighten cross-browser assertion ([#1289](https://github.com/MustardSeedNetworks/seed/issues/1289)) ([3d62c8f](https://github.com/MustardSeedNetworks/seed/commit/3d62c8f1bf2265bd6f492ed7e29bea65e1184b99))
* **e2e:** fix rotating post-login dashboard flake in auth-complete ([#1588](https://github.com/MustardSeedNetworks/seed/issues/1588)) ([d7689b9](https://github.com/MustardSeedNetworks/seed/commit/d7689b93b0d0b65eb28e8ea5742578c88571449a))
* **e2e:** gateway-* + page-header-help-* testids, rewrite gateway.spec.ts ([#1304](https://github.com/MustardSeedNetworks/seed/issues/1304)) ([e3e4f97](https://github.com/MustardSeedNetworks/seed/commit/e3e4f97302c0106dd0772632993e99bc2f2e7093))
* **e2e:** login-submit testid, migrate 24 sign-in regex sites ([#1306](https://github.com/MustardSeedNetworks/seed/issues/1306)) ([5f78f69](https://github.com/MustardSeedNetworks/seed/commit/5f78f692dbce9b6077ef2d535aa81b963660557f))
* **e2e:** migrate 16 fragile getByText regexes to role=alert ([#1300](https://github.com/MustardSeedNetworks/seed/issues/1300)) ([45c4ace](https://github.com/MustardSeedNetworks/seed/commit/45c4ace1065e6d9a2f425f926ee3f2e1ff1b1a28))
* **e2e:** migrate 6 settings.spec.ts button-regex sites to testids ([#1310](https://github.com/MustardSeedNetworks/seed/issues/1310)) ([9e5b7a3](https://github.com/MustardSeedNetworks/seed/commit/9e5b7a37653cad38671a3042b31028506fb26c56))
* **e2e:** migrate seed auth specs from text-regex to role=alert ([#1301](https://github.com/MustardSeedNetworks/seed/issues/1301)) ([46f4409](https://github.com/MustardSeedNetworks/seed/commit/46f4409f16bbeb8694abef1184de00fde3b98cea))
* **e2e:** mock /api/auth/logout in Logout describe (real root cause) ([#1291](https://github.com/MustardSeedNetworks/seed/issues/1291)) ([b107b1b](https://github.com/MustardSeedNetworks/seed/commit/b107b1bd49fc534bc82404ff73508ef77e2abb1b))
* **e2e:** open profile dropdown before clicking header-logout ([#1285](https://github.com/MustardSeedNetworks/seed/issues/1285)) ([c64c807](https://github.com/MustardSeedNetworks/seed/commit/c64c80738a4b901f0ffef7e03c852f3aafee9c33))
* **e2e:** page-header-description testid, migrate 7 per-page specs ([#1302](https://github.com/MustardSeedNetworks/seed/issues/1302)) ([b53e033](https://github.com/MustardSeedNetworks/seed/commit/b53e033116d61c034f39f9cf970ec26582e7d6d1))
* **e2e:** replace all test.skip with loud expect-failures ([#1147](https://github.com/MustardSeedNetworks/seed/issues/1147)) ([937381e](https://github.com/MustardSeedNetworks/seed/commit/937381e7d2cf06be929bfc8e45704a94c7637b3a))
* **e2e:** Replace fragile .or() selector chains with stable data-testids ([#1151](https://github.com/MustardSeedNetworks/seed/issues/1151)) ([eb1e95f](https://github.com/MustardSeedNetworks/seed/commit/eb1e95f7b07671ab2048566d7c8fd13383bd5bef))
* **e2e:** rewrite smoke.spec.ts as a golden-path required-status gate ([#1295](https://github.com/MustardSeedNetworks/seed/issues/1295)) ([ab1dd08](https://github.com/MustardSeedNetworks/seed/commit/ab1dd08be31f17149687d2728e37326f1e7d54a9))
* **e2e:** stable selectors on dashboard + responsive specs ([#1287](https://github.com/MustardSeedNetworks/seed/issues/1287)) ([350426a](https://github.com/MustardSeedNetworks/seed/commit/350426a584a3ddce20dff491f8a04bb99d3e029b))
* **e2e:** testid on Thresholds/Discovery/DNS/Performance settings sections ([#1284](https://github.com/MustardSeedNetworks/seed/issues/1284)) ([fdb486b](https://github.com/MustardSeedNetworks/seed/commit/fdb486b084c5cd8cda6837996464e05849fd7707))
* **e2e:** tighten help-drawer backdrop test with stable testid ([#1299](https://github.com/MustardSeedNetworks/seed/issues/1299)) ([a906e39](https://github.com/MustardSeedNetworks/seed/commit/a906e397791618a5d6cfdd1c3f21d32b5fd82321))
* **e2e:** use storageState in auth-complete Logout describe ([cefd4a0](https://github.com/MustardSeedNetworks/seed/commit/cefd4a0237ddf6c6d8f618d43697abd9cb53c248))
* **e2e:** use storageState in auth-complete Logout describe ([#1283](https://github.com/MustardSeedNetworks/seed/issues/1283)) ([64730d3](https://github.com/MustardSeedNetworks/seed/commit/64730d321c5a18b0209391eb4078a14cf6b77d59))
* **e2e:** wifi-wired-fallback testid + replace fragile wifi-page selectors ([#1290](https://github.com/MustardSeedNetworks/seed/issues/1290)) ([c4333b8](https://github.com/MustardSeedNetworks/seed/commit/c4333b8a62f32bca53cc5e62c755be05e02dce82))
* **i18n:** add ES layout sanity e2e for overflow detection ([#1217](https://github.com/MustardSeedNetworks/seed/issues/1217)) ([35b8430](https://github.com/MustardSeedNetworks/seed/commit/35b843059cbad66e7dbc133726f71b05d1389257))
* **i18n:** port phase 6 language-switch e2e spec to seed ([#1215](https://github.com/MustardSeedNetworks/seed/issues/1215)) ([3a60a01](https://github.com/MustardSeedNetworks/seed/commit/3a60a015195e8b685f0dd6bb1642bd5ff905e8ab))


### Continuous Integration

* add JSON wire-casing ratchet gate (P8.1, ADR-0010) ([#1514](https://github.com/MustardSeedNetworks/seed/issues/1514)) ([52e6b4c](https://github.com/MustardSeedNetworks/seed/commit/52e6b4cd01c11b01f803ad8e595cd1b65f07ed87))
* block undefined design tokens in token-discipline guard ([#1264](https://github.com/MustardSeedNetworks/seed/issues/1264)) ([eadf33d](https://github.com/MustardSeedNetworks/seed/commit/eadf33dc03cf6fee8f04a759d9173f7e2086a947))
* document the gosec gate posture (gate=golangci-lint, standalone=SARIF only) ([#1548](https://github.com/MustardSeedNetworks/seed/issues/1548)) ([d824237](https://github.com/MustardSeedNetworks/seed/commit/d8242378d9013ec7d5d02fe42a51035caa0c0cf4))
* **e2e:** adopt Chromium + WebKit per E2E_CONVENTIONS ([#1148](https://github.com/MustardSeedNetworks/seed/issues/1148)) ([10ec72a](https://github.com/MustardSeedNetworks/seed/commit/10ec72a0f353de8e1b54ba79035d313adb71f2a2))
* **e2e:** bump workers to 4 + shard suite ×4 per browser (8 parallel jobs) ([#1286](https://github.com/MustardSeedNetworks/seed/issues/1286)) ([34b346e](https://github.com/MustardSeedNetworks/seed/commit/34b346e06650481867fa869c50a2def014a55dbb))
* **e2e:** finish multi-browser rollout for full e2e job ([#1180](https://github.com/MustardSeedNetworks/seed/issues/1180)) ([aca73a2](https://github.com/MustardSeedNetworks/seed/commit/aca73a25cef19d194bcea16e86556aaa8dab694f))
* **e2e:** gate ignoreHTTPSErrors to !CI per E2E_CONVENTIONS ([#1150](https://github.com/MustardSeedNetworks/seed/issues/1150)) ([3bccc55](https://github.com/MustardSeedNetworks/seed/commit/3bccc5528ad50ae4cc883689057f80845e6a10ab))
* **e2e:** promote full E2E suite to a blocking gate via CI Complete ([#1590](https://github.com/MustardSeedNetworks/seed/issues/1590)) ([1df9974](https://github.com/MustardSeedNetworks/seed/commit/1df9974ad352aa496f00f967f65b043dcc2b2601))
* **e2e:** rename matrix key project -&gt; browser for clarity ([#1238](https://github.com/MustardSeedNetworks/seed/issues/1238)) ([0c2a1bc](https://github.com/MustardSeedNetworks/seed/commit/0c2a1bc00cec7830223a42dcd9e7cde8fe423f58))
* **e2e:** shard browser tests by project; mark advisory tier non-blocking ([#1232](https://github.com/MustardSeedNetworks/seed/issues/1232)) ([b5f0bd6](https://github.com/MustardSeedNetworks/seed/commit/b5f0bd65c11b3428e2ae24c5efb9daba97ee7223))
* **governance:** exempt release-please PRs from human PR-body template ([#1741](https://github.com/MustardSeedNetworks/seed/issues/1741)) ([e083074](https://github.com/MustardSeedNetworks/seed/commit/e083074a5012085b678f451f8fa5a8c55dfbf5d0))
* make the frontend TypeScript check blocking ([#1547](https://github.com/MustardSeedNetworks/seed/issues/1547)) ([6081652](https://github.com/MustardSeedNetworks/seed/commit/6081652a6ba28a2316941c6e840b32c1be325636))
* **perf:** build UI once, add job timeouts, least-privilege ([#1733](https://github.com/MustardSeedNetworks/seed/issues/1733)) ([02e2a17](https://github.com/MustardSeedNetworks/seed/commit/02e2a17bc8015357b8a61ad16e8242ca17c8526f))
* **release-please:** honor pre-major bump config; force next release to 0.211.0 ([#1603](https://github.com/MustardSeedNetworks/seed/issues/1603)) ([d5a784e](https://github.com/MustardSeedNetworks/seed/commit/d5a784e68eeebf787004defecdf77c328249dcdc))
* **release:** bump goreleaser-cross + install arm64 libpcap from ubuntu-ports ([#1577](https://github.com/MustardSeedNetworks/seed/issues/1577)) ([2211195](https://github.com/MustardSeedNetworks/seed/commit/2211195f8a4893a39b3086745027f2df1a325b3f))
* **release:** drop macOS amd64 (Intel) build target ([#1579](https://github.com/MustardSeedNetworks/seed/issues/1579)) ([9eaa9d4](https://github.com/MustardSeedNetworks/seed/commit/9eaa9d474dbb04bd7d185ae72fc46a32acfa4c50))
* run gitleaks via the CLI instead of the licensed action ([#1552](https://github.com/MustardSeedNetworks/seed/issues/1552)) ([ce9b039](https://github.com/MustardSeedNetworks/seed/commit/ce9b039a80dce3fb03f57e7b1f63ce38d96e6028))
* **security:** add Semgrep SAST gate, pin curl|sh install ([#1737](https://github.com/MustardSeedNetworks/seed/issues/1737)) ([ecbc6ec](https://github.com/MustardSeedNetworks/seed/commit/ecbc6ec12d383d3e34fe9408f7d387d90308af9b)), closes [#1736](https://github.com/MustardSeedNetworks/seed/issues/1736)
* **setup-go:** persist Go build+module cache; document CGO strategy ([#1433](https://github.com/MustardSeedNetworks/seed/issues/1433)) ([f3cfa77](https://github.com/MustardSeedNetworks/seed/commit/f3cfa778ce555fc04169366c9ae2112b936fab1e))


### Miscellaneous

* **build:** remove Docker/GHCR publishing ([#1729](https://github.com/MustardSeedNetworks/seed/issues/1729)) ([513e1e0](https://github.com/MustardSeedNetworks/seed/commit/513e1e01d4291e13fa4e32231371644a26778e2e))
* **ci:** add filename-policy gate for decomposed packages ([#1617](https://github.com/MustardSeedNetworks/seed/issues/1617)) ([cd8642c](https://github.com/MustardSeedNetworks/seed/commit/cd8642ccea29b2324573d43447f2a2a2d8b8772b))
* **cleanup:** remove dead HTTP-redirector remnants ([#1621](https://github.com/MustardSeedNetworks/seed/issues/1621)) ([1fc1275](https://github.com/MustardSeedNetworks/seed/commit/1fc12751a4c548be59b7f21bc20708f31f95f22b))
* **deps:** always-latest toolchain sweep (lockstep with stem) ([#1687](https://github.com/MustardSeedNetworks/seed/issues/1687)) ([60b54c4](https://github.com/MustardSeedNetworks/seed/commit/60b54c415ab2a120a5c39a98b0a2838859086868))
* **deps:** bump @biomejs/biome 2.4.15 -&gt; 2.4.16 ([#1202](https://github.com/MustardSeedNetworks/seed/issues/1202)) ([2947044](https://github.com/MustardSeedNetworks/seed/commit/29470444910dbbebcbf07536cf85b7d605f91909))
* **deps:** bump Go to 1.26.4 (security) ([#1469](https://github.com/MustardSeedNetworks/seed/issues/1469)) ([d4d3c11](https://github.com/MustardSeedNetworks/seed/commit/d4d3c111ceb52497bfca1b94df8cbb0c5cc23eb6))
* **deps:** bump i18next 26.2.0 -&gt; 26.3.0 per always-latest policy ([#1187](https://github.com/MustardSeedNetworks/seed/issues/1187)) ([9acf789](https://github.com/MustardSeedNetworks/seed/commit/9acf78943ec8c28fc69ef6db83e97a5da6789bf5))
* **deps:** refresh go module graph ([#1690](https://github.com/MustardSeedNetworks/seed/issues/1690)) ([82144b5](https://github.com/MustardSeedNetworks/seed/commit/82144b5e5c5c479289df6b759aaedb3119509b06))
* **deps:** update frontend deps to latest + clear esbuild advisory ([#1666](https://github.com/MustardSeedNetworks/seed/issues/1666)) ([9d669bf](https://github.com/MustardSeedNetworks/seed/commit/9d669bfbc22b448c26986835c8f4e05f29ea2408))
* **deps:** update Go modules to latest ([#1667](https://github.com/MustardSeedNetworks/seed/issues/1667)) ([2153916](https://github.com/MustardSeedNetworks/seed/commit/21539163bc5ea64f432991a9b2c2b0d3b96f3477))
* **e2e:** delete 17 no-value tests + 1 fixme per per-test eval ([#1185](https://github.com/MustardSeedNetworks/seed/issues/1185)) ([03c1298](https://github.com/MustardSeedNetworks/seed/commit/03c1298f2214513f4be8a18dd259db1d49431a38))
* **github:** standardize repo governance ([#1689](https://github.com/MustardSeedNetworks/seed/issues/1689)) ([f409599](https://github.com/MustardSeedNetworks/seed/commit/f40959915e63522e426b5c60236794122085977e))
* **i18n:** promote check-keys.py unused-key check from warn to fail ([#1218](https://github.com/MustardSeedNetworks/seed/issues/1218)) ([68deaaf](https://github.com/MustardSeedNetworks/seed/commit/68deaaffdb85fcc0a6f9683985e9ed8ae7c17124))
* **license:** add license-key-circumvention clause to BUSL Additional Use Grant ([#1735](https://github.com/MustardSeedNetworks/seed/issues/1735)) ([e961114](https://github.com/MustardSeedNetworks/seed/commit/e96111461834308534e4712e5a5b4dd1586e9d23))
* **main:** release 0.197.1 ([#1145](https://github.com/MustardSeedNetworks/seed/issues/1145)) ([25bc3f8](https://github.com/MustardSeedNetworks/seed/commit/25bc3f815edafa9b1fbc701839a5aa09a3af5524))
* **main:** release 0.198.0 ([620cb87](https://github.com/MustardSeedNetworks/seed/commit/620cb87bea912d495188c2b466762a20403e803d))
* **main:** release 0.198.0 ([cecbf51](https://github.com/MustardSeedNetworks/seed/commit/cecbf514b1f21c7dadb460b210218975fc6d699f))
* **main:** release 0.199.0 ([#1188](https://github.com/MustardSeedNetworks/seed/issues/1188)) ([27e22db](https://github.com/MustardSeedNetworks/seed/commit/27e22dbbbb4455318d389cfec95f594ebd48130a))
* **main:** release 0.200.0 ([#1220](https://github.com/MustardSeedNetworks/seed/issues/1220)) ([6d878e2](https://github.com/MustardSeedNetworks/seed/commit/6d878e2231126c786770a551b331825ba5fb93ec))
* **main:** release 0.201.0 ([#1237](https://github.com/MustardSeedNetworks/seed/issues/1237)) ([33e2dc8](https://github.com/MustardSeedNetworks/seed/commit/33e2dc85fb7d7c1c444f0d5d98c02c29bbec71b9))
* **main:** release 0.201.1 ([#1241](https://github.com/MustardSeedNetworks/seed/issues/1241)) ([28be347](https://github.com/MustardSeedNetworks/seed/commit/28be347a89d81b8c7a18a17fb83d44c21f9e6aa8))
* **main:** release 0.201.2 ([#1244](https://github.com/MustardSeedNetworks/seed/issues/1244)) ([f361fbe](https://github.com/MustardSeedNetworks/seed/commit/f361fbe49741fdc5ce29580b46f9bbe0f5e6a11a))
* **main:** release 0.202.0 ([#1252](https://github.com/MustardSeedNetworks/seed/issues/1252)) ([d55e11d](https://github.com/MustardSeedNetworks/seed/commit/d55e11d7be0bea17b9aeb10730a75a01a8234b7e))
* **main:** release 0.202.1 ([#1259](https://github.com/MustardSeedNetworks/seed/issues/1259)) ([2f794bc](https://github.com/MustardSeedNetworks/seed/commit/2f794bc7b195dfa0d51adba83e1fc7ed513a1150))
* **main:** release 0.203.0 ([#1266](https://github.com/MustardSeedNetworks/seed/issues/1266)) ([e935879](https://github.com/MustardSeedNetworks/seed/commit/e93587925153bec6b715c4f6cd37bcb067294da2))
* **main:** release 0.204.0 ([#1275](https://github.com/MustardSeedNetworks/seed/issues/1275)) ([c9a1fc5](https://github.com/MustardSeedNetworks/seed/commit/c9a1fc5d76fde5ab9efad35b3c7f885296ea1906))
* **main:** release 0.205.0 ([#1279](https://github.com/MustardSeedNetworks/seed/issues/1279)) ([99fcc19](https://github.com/MustardSeedNetworks/seed/commit/99fcc199e0a0886e20159a70cf92d0f0f5972ef3))
* **main:** release 0.206.0 ([#1298](https://github.com/MustardSeedNetworks/seed/issues/1298)) ([5bd94bf](https://github.com/MustardSeedNetworks/seed/commit/5bd94bfd2138acc80b05f9cd7bf97cef73fc9a1f))
* **main:** release 0.207.0 ([#1331](https://github.com/MustardSeedNetworks/seed/issues/1331)) ([de6754c](https://github.com/MustardSeedNetworks/seed/commit/de6754c89f65e7ea050ae6e02966421ef0669644))
* **main:** release 0.208.0 ([#1400](https://github.com/MustardSeedNetworks/seed/issues/1400)) ([ecbfb93](https://github.com/MustardSeedNetworks/seed/commit/ecbfb9321608c549ac3a0ba0407fb685317b486f))
* **main:** release 0.209.0 ([#1409](https://github.com/MustardSeedNetworks/seed/issues/1409)) ([e936260](https://github.com/MustardSeedNetworks/seed/commit/e936260857d7f19c7f3084d3ac90c275a31cda26))
* **main:** release 0.210.0 ([#1576](https://github.com/MustardSeedNetworks/seed/issues/1576)) ([4bfa440](https://github.com/MustardSeedNetworks/seed/commit/4bfa4408a68d9182dd1c99b8721b4ccea25c081e))
* **main:** release 0.211.0 ([#1604](https://github.com/MustardSeedNetworks/seed/issues/1604)) ([dd6763a](https://github.com/MustardSeedNetworks/seed/commit/dd6763a5b53538c871a307955577242392361396))
* **main:** release 0.212.0 ([#1610](https://github.com/MustardSeedNetworks/seed/issues/1610)) ([8124e7a](https://github.com/MustardSeedNetworks/seed/commit/8124e7a537eb0acb7c2ccddd18bd44a19efba990))
* **main:** release 0.212.1 ([#1674](https://github.com/MustardSeedNetworks/seed/issues/1674)) ([33e4388](https://github.com/MustardSeedNetworks/seed/commit/33e4388b4faa927b72e176803f0271576c9fd848))
* **release-please:** sync version manifest to current release (0.210.0) ([#1601](https://github.com/MustardSeedNetworks/seed/issues/1601)) ([f0a4975](https://github.com/MustardSeedNetworks/seed/commit/f0a4975c8efb182b0ff7026a065117ebd2a4daf6))
* **release:** expose release train metadata ([#1693](https://github.com/MustardSeedNetworks/seed/issues/1693)) ([58c79fc](https://github.com/MustardSeedNetworks/seed/commit/58c79fc26b96d19c213afc4773dcf54e65d7212e))
* rename module path to the MustardSeedNetworks org ([#1551](https://github.com/MustardSeedNetworks/seed/issues/1551)) ([3b3ed98](https://github.com/MustardSeedNetworks/seed/commit/3b3ed985b7af9ef6be99b4de486e016dd68a16e3))
* **security:** add output-escaping/XSS regression gate ([#1225](https://github.com/MustardSeedNetworks/seed/issues/1225)) ([#1245](https://github.com/MustardSeedNetworks/seed/issues/1245)) ([6079e62](https://github.com/MustardSeedNetworks/seed/commit/6079e623e7adf5a4e29600d97c604e4584249d21))
* simplify local make system ([1fde493](https://github.com/MustardSeedNetworks/seed/commit/1fde493020b00afb41caa1bb666b49fcd0b42325))
* **test:** add libpcap-free 'make test-fast' inner-loop target ([#1519](https://github.com/MustardSeedNetworks/seed/issues/1519)) ([bb721f0](https://github.com/MustardSeedNetworks/seed/commit/bb721f0f95111e3c3a8c3bb8d63fd72fe5d39a21))
* **ui:** harmonize tsconfig layout, remove emitted vite.config.d.ts ([#1594](https://github.com/MustardSeedNetworks/seed/issues/1594)) ([c7eec34](https://github.com/MustardSeedNetworks/seed/commit/c7eec34920b533e05e48cee743cbd47f9c37b1a9))
* **ui:** replace eslint references with biome ([#1695](https://github.com/MustardSeedNetworks/seed/issues/1695)) ([139b5d5](https://github.com/MustardSeedNetworks/seed/commit/139b5d5a552aeda50cc6ec4721736ef7f835d297))
* **ui:** retire sync-shell — seed owns its shell files independently ([#1234](https://github.com/MustardSeedNetworks/seed/issues/1234)) ([270455a](https://github.com/MustardSeedNetworks/seed/commit/270455a9578669924d83383ecd683d77d9c75462))

## [0.212.1](https://github.com/MustardSeedNetworks/seed/compare/v0.212.0...v0.212.1) (2026-06-16)


### Features

* **ui:** per-item module accent colour in the sidebar (M1 follow-up) ([#1682](https://github.com/MustardSeedNetworks/seed/issues/1682)) ([2aa1f1c](https://github.com/MustardSeedNetworks/seed/commit/2aa1f1c730fd9da38315f6417dff246eee0d9558))


### Bug Fixes

* **ui:** close SEED_UI_ARCH_PLAN D-batch — distinct brand green (L1) + profile entry points (VL1) + on-brand AA (VL2) ([#1680](https://github.com/MustardSeedNetworks/seed/issues/1680)) ([af1e7d1](https://github.com/MustardSeedNetworks/seed/commit/af1e7d1496b965f3c507df7fa9cd6c8af3d7b35e))


### Code Refactoring

* **alerts:** relocate Alert/Rule/ListenerEvent rows to domain pkgs, make persistence-free (WS-B) ([#1676](https://github.com/MustardSeedNetworks/seed/issues/1676)) ([0323857](https://github.com/MustardSeedNetworks/seed/commit/0323857e9d56274430e766e76e3790415ed93b4a))
* **ci:** empty the json-casing baseline; exempt external adapters by marker (ADR-0010 revised) ([#1684](https://github.com/MustardSeedNetworks/seed/issues/1684)) ([1df12dd](https://github.com/MustardSeedNetworks/seed/commit/1df12dd6c5a80fd8a9ade6b746aea8aabbe0be66))
* **health:** make health surface persistence-free (WS-B) ([#1678](https://github.com/MustardSeedNetworks/seed/issues/1678)) ([3215b97](https://github.com/MustardSeedNetworks/seed/commit/3215b97f13640d680083445865add9d5b27fcd16))
* **polling:** relocate PollingTarget to domain pkg + orchestrator ports, make persistence-free (WS-B) ([#1675](https://github.com/MustardSeedNetworks/seed/issues/1675)) ([74c4162](https://github.com/MustardSeedNetworks/seed/commit/74c4162e637d459ad2be551b13e49e0489796b2c))
* **probe:** narrow persistence port to domain types (WS-B1) ([#1672](https://github.com/MustardSeedNetworks/seed/issues/1672)) ([e07d928](https://github.com/MustardSeedNetworks/seed/commit/e07d9285c75763bdd12bc59b3c59f10a06b6a4bb))
* **retention:** relocate rollup SQL adapters to internal/database (WS-B5) ([#1677](https://github.com/MustardSeedNetworks/seed/issues/1677)) ([7370c51](https://github.com/MustardSeedNetworks/seed/commit/7370c515dfd01046fd9c666a4a9a2c73c1edca34))
* **topology:** relocate row types to domain pkgs, make persistence-free (WS-B) ([#1673](https://github.com/MustardSeedNetworks/seed/issues/1673)) ([e768fcc](https://github.com/MustardSeedNetworks/seed/commit/e768fcc7b884fd387b87faf46a799ca4740f7256))
* **ui:** function-first sidebar nav IA, retire botanical metaphor (M1, [#1452](https://github.com/MustardSeedNetworks/seed/issues/1452)) ([#1681](https://github.com/MustardSeedNetworks/seed/issues/1681)) ([546d937](https://github.com/MustardSeedNetworks/seed/commit/546d93724dd5f26a7ba984b32a0923bc61fe1ff2))


### Documentation

* **adr-0010:** revise to pure boundary mapping — wire is 100% camelCase, no exceptions ([#1683](https://github.com/MustardSeedNetworks/seed/issues/1683)) ([0259e3d](https://github.com/MustardSeedNetworks/seed/commit/0259e3d39c94e60887446307f33487e02fc6cde1))
* **ws-b:** close B3 — identity is the documented ADR-0024 exception ([#1679](https://github.com/MustardSeedNetworks/seed/issues/1679)) ([2bd9ab6](https://github.com/MustardSeedNetworks/seed/commit/2bd9ab6ccde0436c13ec38c172e5f3525723cb9f))


### Miscellaneous

* **deps:** always-latest toolchain sweep (lockstep with stem) ([#1687](https://github.com/MustardSeedNetworks/seed/issues/1687)) ([60b54c4](https://github.com/MustardSeedNetworks/seed/commit/60b54c415ab2a120a5c39a98b0a2838859086868))
* **deps:** refresh go module graph ([#1690](https://github.com/MustardSeedNetworks/seed/issues/1690)) ([82144b5](https://github.com/MustardSeedNetworks/seed/commit/82144b5e5c5c479289df6b759aaedb3119509b06))
* **github:** standardize repo governance ([#1689](https://github.com/MustardSeedNetworks/seed/issues/1689)) ([f409599](https://github.com/MustardSeedNetworks/seed/commit/f40959915e63522e426b5c60236794122085977e))
* **release:** expose release train metadata ([#1693](https://github.com/MustardSeedNetworks/seed/issues/1693)) ([58c79fc](https://github.com/MustardSeedNetworks/seed/commit/58c79fc26b96d19c213afc4773dcf54e65d7212e))
* **ui:** replace eslint references with biome ([#1695](https://github.com/MustardSeedNetworks/seed/issues/1695)) ([139b5d5](https://github.com/MustardSeedNetworks/seed/commit/139b5d5a552aeda50cc6ec4721736ef7f835d297))

## [0.212.0](https://github.com/MustardSeedNetworks/seed/compare/v0.211.0...v0.212.0) (2026-06-13)


### ⚠ BREAKING CHANGES

* **probe:** the health-check run/settings/anomalies endpoints moved from /telemetry/health-checks/* to /telemetry/probes/*. Pre-alpha, no compat.
* **probe:** /run response drops the unrendered CustomTestResult fields (security headers, redirect chains, body-match, per-endpoint vertical detail that was never populated). Pre-alpha, no compat.
* **health:** delete the dead health_check_results read-path (ADR-0026) ([#1636](https://github.com/MustardSeedNetworks/seed/issues/1636))

### Features

* **anomaly:** add error severity to the four-level ladder (ADR-0021 ph5) ([#1637](https://github.com/MustardSeedNetworks/seed/issues/1637)) ([5127ca0](https://github.com/MustardSeedNetworks/seed/commit/5127ca0fe53a59bd36e1b86aab2f2a544f872bc1))
* **anomaly:** daily-rollup census of the anomaly store (ADR-0028) ([#1650](https://github.com/MustardSeedNetworks/seed/issues/1650)) ([97aab3e](https://github.com/MustardSeedNetworks/seed/commit/97aab3ec2598a3ceb9d8976746dcf7483b058041))
* **anomaly:** persist the anomaly engine in SQL (ADR-0021 phase 1) ([#1629](https://github.com/MustardSeedNetworks/seed/issues/1629)) ([83f2ce4](https://github.com/MustardSeedNetworks/seed/commit/83f2ce46eeebf55335c3de548117dc0ca7e71527))
* **anomaly:** persist the Wi-Fi anomaly stream + load-on-start (ADR-0021 phase 3) ([#1632](https://github.com/MustardSeedNetworks/seed/issues/1632)) ([d24f896](https://github.com/MustardSeedNetworks/seed/commit/d24f896670a1c62cd4d19e1522cf3e14af3f9978))
* **anomaly:** probe is the active-monitoring anomaly producer (ADR-0025) ([#1635](https://github.com/MustardSeedNetworks/seed/issues/1635)) ([500c890](https://github.com/MustardSeedNetworks/seed/commit/500c8902a7ecfeb51ea49a71e8b8b8665d9507d7))
* **anomaly:** re-derive catalog-static impact/followUps on store read (ADR-0029) ([#1653](https://github.com/MustardSeedNetworks/seed/issues/1653)) ([be81b0d](https://github.com/MustardSeedNetworks/seed/commit/be81b0db76e31a7fccc749b0cc9b58a0f5bf3e56))
* **anomaly:** resolve probe anomalies immediately on a clean result ([#1638](https://github.com/MustardSeedNetworks/seed/issues/1638)) ([1365e9a](https://github.com/MustardSeedNetworks/seed/commit/1365e9a1b154a45c59ea1f9036e2fdf7af115957))
* **anomaly:** TTL-purge resolved anomalies in retention (ADR-0021 phase 2) ([#1630](https://github.com/MustardSeedNetworks/seed/issues/1630)) ([0ca47d7](https://github.com/MustardSeedNetworks/seed/commit/0ca47d7566e978a421b7189b3341a458dfea089e))
* **probe:** add eight health-check vertical checkers (ADR-0027 P1) ([#1641](https://github.com/MustardSeedNetworks/seed/issues/1641)) ([99649e5](https://github.com/MustardSeedNetworks/seed/commit/99649e5c55185c3ea511677401edc01a480afb7a))
* **probe:** detect TLS certificate expiry as a probe anomaly ([#1639](https://github.com/MustardSeedNetworks/seed/issues/1639)) ([a93cac5](https://github.com/MustardSeedNetworks/seed/commit/a93cac5fd260315b5fdd6009dcf7f95cfb0cff20))
* **probe:** enrich HTTP/HTTPS checker with per-phase timings and cert summary (ADR-0027 P3a) ([#1643](https://github.com/MustardSeedNetworks/seed/issues/1643)) ([94c47ab](https://github.com/MustardSeedNetworks/seed/commit/94c47ab9a20b901c8e02d5270a980f218d6f7977))
* **probe:** make the probes table store-of-record for health-check settings (ADR-0027 P2) ([#1642](https://github.com/MustardSeedNetworks/seed/issues/1642)) ([b37fe80](https://github.com/MustardSeedNetworks/seed/commit/b37fe8024b75c2398652c2376d78daeafae67de2))
* **probe:** rename health-check transport to /telemetry/probes/* (ADR-0027 P5) ([#1645](https://github.com/MustardSeedNetworks/seed/issues/1645)) ([8296cc2](https://github.com/MustardSeedNetworks/seed/commit/8296cc2a64645ae94cf38859b701da4270d6a873))
* **probe:** run health checks through the probe engine; delete the legacy stack (ADR-0027 P3+P4) ([#1644](https://github.com/MustardSeedNetworks/seed/issues/1644)) ([15bf342](https://github.com/MustardSeedNetworks/seed/commit/15bf3423b4bcb45636578cae5f291d5449d5efd8))
* **ui:** add RequireRole/RequireAdmin gate + hide user mgmt from non-admins ([#1254](https://github.com/MustardSeedNetworks/seed/issues/1254)) ([#1586](https://github.com/MustardSeedNetworks/seed/issues/1586)) ([0ee5694](https://github.com/MustardSeedNetworks/seed/commit/0ee569495c9e1ea2efbd41a5c6b7ff3207d24cde))


### Bug Fixes

* **api:** require operator role on persistent-write routes ([#1631](https://github.com/MustardSeedNetworks/seed/issues/1631)) ([fc7e3f5](https://github.com/MustardSeedNetworks/seed/commit/fc7e3f5e2bb3d6af97652e74f45198c2509546ae))
* **database:** de-collide anomaly census test fixtures (flaky) ([#1670](https://github.com/MustardSeedNetworks/seed/issues/1670)) ([c6414cd](https://github.com/MustardSeedNetworks/seed/commit/c6414cd709511719595dce59560ccca8cffddd7b))
* **probe:** seed factory health-check targets on first run ([#1646](https://github.com/MustardSeedNetworks/seed/issues/1646)) ([4283b6e](https://github.com/MustardSeedNetworks/seed/commit/4283b6e228851c4e201db7799c556be1c4de49a1))


### Code Refactoring

* **anomaly:** converge producers onto one server-owned engine (ADR-0029 P2+P3) ([#1652](https://github.com/MustardSeedNetworks/seed/issues/1652)) ([58007b4](https://github.com/MustardSeedNetworks/seed/commit/58007b4c36b3bbb011712901f52ed178b2aac6c8))
* **anomaly:** delete bespoke health detector, read unified store (ADR-0021 phase 4) ([#1633](https://github.com/MustardSeedNetworks/seed/issues/1633)) ([438e3d1](https://github.com/MustardSeedNetworks/seed/commit/438e3d1b7dbb8256ce2cf31c60fde2485d9ca627))
* **anomaly:** source on detection + source-scoped prune (ADR-0029 P1) ([#1651](https://github.com/MustardSeedNetworks/seed/issues/1651)) ([21dc576](https://github.com/MustardSeedNetworks/seed/commit/21dc576d5cdc573ce64128230615175278bff469))
* **api:** clean-hexagonal alerts retrofit (ADR-0020) ([#1612](https://github.com/MustardSeedNetworks/seed/issues/1612)) ([412e2b9](https://github.com/MustardSeedNetworks/seed/commit/412e2b984f715a861dd992b5bc99ea551235b4cc))
* **api:** clean-hexagonal discovery retrofit (ADR-0020) ([#1616](https://github.com/MustardSeedNetworks/seed/issues/1616)) ([6d0c035](https://github.com/MustardSeedNetworks/seed/commit/6d0c0352bfc159ade52df7b2a2c19244e6864cb7))
* **api:** clean-hexagonal health-monitoring retrofit (ADR-0020) ([#1618](https://github.com/MustardSeedNetworks/seed/issues/1618)) ([4678697](https://github.com/MustardSeedNetworks/seed/commit/46786970f84b34681749630cd53373144ead5258))
* **api:** clean-hexagonal network exemplar + ADR-0020 ([#1611](https://github.com/MustardSeedNetworks/seed/issues/1611)) ([dbaf8dc](https://github.com/MustardSeedNetworks/seed/commit/dbaf8dcac55b6805b00fa5695293ef8bac1005f4))
* **api:** clean-hexagonal profiles retrofit (ADR-0020) ([#1614](https://github.com/MustardSeedNetworks/seed/issues/1614)) ([b5516b2](https://github.com/MustardSeedNetworks/seed/commit/b5516b2267fce5a8a8508ea9a6059c4b3c62767b))
* **api:** clean-hexagonal settings retrofit (ADR-0020) ([#1613](https://github.com/MustardSeedNetworks/seed/issues/1613)) ([5d98097](https://github.com/MustardSeedNetworks/seed/commit/5d980976222a0ed2aad5ce459c7eb85cb4ec70b6))
* **api:** clean-hexagonal wifi retrofit (ADR-0020) ([#1615](https://github.com/MustardSeedNetworks/seed/issues/1615)) ([e903744](https://github.com/MustardSeedNetworks/seed/commit/e90374445d7b1eb66b2d1cb91cc40e9b6319973d))
* **api:** delete ServiceContainer, flatten services onto Server (D1) ([#1627](https://github.com/MustardSeedNetworks/seed/issues/1627)) ([feb198e](https://github.com/MustardSeedNetworks/seed/commit/feb198e3df9db18ebb83aa497535f550f8d18fc2))
* **api:** extract drainJobSubstrate + test the shutdown drain ([#1628](https://github.com/MustardSeedNetworks/seed/issues/1628)) ([1b8d2cf](https://github.com/MustardSeedNetworks/seed/commit/1b8d2cf5c69833b0152a44b734b41b8aeb73c213))
* **api:** finish the profiles handler strangle (ADR-0020, WS-A11c) ([#1669](https://github.com/MustardSeedNetworks/seed/issues/1669)) ([c92f82b](https://github.com/MustardSeedNetworks/seed/commit/c92f82b865c1881b4dda210bef3ab07c0f81e657))
* **api:** route handler license reads through s.licenseManager() (D1 prep) ([#1626](https://github.com/MustardSeedNetworks/seed/issues/1626)) ([54318de](https://github.com/MustardSeedNetworks/seed/commit/54318de5ef736edddf58b1776f736ca05d14f787))
* **api:** route the config-read/write residuals through use-cases (ADR-0020, WS-A11a) ([#1665](https://github.com/MustardSeedNetworks/seed/issues/1665)) ([50d92e5](https://github.com/MustardSeedNetworks/seed/commit/50d92e5a89cc4ba7ef10db8691edbd3876bcf462))
* **api:** strangle alert inbox into a use-case (ADR-0020, WS-A8) ([#1662](https://github.com/MustardSeedNetworks/seed/issues/1662)) ([c25d4cf](https://github.com/MustardSeedNetworks/seed/commit/c25d4cf20fe0f1db1dfe8c8e443f42996e5a209d))
* **api:** strangle config backup/restore into a use-case (ADR-0020, WS-A9) ([#1663](https://github.com/MustardSeedNetworks/seed/issues/1663)) ([5460177](https://github.com/MustardSeedNetworks/seed/commit/54601779883293399a0472a98be3e78463b0f799))
* **api:** strangle device-discovery settings into a use-case (ADR-0020, WS-A1) ([#1655](https://github.com/MustardSeedNetworks/seed/issues/1655)) ([f39f91e](https://github.com/MustardSeedNetworks/seed/commit/f39f91e45d81fd13f26758f52d0a366edd1e3220))
* **api:** strangle diagnostic export into a use-case (ADR-0020, WS-A10) ([#1664](https://github.com/MustardSeedNetworks/seed/issues/1664)) ([b1774de](https://github.com/MustardSeedNetworks/seed/commit/b1774de2d1963efa22e21511783cc7d617808ac8))
* **api:** strangle engine-status into a use-case over the engine registry (ADR-0020) ([#1625](https://github.com/MustardSeedNetworks/seed/issues/1625)) ([1f2c242](https://github.com/MustardSeedNetworks/seed/commit/1f2c242a6af96780696974bf9dd2e212fff5265b))
* **api:** strangle health-checks settings into a use-case (ADR-0020, WS-A4) ([#1658](https://github.com/MustardSeedNetworks/seed/issues/1658)) ([6967217](https://github.com/MustardSeedNetworks/seed/commit/696721726d007f55b6a596b86e7f2ecfbc581a4e))
* **api:** strangle identity (users/oauth/tokens) into use-cases over repository ports (C4) ([#1624](https://github.com/MustardSeedNetworks/seed/issues/1624)) ([8c335f8](https://github.com/MustardSeedNetworks/seed/commit/8c335f82a8396e0ed5242f873a8a37abc6d0db58))
* **api:** strangle main settings into a use-case (ADR-0020, WS-A2) ([#1656](https://github.com/MustardSeedNetworks/seed/issues/1656)) ([c9f16ce](https://github.com/MustardSeedNetworks/seed/commit/c9f16cefacc7415254b49c5c3894e3a0b8ce950e))
* **api:** strangle polling-targets CRUD into a use-case (ADR-0020, WS-A7) ([#1661](https://github.com/MustardSeedNetworks/seed/issues/1661)) ([3640352](https://github.com/MustardSeedNetworks/seed/commit/364035220a43b59357a5716e12bf514aae3f9a6a))
* **api:** strangle security settings into a use-case + fix SNMP deadlock (ADR-0020, WS-A3) ([#1657](https://github.com/MustardSeedNetworks/seed/issues/1657)) ([7146550](https://github.com/MustardSeedNetworks/seed/commit/714655090163ec6fe53fd43e9c97cc94ff479929))
* **api:** strangle the log query into a use-case (ADR-0020, WS-A11b) ([#1668](https://github.com/MustardSeedNetworks/seed/issues/1668)) ([9e22a05](https://github.com/MustardSeedNetworks/seed/commit/9e22a0516eea99a31ed116b895b4824f9f3f7f18))
* **api:** strangle topology read endpoints into a use-case (ADR-0020, WS-A6) ([#1660](https://github.com/MustardSeedNetworks/seed/issues/1660)) ([b306028](https://github.com/MustardSeedNetworks/seed/commit/b30602829e09df79d561e39fedb0a3dffd7abade))
* **api:** strangle update handlers into internal/update/lifecycle (C3) ([#1620](https://github.com/MustardSeedNetworks/seed/issues/1620)) ([decba0c](https://github.com/MustardSeedNetworks/seed/commit/decba0c78f65dc2522fcfe6e5148cefe386304fa))
* **api:** strangle vulnerability-scanner settings into the security use-case (ADR-0020, WS-A5) ([#1659](https://github.com/MustardSeedNetworks/seed/issues/1659)) ([1d0d33e](https://github.com/MustardSeedNetworks/seed/commit/1d0d33edf428950482953cec542b4bb9b7c7299f))
* **config:** reject plaintext credentials, delete legacy v0/JWT path ([#1623](https://github.com/MustardSeedNetworks/seed/issues/1623)) ([05350bf](https://github.com/MustardSeedNetworks/seed/commit/05350bfbece01b143d7f7fde135fa52249d6bc6e))
* **health:** delete the dead health_check_results read-path (ADR-0026) ([#1636](https://github.com/MustardSeedNetworks/seed/issues/1636)) ([ded0f2b](https://github.com/MustardSeedNetworks/seed/commit/ded0f2bd43ee077374a9bb5322a4f4413782bf37))
* **paths:** drop legacy config-name fallback and dead DetectLegacyConfig ([#1609](https://github.com/MustardSeedNetworks/seed/issues/1609)) ([7b0c6f9](https://github.com/MustardSeedNetworks/seed/commit/7b0c6f926416c2dd9ee054497d6ee64785b0c609))


### Documentation

* **adr:** ADR-0021 persist anomaly engine in SQL + converge all sources ([#1619](https://github.com/MustardSeedNetworks/seed/issues/1619)) ([d2d367a](https://github.com/MustardSeedNetworks/seed/commit/d2d367a67f0e95302d593ad65e0da8b4c3f7628f))
* **adr:** design daily rollups for the anomaly store (ADR-0028) ([#1649](https://github.com/MustardSeedNetworks/seed/issues/1649)) ([1150b81](https://github.com/MustardSeedNetworks/seed/commit/1150b8144ff471aa7534648dbf8172c3ba5baea6))
* **adr:** hygiene pass — honest statuses + two as-built ADRs ([#1622](https://github.com/MustardSeedNetworks/seed/issues/1622)) ([db669f7](https://github.com/MustardSeedNetworks/seed/commit/db669f7615fe7082a8d2b9de875475178d8ee492))
* **adr:** probe is the active-monitoring anomaly source (ADR-0025) ([#1634](https://github.com/MustardSeedNetworks/seed/issues/1634)) ([58370b8](https://github.com/MustardSeedNetworks/seed/commit/58370b8128df232344cd26beb435f06645160c9e))
* **adr:** scope migrating health-checks onto probe, then renaming (ADR-0027) ([#1640](https://github.com/MustardSeedNetworks/seed/issues/1640)) ([a7da0da](https://github.com/MustardSeedNetworks/seed/commit/a7da0da7a4522dd7b37e2f18c41a754b3c57026b))
* **architecture:** add the architecture completion plan (no-shortcuts strangle finish) ([#1654](https://github.com/MustardSeedNetworks/seed/issues/1654)) ([4b189f1](https://github.com/MustardSeedNetworks/seed/commit/4b189f12644392cad33446086d1854d55eed326a))


### Miscellaneous

* **ci:** add filename-policy gate for decomposed packages ([#1617](https://github.com/MustardSeedNetworks/seed/issues/1617)) ([cd8642c](https://github.com/MustardSeedNetworks/seed/commit/cd8642ccea29b2324573d43447f2a2a2d8b8772b))
* **cleanup:** remove dead HTTP-redirector remnants ([#1621](https://github.com/MustardSeedNetworks/seed/issues/1621)) ([1fc1275](https://github.com/MustardSeedNetworks/seed/commit/1fc12751a4c548be59b7f21bc20708f31f95f22b))
* **deps:** update frontend deps to latest + clear esbuild advisory ([#1666](https://github.com/MustardSeedNetworks/seed/issues/1666)) ([9d669bf](https://github.com/MustardSeedNetworks/seed/commit/9d669bfbc22b448c26986835c8f4e05f29ea2408))
* **deps:** update Go modules to latest ([#1667](https://github.com/MustardSeedNetworks/seed/issues/1667)) ([2153916](https://github.com/MustardSeedNetworks/seed/commit/21539163bc5ea64f432991a9b2c2b0d3b96f3477))

## [0.211.0](https://github.com/MustardSeedNetworks/seed/compare/v0.210.0...v0.211.0) (2026-06-09)


### ⚠ BREAKING CHANGES

* the SSO config API (/api/v1 sso update) and the CVE-database file format now use camelCase keys (clientId, clientSecret, redirectUrl, tenantId, cvssScore, lastUpdated) instead of snake_case. Pre-1.0, no external consumers.
* **api:** the MFA (/api/v1 mfa) and auth-login responses now use camelCase json keys (totpEnabled, provisioningUri, qrCodePngBase64, mfaRequired, mfaToken, etc.) instead of snake_case. Pre-1.0, no external consumers; the bundled UI is updated in lockstep.
* **api:** the logs (/api logs query + stats), profiles (/api profiles), and config-version/restore APIs now use camelCase json keys (requestId, durationMs, totalCount, byLevel, isDefault, needsMigration, backupName, etc.) instead of snake_case. Pre-1.0, no external consumers; the bundled UI is updated in lockstep.
* **discovery:** the network problem-detection API (/api problem + network-problems responses) now uses camelCase json keys (deviceId, bySeverity, signalDbm, interfaceErrors, scanDurationMs, etc.) instead of snake_case. Pre-1.0, no external consumers; the bundled UI is updated in lockstep via regenerated types.
* **discovery:** the wi-fi discovery API (/api/v1/wifi discovery responses) now uses camelCase json keys (isHidden, securityType, frequencyMhz, signalDbm, etc.) instead of snake_case. Pre-1.0, no external consumers; the bundled UI is updated in lockstep via regenerated types.

### Features

* **discovery:** embed ieee oui registry as single source, drop hardcoded maps ([#1591](https://github.com/MustardSeedNetworks/seed/issues/1591)) ([18cc72b](https://github.com/MustardSeedNetworks/seed/commit/18cc72b7b54adfa27b523d4b4d584a4d2d05118c))


### Bug Fixes

* **auth:** stop a late mount /status probe from clobbering a completed login ([#1598](https://github.com/MustardSeedNetworks/seed/issues/1598)) ([cc43f4a](https://github.com/MustardSeedNetworks/seed/commit/cc43f4a86933502bcaa71a1eedb40be194f1761b))
* **ui:** suppress node dep0205 build warning ([e3dadd7](https://github.com/MustardSeedNetworks/seed/commit/e3dadd74db4be15a8228a5129e0a8d7c535e04df))


### Performance Improvements

* **ui:** split vendor chunks, add modern build target + analyzer ([#1599](https://github.com/MustardSeedNetworks/seed/issues/1599)) ([f451a70](https://github.com/MustardSeedNetworks/seed/commit/f451a70af38814f805ed97f7dc6846887fb3f723))


### Code Refactoring

* **api:** camelcase the logs, profiles, and config-version apis (no grandfathering) ([#1606](https://github.com/MustardSeedNetworks/seed/issues/1606)) ([865fc78](https://github.com/MustardSeedNetworks/seed/commit/865fc78f29faa791ec92a20510a803f67a1a9c2f))
* **api:** camelcase the mfa + auth-login apis (no grandfathering) ([#1607](https://github.com/MustardSeedNetworks/seed/issues/1607)) ([bac77f5](https://github.com/MustardSeedNetworks/seed/commit/bac77f5c947b772755fb373b79aea948dcd21462))
* **api:** drop redundant in-handler method guards (registry is authoritative) ([#1597](https://github.com/MustardSeedNetworks/seed/issues/1597)) ([2abceb3](https://github.com/MustardSeedNetworks/seed/commit/2abceb33e30ec293007039cae1b14c4c159d9059))
* camelcase the sso-config + cve-db apis; finish OUR de-grandfathering ([#1608](https://github.com/MustardSeedNetworks/seed/issues/1608)) ([e072884](https://github.com/MustardSeedNetworks/seed/commit/e0728848990f356d9a4c085113cf6df677adb746))
* **discovery:** camelcase the problem-detection api (no grandfathering) ([#1605](https://github.com/MustardSeedNetworks/seed/issues/1605)) ([5aff651](https://github.com/MustardSeedNetworks/seed/commit/5aff651975a10b85fec246dbf2a7820a0966ba16))
* **discovery:** drive Engine collectors through ports (adr-0018) ([#1592](https://github.com/MustardSeedNetworks/seed/issues/1592)) ([9edde7b](https://github.com/MustardSeedNetworks/seed/commit/9edde7b6031041b66b59506e56cfe25870684934))
* **discovery:** extract name/identity resolution to resolve leaf (adr-0018) ([#1589](https://github.com/MustardSeedNetworks/seed/issues/1589)) ([a874afd](https://github.com/MustardSeedNetworks/seed/commit/a874afd779d066ecf507d0130db5d2f8da3e3eba))
* **discovery:** relocate bluetooth collector to enumerate stage (adr-0018) ([#1595](https://github.com/MustardSeedNetworks/seed/issues/1595)) ([4933b79](https://github.com/MustardSeedNetworks/seed/commit/4933b79351872eba417584471a1e8c6d73b37ddb))
* **discovery:** relocate port-scan leaf to fingerprint stage (adr-0018) ([#1587](https://github.com/MustardSeedNetworks/seed/issues/1587)) ([e598809](https://github.com/MustardSeedNetworks/seed/commit/e5988094e2e6458f276cda5bcb172a8a63ccacc7))
* **discovery:** relocate wi-fi collector to enumerate + camelcase the wi-fi api (adr-0018) ([#1596](https://github.com/MustardSeedNetworks/seed/issues/1596)) ([6d2f3d8](https://github.com/MustardSeedNetworks/seed/commit/6d2f3d8a83198da719bdd4d4f6b6339cb8ea3398))
* **discovery:** relocate wired+service collector to enumerate stage (adr-0018) ([#1600](https://github.com/MustardSeedNetworks/seed/issues/1600)) ([9139cbe](https://github.com/MustardSeedNetworks/seed/commit/9139cbe31b746e1cc01d445263cec29284bf8737))
* **ui:** decompose app.tsx god component into AppShell + useAppOrchestration (b1) ([#1578](https://github.com/MustardSeedNetworks/seed/issues/1578)) ([c58408d](https://github.com/MustardSeedNetworks/seed/commit/c58408d817a8cd8d2496b57e4159dc6b3a9fd58c))
* **ui:** replace HeaderBar inline SVGs with lucide icons (m2) ([#1582](https://github.com/MustardSeedNetworks/seed/issues/1582)) ([3932184](https://github.com/MustardSeedNetworks/seed/commit/39321848d97bb60858b001c31b05e7a4e19e2f67))
* **ui:** single canonical SeedLogo brand mark (m3) ([#1580](https://github.com/MustardSeedNetworks/seed/issues/1580)) ([90663a6](https://github.com/MustardSeedNetworks/seed/commit/90663a68e3e35ef516aa645ca25ad61d963cfb46))


### Documentation

* **ui:** add STATE.md state-placement guide (l2) ([#1583](https://github.com/MustardSeedNetworks/seed/issues/1583)) ([02f995c](https://github.com/MustardSeedNetworks/seed/commit/02f995c5b0ad82ea767ffce85abc684099ad863d))


### Tests

* **auth:** lock the first-run no-usable-default-password invariant ([#1242](https://github.com/MustardSeedNetworks/seed/issues/1242)) ([#1585](https://github.com/MustardSeedNetworks/seed/issues/1585)) ([322c902](https://github.com/MustardSeedNetworks/seed/commit/322c902a9ea9877b7865ca2cfdac926665d6ec2b))
* **auth:** pin the CSRF exempt-list with a coverage gate ([#1223](https://github.com/MustardSeedNetworks/seed/issues/1223)) ([#1584](https://github.com/MustardSeedNetworks/seed/issues/1584)) ([ddb05de](https://github.com/MustardSeedNetworks/seed/commit/ddb05dece69e7e5c63a0fc7ef6ea6282412eb19c))
* **e2e:** fix rotating post-login dashboard flake in auth-complete ([#1588](https://github.com/MustardSeedNetworks/seed/issues/1588)) ([d7689b9](https://github.com/MustardSeedNetworks/seed/commit/d7689b93b0d0b65eb28e8ea5742578c88571449a))


### Continuous Integration

* **e2e:** promote full E2E suite to a blocking gate via CI Complete ([#1590](https://github.com/MustardSeedNetworks/seed/issues/1590)) ([1df9974](https://github.com/MustardSeedNetworks/seed/commit/1df9974ad352aa496f00f967f65b043dcc2b2601))
* **release-please:** honor pre-major bump config; force next release to 0.211.0 ([#1603](https://github.com/MustardSeedNetworks/seed/issues/1603)) ([d5a784e](https://github.com/MustardSeedNetworks/seed/commit/d5a784e68eeebf787004defecdf77c328249dcdc))
* **release:** drop macOS amd64 (Intel) build target ([#1579](https://github.com/MustardSeedNetworks/seed/issues/1579)) ([9eaa9d4](https://github.com/MustardSeedNetworks/seed/commit/9eaa9d474dbb04bd7d185ae72fc46a32acfa4c50))


### Miscellaneous

* **release-please:** sync version manifest to current release (0.210.0) ([#1601](https://github.com/MustardSeedNetworks/seed/issues/1601)) ([f0a4975](https://github.com/MustardSeedNetworks/seed/commit/f0a4975c8efb182b0ff7026a065117ebd2a4daf6))
* **ui:** harmonize tsconfig layout, remove emitted vite.config.d.ts ([#1594](https://github.com/MustardSeedNetworks/seed/issues/1594)) ([c7eec34](https://github.com/MustardSeedNetworks/seed/commit/c7eec34920b533e05e48cee743cbd47f9c37b1a9))

## [0.210.0](https://github.com/MustardSeedNetworks/seed/compare/v0.209.0...v0.210.0) (2026-06-08)


### Features

* **license:** replace forgeable rotor cipher with Ed25519-signed tokens ([#1575](https://github.com/MustardSeedNetworks/seed/issues/1575)) ([bb70f10](https://github.com/MustardSeedNetworks/seed/commit/bb70f10f3d0450c72cc13cd94b6538224ec19ad7))

## [0.209.0](https://github.com/MustardSeedNetworks/seed/compare/v0.208.0...v0.209.0) (2026-06-08)


### Features

* **anomaly:** general network-anomaly engine + data-driven catalog (W4a) ([#1531](https://github.com/MustardSeedNetworks/seed/issues/1531)) ([37ebd31](https://github.com/MustardSeedNetworks/seed/commit/37ebd31ce9bfb45ab34fc62df1aa46cd3193473b))
* **api:** add unified job runner HTTP surface (ADR-0005) ([#1468](https://github.com/MustardSeedNetworks/seed/issues/1468)) ([5c36baf](https://github.com/MustardSeedNetworks/seed/commit/5c36baf1f8bc7359b5551843a5ec79400d409ca6))
* **api:** capability manifest + /__capabilities + route-policy CI gate ([#1412](https://github.com/MustardSeedNetworks/seed/issues/1412)) ([80202b2](https://github.com/MustardSeedNetworks/seed/commit/80202b23a26262c4fe508c03a9f65ff48ef896a0))
* **api:** capability registry + convert Canopy routes (Phase 1) ([#1406](https://github.com/MustardSeedNetworks/seed/issues/1406)) ([7b5dfc9](https://github.com/MustardSeedNetworks/seed/commit/7b5dfc990e223a785c9c2b46c141cfbb94f05ef0))
* **api:** convert Roots/Harvest/Topology/API-token routes to registry ([#1407](https://github.com/MustardSeedNetworks/seed/issues/1407)) ([d9a0dda](https://github.com/MustardSeedNetworks/seed/commit/d9a0dda9048470955af04021c039c656caedff60))
* **api:** convert SAP + Shell routes to registry ([#1410](https://github.com/MustardSeedNetworks/seed/issues/1410)) ([c4cbd36](https://github.com/MustardSeedNetworks/seed/commit/c4cbd36fee35b4b9c47baecdcc8901c1f2cbfd52))
* **api:** convert Update routes to registry ([#1411](https://github.com/MustardSeedNetworks/seed/issues/1411)) ([c2d89a7](https://github.com/MustardSeedNetworks/seed/commit/c2d89a77bae589c1ee79b308ffcc4d22a596d188))
* **api:** expose IncludeNameRes + IncludeProfiling on EngineScanRequest (P7 S4.1c) ([#1502](https://github.com/MustardSeedNetworks/seed/issues/1502)) ([dd7cc62](https://github.com/MustardSeedNetworks/seed/commit/dd7cc624af7f9d3c157d26cab12248936501cb81))
* **api:** make HTTP method + body-limit authoritative in the route registry ([#1530](https://github.com/MustardSeedNetworks/seed/issues/1530)) ([8314d24](https://github.com/MustardSeedNetworks/seed/commit/8314d246c114096fc32d16e5ed42d0e139f3c28e))
* **api:** migrate bluetooth/wifi-discovery/device scans to job kinds (ADR-0005) ([#1474](https://github.com/MustardSeedNetworks/seed/issues/1474)) ([f28a890](https://github.com/MustardSeedNetworks/seed/commit/f28a890ac48d3a8ed331a0de2cc3299487cba8db))
* **api:** migrate discovery engine scan to a job kind (ADR-0005) ([#1473](https://github.com/MustardSeedNetworks/seed/issues/1473)) ([857eb64](https://github.com/MustardSeedNetworks/seed/commit/857eb645018e4bed9d68898bdcac9042a44fc83f))
* **api:** migrate iperf3 client test to a job kind (ADR-0005) ([#1471](https://github.com/MustardSeedNetworks/seed/issues/1471)) ([0fc9f2a](https://github.com/MustardSeedNetworks/seed/commit/0fc9f2a2023e6b0dd4ce7542e76556823eb62db0))
* **api:** migrate speedtest to a unified job kind (ADR-0005) ([#1470](https://github.com/MustardSeedNetworks/seed/issues/1470)) ([5a447f3](https://github.com/MustardSeedNetworks/seed/commit/5a447f312408c2737cbda5887f188c53260193e6))
* **api:** migrate vulnerability scan to a job kind (ADR-0005) ([#1472](https://github.com/MustardSeedNetworks/seed/issues/1472)) ([f937e86](https://github.com/MustardSeedNetworks/seed/commit/f937e8608487942e924fd2fc485a85d55662183d))
* **bluetooth:** Bluetooth visibility UI (card + full-screen device modal) ([#1520](https://github.com/MustardSeedNetworks/seed/issues/1520)) ([8177448](https://github.com/MustardSeedNetworks/seed/commit/8177448148bc5b3528da09da1663c5055288fe4b))
* **bluetooth:** decode manufacturer ID, GATT services, and BLE appearance ([#1517](https://github.com/MustardSeedNetworks/seed/issues/1517)) ([67b2168](https://github.com/MustardSeedNetworks/seed/commit/67b21682ee6d0757086d2dbe16be67b9947c7781))
* **config:** separate credential encryption key from JWTSecret (ADR-0015) ([#1549](https://github.com/MustardSeedNetworks/seed/issues/1549)) ([69e19c5](https://github.com/MustardSeedNetworks/seed/commit/69e19c549fbde40559939e2c8b43f58bb781ce18))
* **contract:** code-first contract decision (ADR-0003 amended) + widen DTO coverage ([#1413](https://github.com/MustardSeedNetworks/seed/issues/1413)) ([d8d70ba](https://github.com/MustardSeedNetworks/seed/commit/d8d70ba5785cde926a8c833bc37917497f7aa33e))
* **contract:** widen DTO coverage batch 2 (7 DTOs); flag nested-defs blocker ([#1414](https://github.com/MustardSeedNetworks/seed/issues/1414)) ([35111cb](https://github.com/MustardSeedNetworks/seed/commit/35111cb1678c6eb750c457478ce8c55583924a61))
* **contract:** widen DTO coverage batch 3 (+9 SAP/network/discovery) ([#1416](https://github.com/MustardSeedNetworks/seed/issues/1416)) ([902eb32](https://github.com/MustardSeedNetworks/seed/commit/902eb32b1d2a888054bda65cbea6282d9bcec268))
* **contract:** widen DTO coverage batch 4 (+10 SAP/network settings) ([#1418](https://github.com/MustardSeedNetworks/seed/issues/1418)) ([140beba](https://github.com/MustardSeedNetworks/seed/commit/140beba6e85c50804e9595f2ba040b43637b20be))
* **contract:** widen DTO coverage batch 5 (+22 health-check DTOs) ([#1419](https://github.com/MustardSeedNetworks/seed/issues/1419)) ([b786040](https://github.com/MustardSeedNetworks/seed/commit/b786040e20f2f4e2a1e75e76f591c2cac2e7ebed))
* **contract:** widen DTO coverage batch 6 (+11 iperf/tools/dns/engine) ([#1420](https://github.com/MustardSeedNetworks/seed/issues/1420)) ([40beeda](https://github.com/MustardSeedNetworks/seed/commit/40beeda8271986723fcaa6d9970c746d50a1067e))
* **contract:** widen DTO coverage batch 7 (+16 users/tokens/update/sso/logs) ([#1421](https://github.com/MustardSeedNetworks/seed/issues/1421)) ([b31b585](https://github.com/MustardSeedNetworks/seed/commit/b31b58572ba90c5bff35a5fa6ee5e1cb7f69c62a))
* **contract:** widen DTO coverage batch 8 (+10 survey) — self-contained set complete ([#1422](https://github.com/MustardSeedNetworks/seed/issues/1422)) ([f50d8f6](https://github.com/MustardSeedNetworks/seed/commit/f50d8f6919b6db406c84a0fbf6bac747320639c0))
* **database:** add proven goose schema baseline (ADR-0006, Phase 5b-1) ([#1477](https://github.com/MustardSeedNetworks/seed/issues/1477)) ([f2d41b0](https://github.com/MustardSeedNetworks/seed/commit/f2d41b0aac898f4ef6e3a508efe97f587b9e700f))
* **database:** durable jobs table + repository (ADR-0005, Phase 5c-1) ([#1481](https://github.com/MustardSeedNetworks/seed/issues/1481)) ([86e71b3](https://github.com/MustardSeedNetworks/seed/commit/86e71b3e82699976c1f2ac217e2c07e3b4860f26))
* **database:** swap migration runner to goose (ADR-0006, Phase 5b-2) ([#1478](https://github.com/MustardSeedNetworks/seed/issues/1478)) ([21ee962](https://github.com/MustardSeedNetworks/seed/commit/21ee9625a04f16cb7cfc6cff1c7a2e77cf4d3ea2))
* **discovery:** emit phase-grained scan progress from the engine (P7 S4.2) ([#1501](https://github.com/MustardSeedNetworks/seed/issues/1501)) ([95254e7](https://github.com/MustardSeedNetworks/seed/commit/95254e73711e34e82091343af2084444c027dc12))
* **discovery:** fold port-scan intensity + timing into the engine (P7 S4.1) ([#1500](https://github.com/MustardSeedNetworks/seed/issues/1500)) ([14839df](https://github.com/MustardSeedNetworks/seed/commit/14839df85faa606f94270dfb67c0c2b4c186ad86))
* **jobs:** add durable Store seam to the runner (ADR-0005, Phase 5c-2) ([#1482](https://github.com/MustardSeedNetworks/seed/issues/1482)) ([4b2dfdb](https://github.com/MustardSeedNetworks/seed/commit/4b2dfdb95a2435742e3880096d53771bd24e413f))
* **jobs:** durable Idempotency-Key store for POST /jobs (ADR-0005, Phase 5c-4) ([#1484](https://github.com/MustardSeedNetworks/seed/issues/1484)) ([e55f2db](https://github.com/MustardSeedNetworks/seed/commit/e55f2db8a8b18f9afd08e2cbae65fd9755b53066))
* **jobs:** wire durable SQLite store into the runner + boot recovery (ADR-0005, Phase 5c-3) ([#1483](https://github.com/MustardSeedNetworks/seed/issues/1483)) ([8ca9eb9](https://github.com/MustardSeedNetworks/seed/commit/8ca9eb9de6df5f03bae50c2a5a1ce3f252612894))
* **jobs:** wire jobs retention into the maintenance loop (ADR-0005, Phase 5c) ([#1485](https://github.com/MustardSeedNetworks/seed/issues/1485)) ([a2e0593](https://github.com/MustardSeedNetworks/seed/commit/a2e059368869e5fc439e54d60d985218dc7b03bb))
* **outbox:** transactional outbox relay for durable event delivery (ADR-0017) ([#1562](https://github.com/MustardSeedNetworks/seed/issues/1562)) ([958b6e8](https://github.com/MustardSeedNetworks/seed/commit/958b6e84f55d8cf421a2a34547051ec1fe11845c))
* **path:** unify L2+L3 path discovery into one ordered timeline ([#1436](https://github.com/MustardSeedNetworks/seed/issues/1436)) ([5f070bd](https://github.com/MustardSeedNetworks/seed/commit/5f070bd45e2f48e4c7ef0c7938556cacbbc186e4))
* **platform:** add in-process domain event bus (ADR-0004) ([#1466](https://github.com/MustardSeedNetworks/seed/issues/1466)) ([8831d8f](https://github.com/MustardSeedNetworks/seed/commit/8831d8f12cc769f0f27e74900fd362f77279b7c4))
* **platform:** add unified async job runner core (ADR-0005) ([#1467](https://github.com/MustardSeedNetworks/seed/issues/1467)) ([1194daf](https://github.com/MustardSeedNetworks/seed/commit/1194daffcf823a4d3b743a143fadfd7ff61cce4b))
* **profiles:** optimistic concurrency via ETag / If-Match (Phase 5) ([#1559](https://github.com/MustardSeedNetworks/seed/issues/1559)) ([558e923](https://github.com/MustardSeedNetworks/seed/commit/558e9239d63fbd9da050f398246fe17ae75cd0dc))
* **profiles:** row_version optimistic-concurrency token (Phase 5 hardening) ([#1561](https://github.com/MustardSeedNetworks/seed/issues/1561)) ([4a1c9d7](https://github.com/MustardSeedNetworks/seed/commit/4a1c9d7be9be9132bf5f3b66258676ad3e4c8407))
* **schema:** publish profile envelope schemas (config as opaque JSON) ([#1465](https://github.com/MustardSeedNetworks/seed/issues/1465)) ([2bd1639](https://github.com/MustardSeedNetworks/seed/commit/2bd16394b2d8db7ffb43c9f580d9caf51ec0ac08))
* **schema:** publish self-contained composer DTO schemas ([#1454](https://github.com/MustardSeedNetworks/seed/issues/1454)) ([902ce46](https://github.com/MustardSeedNetworks/seed/commit/902ce46c90ab934881f2b3f7c358d30604258916))
* **schema:** register config.Config as the code-first Config type (P7 S6.1) ([#1510](https://github.com/MustardSeedNetworks/seed/issues/1510)) ([d9b6008](https://github.com/MustardSeedNetworks/seed/commit/d9b600810c80bdffbc9276d7025cff900f5a4248))
* **schema:** register EngineDiscoveryResponse via the ADR-0008 pure-data exception (P7 S2) ([#1499](https://github.com/MustardSeedNetworks/seed/issues/1499)) ([c8599af](https://github.com/MustardSeedNetworks/seed/commit/c8599af9799014023a63fc8f989bfc7415abc563))
* **schema:** register JobResponse + CreateJobRequest DTOs (P7 S1a) ([#1497](https://github.com/MustardSeedNetworks/seed/issues/1497)) ([49ee599](https://github.com/MustardSeedNetworks/seed/commit/49ee599b674379e08158faa95b9e801f736e153e))
* **settings:** optimistic concurrency via ETag / If-Match (Phase 5) ([#1560](https://github.com/MustardSeedNetworks/seed/issues/1560)) ([e15bfb9](https://github.com/MustardSeedNetworks/seed/commit/e15bfb906a74f7d8958f684f4f3909011263918f))
* **ui:** add jobs client + job event stream hook (P7 S1b) ([#1498](https://github.com/MustardSeedNetworks/seed/issues/1498)) ([9937a20](https://github.com/MustardSeedNetworks/seed/commit/9937a2037e9e9e48d5b83f496616d450179b2668))
* **ui:** add useEnginePhase hook tracking the current scan phase (P7 S3.2a) ([#1504](https://github.com/MustardSeedNetworks/seed/issues/1504)) ([79aca7e](https://github.com/MustardSeedNetworks/seed/commit/79aca7e49f157b93bb39933a43836f3396d7f653))
* **ui:** add useEngineScan hook driving discovery via the jobs spine (P7 S3.1) ([#1503](https://github.com/MustardSeedNetworks/seed/issues/1503)) ([83ea33a](https://github.com/MustardSeedNetworks/seed/commit/83ea33a5a8d4c221bb0b57c1343ef29fa183cdb3))
* **ui:** migrate discovery card onto the engine-scan job (P7 S3.2b) ([#1505](https://github.com/MustardSeedNetworks/seed/issues/1505)) ([cbd634b](https://github.com/MustardSeedNetworks/seed/commit/cbd634b631200d8869dd4b837f68cb66a0258e93))
* **wifi:** 802.11 decoder + airspace model foundation (W1+W2) ([#1526](https://github.com/MustardSeedNetworks/seed/issues/1526)) ([6245b89](https://github.com/MustardSeedNetworks/seed/commit/6245b89de14d3aea7b48cd175fb76be1be3ced0c))
* **wifi:** add deauth/disassoc-flood anomaly rule (w4e) ([#1544](https://github.com/MustardSeedNetworks/seed/issues/1544)) ([ff4ee00](https://github.com/MustardSeedNetworks/seed/commit/ff4ee0028cedfe3ce2f90f2583e787650682fe6e))
* **wifi:** add rogue-ap-on-lan cross-reference rule (w7) ([#1545](https://github.com/MustardSeedNetworks/seed/issues/1545)) ([acf6814](https://github.com/MustardSeedNetworks/seed/commit/acf68142d4c060d20c8d7764188a7d7d2770b28f))
* **wifi:** airspace tree + anomaly stream UI (W6) ([#1539](https://github.com/MustardSeedNetworks/seed/issues/1539)) ([5535585](https://github.com/MustardSeedNetworks/seed/commit/5535585104111e880772782a5dcc03385416561d))
* **wifi:** airspace visibility service (W5a) ([#1533](https://github.com/MustardSeedNetworks/seed/issues/1533)) ([691f644](https://github.com/MustardSeedNetworks/seed/commit/691f6446667c8c949253db102aa8e524421c11a8))
* **wifi:** anomaly catalog + airspace rules (W4b) ([#1532](https://github.com/MustardSeedNetworks/seed/issues/1532)) ([66e0a0f](https://github.com/MustardSeedNetworks/seed/commit/66e0a0fb365bfb88043c6941c405e23f3fe5be6d))
* **wifi:** bss-load + channel-width anomaly rules (W4d) ([#1540](https://github.com/MustardSeedNetworks/seed/issues/1540)) ([2b8d729](https://github.com/MustardSeedNetworks/seed/commit/2b8d72982ffe1b47a648c7d43f792e367e0cf910))
* **wifi:** four buildable-now anomaly rules (W4c) ([#1535](https://github.com/MustardSeedNetworks/seed/issues/1535)) ([988147f](https://github.com/MustardSeedNetworks/seed/commit/988147f4a44b00b06e96ce3fe91347d3af658e0f))
* **wifi:** monitor-mode auto-enablement (W3 follow-up) ([#1538](https://github.com/MustardSeedNetworks/seed/issues/1538)) ([996bc66](https://github.com/MustardSeedNetworks/seed/commit/996bc66806a226aed6417ea8e688eb5555a2bbd6))
* **wifi:** monitor-mode capture producer (W3) ([#1536](https://github.com/MustardSeedNetworks/seed/issues/1536)) ([25f6ea7](https://github.com/MustardSeedNetworks/seed/commit/25f6ea763d316feac4e3c41590de773bf0f4071d))
* **wifi:** pro-gated airspace + anomaly read API (W5b) ([#1534](https://github.com/MustardSeedNetworks/seed/issues/1534)) ([05a5bbf](https://github.com/MustardSeedNetworks/seed/commit/05a5bbfdb787cb055935ebcf4c91f2714e97dbf3))
* **wifi:** regulatory-violation rule (802.11d, 2.4 GHz) ([#1537](https://github.com/MustardSeedNetworks/seed/issues/1537)) ([336a0fa](https://github.com/MustardSeedNetworks/seed/commit/336a0fa3a73f617e6824fa8d8dc1586ddae3f004))
* **wifi:** run the anomaly engine over wi-fi survey samples ([#1543](https://github.com/MustardSeedNetworks/seed/issues/1543)) ([dcee9c8](https://github.com/MustardSeedNetworks/seed/commit/dcee9c821470377c605cd46752c3f2b97fc78851))


### Bug Fixes

* **config:** unify on-disk config format to JSON (seed.json) ([#1528](https://github.com/MustardSeedNetworks/seed/issues/1528)) ([3663f25](https://github.com/MustardSeedNetworks/seed/commit/3663f25090d7801d568bcf8c6e18787902c7dc2f))
* **contract:** gen-types handles nested-type DTOs (unblocks bulk rollout) ([#1415](https://github.com/MustardSeedNetworks/seed/issues/1415)) ([323e0fd](https://github.com/MustardSeedNetworks/seed/commit/323e0fd3f34f4b1697f85079bc9ab78b68cd8e18))
* **database:** enforce owner-only (0600) permissions on the database file ([#1546](https://github.com/MustardSeedNetworks/seed/issues/1546)) ([1f4c989](https://github.com/MustardSeedNetworks/seed/commit/1f4c9892511978ff0a87196fc02e5480ce78e1b6))
* **deploy:** do not auto-open the firewall on install; require opt-in ([#1529](https://github.com/MustardSeedNetworks/seed/issues/1529)) ([437cb00](https://github.com/MustardSeedNetworks/seed/commit/437cb0066ce1e69992e9e786975deddc0a7c1d4e))
* **e2e:** de-flake gateway IPv6/IPv4 test on WebKit ([#1518](https://github.com/MustardSeedNetworks/seed/issues/1518)) ([2a4029e](https://github.com/MustardSeedNetworks/seed/commit/2a4029e682897096dc26aa77deab74bcde856bb0))
* **help:** add HelpDrawer sections for alerts, polling-targets, topology ([#1425](https://github.com/MustardSeedNetworks/seed/issues/1425)) ([758f485](https://github.com/MustardSeedNetworks/seed/commit/758f4857de6d36db7fd3af5b6e5fc7c6aa4d74af))
* **license:** memoize device fingerprint to stop spurious invalidation ([#1523](https://github.com/MustardSeedNetworks/seed/issues/1523)) ([50aabc8](https://github.com/MustardSeedNetworks/seed/commit/50aabc896da33b9603cc13f9e2fb2533fe767981))
* **ui:** bring AlertsPage onto semantic design tokens (P7 S0) ([#1492](https://github.com/MustardSeedNetworks/seed/issues/1492)) ([5826432](https://github.com/MustardSeedNetworks/seed/commit/5826432c45cd61557657fded5e8e87889ba08b19))
* **ui:** bring PollingTargetsPage onto semantic design tokens (P7 S0) ([#1493](https://github.com/MustardSeedNetworks/seed/issues/1493)) ([a295e3e](https://github.com/MustardSeedNetworks/seed/commit/a295e3e8f17ab3d76bf898a4352200e3d1ec3475))
* **ui:** bring TopologyPage onto semantic design tokens (P7 S0) ([#1491](https://github.com/MustardSeedNetworks/seed/issues/1491)) ([36e61e5](https://github.com/MustardSeedNetworks/seed/commit/36e61e5b6a801d545627e7d2bc57dc38ea482227))
* **ui:** rename customer-facing 'Wi-Fi Planning Mode' to 'Wi-Fi Survey Mode' (l3) ([#1569](https://github.com/MustardSeedNetworks/seed/issues/1569)) ([bad996e](https://github.com/MustardSeedNetworks/seed/commit/bad996e1e45aa13004fa3d2f64de55db09eae992))
* **ui:** stop the FAB presenting a partial run as complete (c2) ([#1568](https://github.com/MustardSeedNetworks/seed/issues/1568)) ([07cecb7](https://github.com/MustardSeedNetworks/seed/commit/07cecb7411a16a3596139ba8935ba8f801a480f1))
* **ui:** surface NMS pages in the sidebar + guard nav/route parity (h3) ([#1565](https://github.com/MustardSeedNetworks/seed/issues/1565)) ([c02d9a2](https://github.com/MustardSeedNetworks/seed/commit/c02d9a2637a3c3276d9bc1947a36a8427192b960))
* **ui:** unblock frontend gate (@storybook/react devDep) + repair reports-page e2e ([#1566](https://github.com/MustardSeedNetworks/seed/issues/1566)) ([9dfe217](https://github.com/MustardSeedNetworks/seed/commit/9dfe21748e5e24f8385137d1442fcad5b310f772))

## [0.208.0](https://github.com/krisarmstrong/seed/compare/v0.207.0...v0.208.0) (2026-06-01)


### Features

* **alerts:** load listener rules from db with hot reload ([#1386](https://github.com/krisarmstrong/seed/issues/1386)) ([2b0082c](https://github.com/krisarmstrong/seed/commit/2b0082c4691e324f6ea71a33a8490d8d947c6e3a))
* **alerts:** replay observation state on startup ([#1399](https://github.com/krisarmstrong/seed/issues/1399)) ([06600ee](https://github.com/krisarmstrong/seed/commit/06600eec96ce89966942c815666b7e19ff1e2431))
* **alerts:** time-windowed rule thresholds ([#1398](https://github.com/krisarmstrong/seed/issues/1398)) ([ee2a991](https://github.com/krisarmstrong/seed/commit/ee2a991abe77bcb0ef901c72ccfcbb4e57ec1031))
* **api:** /api/v1/engines admin endpoint (stage a5.8, item 3) ([#1364](https://github.com/krisarmstrong/seed/issues/1364)) ([ef4fa05](https://github.com/krisarmstrong/seed/commit/ef4fa05f9f191db98659fbede66c225bf03db0a8))
* **api:** alert-rule editor (stage a5.10, item 5) ([#1366](https://github.com/krisarmstrong/seed/issues/1366)) ([871414c](https://github.com/krisarmstrong/seed/commit/871414c866de7c626ec906c1af2922bf9b3a8722))
* **api:** list arp bindings via /topology/arp ([#1387](https://github.com/krisarmstrong/seed/issues/1387)) ([9840f66](https://github.com/krisarmstrong/seed/commit/9840f66448cc33fff56e4a95b5dd1e157b3398eb)), closes [#1382](https://github.com/krisarmstrong/seed/issues/1382) [#1367](https://github.com/krisarmstrong/seed/issues/1367)
* **api:** per-tier engine gating (stage a5.9, item 4) ([#1365](https://github.com/krisarmstrong/seed/issues/1365)) ([8f628dc](https://github.com/krisarmstrong/seed/commit/8f628dce708278cd7d0f72a8cb7b73e42c9994cf))
* **engine:** optional reporter interface + /engines status surface ([#1389](https://github.com/krisarmstrong/seed/issues/1389)) ([19c117e](https://github.com/krisarmstrong/seed/commit/19c117e6f5cd4fda799f8a43ba1a86735cb53705))
* **ui:** alert-rules editor page ([#1397](https://github.com/krisarmstrong/seed/issues/1397)) ([af4f85c](https://github.com/krisarmstrong/seed/commit/af4f85c610e0e721ebdcc5e1d87be6499b6552f3))
* **ui:** alerts page — list + ack + resolve (stage a5.7) ([#1363](https://github.com/krisarmstrong/seed/issues/1363)) ([6e2d3e7](https://github.com/krisarmstrong/seed/commit/6e2d3e70774d9ab874b5eade3d4c9663ca8ec448))
* **ui:** polling targets crud page (stage a5.5) ([#1361](https://github.com/krisarmstrong/seed/issues/1361)) ([c3734af](https://github.com/krisarmstrong/seed/commit/c3734af7cd8aa1228707f04ca8f42c92dbb90eb0))
* **ui:** topology page — nodes list + node detail (stage a5.6) ([#1362](https://github.com/krisarmstrong/seed/issues/1362)) ([55de802](https://github.com/krisarmstrong/seed/commit/55de80281393a59a1defd4e2077f8eb2d4a36904))


### Bug Fixes

* **test:** serialize snmptrap tests to avoid upstream gosnmp race ([#1388](https://github.com/krisarmstrong/seed/issues/1388)) ([6280df4](https://github.com/krisarmstrong/seed/commit/6280df4ca4d8fde3a33043b839e8f722da0d1ab5))

## [0.207.0](https://github.com/krisarmstrong/seed/compare/v0.206.0...v0.207.0) (2026-05-31)


### Features

* **api:** server lifecycle via engine.registry (stage a3.5d) ([#1345](https://github.com/krisarmstrong/seed/issues/1345)) ([8930e5b](https://github.com/krisarmstrong/seed/commit/8930e5b32b6a7443a5a22098ae006effeac75fed))
* **engine:** minimal engine interface + lifecycle registry (stage a3.5a) ([#1343](https://github.com/krisarmstrong/seed/issues/1343)) ([7230f08](https://github.com/krisarmstrong/seed/commit/7230f08080f57d9a6f0ffe437fa71a01b3da5005))
* **listener:** syslog udp listener + listener_events persistence (stage a3.5e-1) ([#1347](https://github.com/krisarmstrong/seed/issues/1347)) ([a80136d](https://github.com/krisarmstrong/seed/commit/a80136db157a9a3f1120b5a1ddfe5ab932d48f91))
* **retention:** unified tier-aware retention engine (Stage A2) ([#1330](https://github.com/krisarmstrong/seed/issues/1330)) ([20fd168](https://github.com/krisarmstrong/seed/commit/20fd168d7267b629b01adb3b7b1af70a06b68491))
* **snmp:** arp collector (stage a3.5) ([#1338](https://github.com/krisarmstrong/seed/issues/1338)) ([03efa76](https://github.com/krisarmstrong/seed/commit/03efa76199c75f89a72735ef54681b6438c6ade8))
* **snmp:** bgp4_mib peer collector (stage a3.9) ([#1342](https://github.com/krisarmstrong/seed/issues/1342)) ([1af3269](https://github.com/krisarmstrong/seed/commit/1af32694490987e4a5af860a130b26617820aa94))
* **snmp:** cdp neighbor collector (stage a3.4b) ([#1336](https://github.com/krisarmstrong/seed/issues/1336)) ([2ca9a89](https://github.com/krisarmstrong/seed/commit/2ca9a89b61654f30c112801c73e9e6167c80aa4d))
* **snmp:** collector-chain poller scaffold (stage a3.1) ([#1332](https://github.com/krisarmstrong/seed/issues/1332)) ([e740ae0](https://github.com/krisarmstrong/seed/commit/e740ae018061ff2021ce5b7c3ebbb01ba1ffa41e))
* **snmp:** fdb collector (stage a3.6) ([#1339](https://github.com/krisarmstrong/seed/issues/1339)) ([b5092e5](https://github.com/krisarmstrong/seed/commit/b5092e53bfe72f91079873016fd367f281aeb71d))
* **snmp:** fdp neighbor collector via cdp wrapper (stage a3.4c) ([#1337](https://github.com/krisarmstrong/seed/issues/1337)) ([cfcd0d4](https://github.com/krisarmstrong/seed/commit/cfcd0d4943410d8c85e540da35759c412a9d7b7e))
* **snmp:** gosnmp-backed client factory (stage a3.5c) ([#1346](https://github.com/krisarmstrong/seed/issues/1346)) ([970bd2c](https://github.com/krisarmstrong/seed/commit/970bd2c18d05360624558e05b359e1d2448194dc))
* **snmp:** host_resources collector (stage a3.8) ([#1341](https://github.com/krisarmstrong/seed/issues/1341)) ([7ae707f](https://github.com/krisarmstrong/seed/commit/7ae707fc45285ce8d699ee6ac0c191dd4b04adad))
* **snmp:** if_table collector (stage a3.3) ([#1334](https://github.com/krisarmstrong/seed/issues/1334)) ([d361c53](https://github.com/krisarmstrong/seed/commit/d361c53060ce42e11595b9faa7bc4f7155f37e3d))
* **snmp:** lldp neighbor collector (stage a3.4) ([#1335](https://github.com/krisarmstrong/seed/issues/1335)) ([8ed4f01](https://github.com/krisarmstrong/seed/commit/8ed4f0193452d6fc864b1ddefb615a981016c04e))
* **snmp:** orchestrator + observation persistence (stage a3.5b) ([#1344](https://github.com/krisarmstrong/seed/issues/1344)) ([a6c7cf1](https://github.com/krisarmstrong/seed/commit/a6c7cf1854279428a24321b977aaacc0584cdd15))
* **snmp:** routing collector (stage a3.7) ([#1340](https://github.com/krisarmstrong/seed/issues/1340)) ([129e105](https://github.com/krisarmstrong/seed/commit/129e105a4b9a3d7c98da8db700997d206bb2b201))
* **snmp:** sys_info collector (stage a3.2) ([#1333](https://github.com/krisarmstrong/seed/issues/1333)) ([7acf498](https://github.com/krisarmstrong/seed/commit/7acf4988f27e72fe46d0ed3dac26fbd4fc8a93db))

## [0.206.0](https://github.com/krisarmstrong/seed/compare/v0.205.0...v0.206.0) (2026-05-31)


### Features

* **api:** wire probe engine into server lifecycle (Stage A1.8) ([#1326](https://github.com/krisarmstrong/seed/issues/1326)) ([cbdacac](https://github.com/krisarmstrong/seed/commit/cbdacacc3fd7fb52a48543980b53226e0484fd3d))
* **db:** drop superseded dns_monitors / ssl_monitors / cert_observations (Stage A1.9) ([#1327](https://github.com/krisarmstrong/seed/issues/1327)) ([0fd3e11](https://github.com/krisarmstrong/seed/commit/0fd3e11c6249be7bf428d84e5b4cdaa08fd0c9f4))
* **probe:** engine lifecycle - storage + scheduler + RunNow (Stage A1.3b) ([#1325](https://github.com/krisarmstrong/seed/issues/1325)) ([0ed1138](https://github.com/krisarmstrong/seed/commit/0ed113883b274fbb00e467a794086123c06929a6))
* **probe:** ping checker via TCP fallback (Stage A1.7 - 1 of N) ([#1328](https://github.com/krisarmstrong/seed/issues/1328)) ([93590dc](https://github.com/krisarmstrong/seed/commit/93590dc1be42295aa6fa6f30e4fa94781eb1c382))
* V1.0 unified architecture foundation (Stage A0 + A1.1-A1.5) ([#1323](https://github.com/krisarmstrong/seed/issues/1323)) ([6004b02](https://github.com/krisarmstrong/seed/commit/6004b02d3e1ae4e8237014206151259c5de83ea0))


### Bug Fixes

* **e2e:** drop .or() in 401 test - strict-mode violation when both render ([#1322](https://github.com/krisarmstrong/seed/issues/1322)) ([d00719b](https://github.com/krisarmstrong/seed/commit/d00719b357740e719e3643ed679426d0b571d44e))
* **e2e:** drop top-level theme-toggle smoke test from seed ([#1297](https://github.com/krisarmstrong/seed/issues/1297)) ([f4d44ea](https://github.com/krisarmstrong/seed/commit/f4d44ea25b55571f06e33d758cab00279a0f4b14))
* **e2e:** mock /api/v1/sap/gateway + add NetworkPage help + race-free FAB keyboard test ([#1321](https://github.com/krisarmstrong/seed/issues/1321)) ([3443d99](https://github.com/krisarmstrong/seed/commit/3443d9974481f6309ddd1f92f91a263e188692f2))
* **e2e:** override storageState for 4 login-form error-scenarios ([#1316](https://github.com/krisarmstrong/seed/issues/1316)) ([26451bb](https://github.com/krisarmstrong/seed/commit/26451bbed33f5121d6dc2311cfb4948f35bf5373))
* **e2e:** route gateway + dashboard card tests to the right pages ([#1315](https://github.com/krisarmstrong/seed/issues/1315)) ([0484ce2](https://github.com/krisarmstrong/seed/commit/0484ce2655684ae6e88cf612fee81b0b58bfebaf))
* **ui:** logout synchronously clears legacy localStorage keys ([#1317](https://github.com/krisarmstrong/seed/issues/1317)) ([7233b8c](https://github.com/krisarmstrong/seed/commit/7233b8c5d6ba2a6aec1a0c601f6affc19f835291))

## [0.205.0](https://github.com/krisarmstrong/seed/compare/v0.204.0...v0.205.0) (2026-05-30)


### Features

* **api:** gate /events SSE on live_telemetry — close Pro revenue leak ([#1278](https://github.com/krisarmstrong/seed/issues/1278)) ([f08b8ef](https://github.com/krisarmstrong/seed/commit/f08b8ef17b999c88af341ccd5d382d1d1f5a09aa))
* **api:** gate wifi_roam_analysis on survey response — close revenue leak ([#1280](https://github.com/krisarmstrong/seed/issues/1280)) ([bdec2ff](https://github.com/krisarmstrong/seed/commit/bdec2ff922725d83ca737e97a876415a87ebe016))

## [0.204.0](https://github.com/krisarmstrong/seed/compare/v0.203.0...v0.204.0) (2026-05-29)


### Features

* **a11y:** axe-core test harness + DiscoveryModal clear-button label ([#1272](https://github.com/krisarmstrong/seed/issues/1272)) ([8d8913d](https://github.com/krisarmstrong/seed/commit/8d8913d46f15c37f0964c030f33cadc84007bb8d))
* **cli:** example blocks on all commands + help completeness test ([#1273](https://github.com/krisarmstrong/seed/issues/1273)) ([8cadbb1](https://github.com/krisarmstrong/seed/commit/8cadbb109307a2e41755efa591695b8e9285472e))
* **help:** add path/reports/logs sections + route-coverage test ([#1274](https://github.com/krisarmstrong/seed/issues/1274)) ([5fc112f](https://github.com/krisarmstrong/seed/commit/5fc112fb779f503039032bf3e4835b18e4a7605c))
* **i18n:** en/es key parity + DNT compliance test ([#1276](https://github.com/krisarmstrong/seed/issues/1276)) ([7c0dc9f](https://github.com/krisarmstrong/seed/commit/7c0dc9f6fc54a81bded5c26745138099680382bb))

## [0.203.0](https://github.com/krisarmstrong/seed/compare/v0.202.1...v0.203.0) (2026-05-29)


### Features

* **api:** enforce viewer read-only via writeGated route wrapper ([#1226](https://github.com/krisarmstrong/seed/issues/1226)) ([#1265](https://github.com/krisarmstrong/seed/issues/1265)) ([b99dae0](https://github.com/krisarmstrong/seed/commit/b99dae0409e68379185b3a9fddd16d49c661c041))
* **api:** per-token role scope for personal-access tokens ([#1255](https://github.com/krisarmstrong/seed/issues/1255)) ([#1268](https://github.com/krisarmstrong/seed/issues/1268)) ([a5078dd](https://github.com/krisarmstrong/seed/commit/a5078dd8ec126d3163225fe8f08becd231be72e4))
* **api:** structured audit log for authz denials ([#1257](https://github.com/krisarmstrong/seed/issues/1257)) ([#1271](https://github.com/krisarmstrong/seed/issues/1271)) ([47bdac6](https://github.com/krisarmstrong/seed/commit/47bdac6ff46e188ee60c748b682a5a26d379a737))
* **config:** refuse CORS `*` origin at startup ([#1256](https://github.com/krisarmstrong/seed/issues/1256)) ([#1269](https://github.com/krisarmstrong/seed/issues/1269)) ([32cd690](https://github.com/krisarmstrong/seed/commit/32cd690973cb4009f8e7bfbf22b554846d27b6df))
* **ui:** role-based write gating with RoleContext + WriteGate ([#1254](https://github.com/krisarmstrong/seed/issues/1254)) ([#1267](https://github.com/krisarmstrong/seed/issues/1267)) ([04a4b35](https://github.com/krisarmstrong/seed/commit/04a4b35c311710d5008627181f541c940701c933))
* **ui:** wrap SettingsDrawer in ReadOnlyView for viewer role ([#1254](https://github.com/krisarmstrong/seed/issues/1254) follow-up) ([#1270](https://github.com/krisarmstrong/seed/issues/1270)) ([2aa3b6d](https://github.com/krisarmstrong/seed/commit/2aa3b6d97d304e544bf745fabbdc371d78f24eb0))

## [0.202.1](https://github.com/krisarmstrong/seed/compare/v0.202.0...v0.202.1) (2026-05-29)


### Bug Fixes

* **ui:** replace undefined bg-surface-secondary token in SLA card ([#1253](https://github.com/krisarmstrong/seed/issues/1253)) ([5d4a922](https://github.com/krisarmstrong/seed/commit/5d4a9228c80546501eadcd678869721a99fd524d))

## [0.202.0](https://github.com/krisarmstrong/seed/compare/v0.201.2...v0.202.0) (2026-05-29)


### Features

* **ui:** establish semantic design-token foundation ([#1246](https://github.com/krisarmstrong/seed/issues/1246)) ([49013de](https://github.com/krisarmstrong/seed/commit/49013de44b56ee8bd161035b6d21edca5aef6e89))


### Bug Fixes

* **ui:** fix stale brand green in canvas/PDF; wire canvas markers to tokens ([#1249](https://github.com/krisarmstrong/seed/issues/1249)) ([055c923](https://github.com/krisarmstrong/seed/commit/055c9237d0d9aa7901672852defca1ce1c4a589d))
* **ui:** repair token-discipline guard and close remaining color leaks ([#1251](https://github.com/krisarmstrong/seed/issues/1251)) ([6dd96c0](https://github.com/krisarmstrong/seed/commit/6dd96c0c55a7f1c02cb460a6d9db08bca4cc6c7b))

## [0.201.2](https://github.com/krisarmstrong/seed/compare/v0.201.1...v0.201.2) (2026-05-29)


### Bug Fixes

* **security:** rate-limit /auth/refresh ([#1224](https://github.com/krisarmstrong/seed/issues/1224)) ([#1243](https://github.com/krisarmstrong/seed/issues/1243)) ([641b79d](https://github.com/krisarmstrong/seed/commit/641b79dd4e9f73a956cd01033c5c62414ae156ed))

## [0.201.1](https://github.com/krisarmstrong/seed/compare/v0.201.0...v0.201.1) (2026-05-28)


### Bug Fixes

* **ui:** replace broken help modal with data-driven help drawer ([#43](https://github.com/krisarmstrong/seed/issues/43)) ([#1239](https://github.com/krisarmstrong/seed/issues/1239)) ([11384f3](https://github.com/krisarmstrong/seed/commit/11384f3ad2aa3a3bf73065d5648e45300bb0686e))
* **ui:** settings drawer focus trap — drop stopPropagation that defeated it ([#1240](https://github.com/krisarmstrong/seed/issues/1240)) ([41c2cbd](https://github.com/krisarmstrong/seed/commit/41c2cbd6498cb98118af85537d20fa18c2e10e0c))

## [0.201.0](https://github.com/krisarmstrong/seed/compare/v0.200.0...v0.201.0) (2026-05-28)


### Features

* **ui:** converge settings drawer shell — focus trap + slide-in (Phase 3c) ([#1236](https://github.com/krisarmstrong/seed/issues/1236)) ([50da4e0](https://github.com/krisarmstrong/seed/commit/50da4e0ae736920786b4193a755161f5d115f131))
* **ui:** converge Tooltip to the shared text/side design (Phase 3a) ([#1235](https://github.com/krisarmstrong/seed/issues/1235)) ([f04b9c4](https://github.com/krisarmstrong/seed/commit/f04b9c4cfe1610559e60806623b7db980691bf39))


### Bug Fixes

* **ui:** re-sync shell from stem — sidebar shows the product name ([#1233](https://github.com/krisarmstrong/seed/issues/1233)) ([2a4641b](https://github.com/krisarmstrong/seed/commit/2a4641b367ccf17e38c57d8e67ed28f01dca054f))

## [0.200.0](https://github.com/krisarmstrong/seed/compare/v0.199.0...v0.200.0) (2026-05-28)


### Features

* **interfaces:** settings ui for multi_interface ([#1210](https://github.com/krisarmstrong/seed/issues/1210)) ([4e6de69](https://github.com/krisarmstrong/seed/commit/4e6de694eb96dbfcf514346af74db12cb86c445d))
* **netif:** linkmonitor pool for multi_interface fan-out ([#1219](https://github.com/krisarmstrong/seed/issues/1219)) ([b2df3fb](https://github.com/krisarmstrong/seed/commit/b2df3fb53dfa74098a5ac936d70e45db07fe8252))
* **seed#1191:** multi_user CRUD + schema hardening + SSO columns ([#1204](https://github.com/krisarmstrong/seed/issues/1204)) ([5c3c6b9](https://github.com/krisarmstrong/seed/commit/5c3c6b9f23060b2e1b29d9f0684ffedc98aa1e37))
* **sso:** gate settings PUT and sync IdP users on callback ([#1207](https://github.com/krisarmstrong/seed/issues/1207)) ([2427d4c](https://github.com/krisarmstrong/seed/commit/2427d4c27faf7d67e0a63ffd3d6ed2f744e19190))
* **ui:** sync canonical shell from stem (Phase 1) ([#1222](https://github.com/krisarmstrong/seed/issues/1222)) ([271f5f4](https://github.com/krisarmstrong/seed/commit/271f5f4068a77f99998575f5d727bfbf45a47f44))
* **users:** settings ui for multi_user crud ([#1208](https://github.com/krisarmstrong/seed/issues/1208)) ([2e3af3d](https://github.com/krisarmstrong/seed/commit/2e3af3d464aae6d26fec0ddfd9eb4d675658c701))


### Bug Fixes

* **e2e:** repoint seed specs to sidebar Settings/Help after Phase 2 ([#1231](https://github.com/krisarmstrong/seed/issues/1231)) ([1c4299d](https://github.com/krisarmstrong/seed/commit/1c4299d8c75353c584f282df9b027309bbe4d2de))
* **help-modal:** add esc handler + testid; fix e2e selectors ([#1228](https://github.com/krisarmstrong/seed/issues/1228)) ([e1a74a5](https://github.com/krisarmstrong/seed/commit/e1a74a536206bee7ec5ad3d929fd84387aeeaa26))
* **ui:** re-sync shell from stem to pull page-header-title testid ([#1230](https://github.com/krisarmstrong/seed/issues/1230)) ([7092857](https://github.com/krisarmstrong/seed/commit/7092857d3c7f20eec93300628d7bf847fe91af3e))
* **ui:** users settings TS error blocking CI strict tsc check ([#1229](https://github.com/krisarmstrong/seed/issues/1229)) ([a0b5ced](https://github.com/krisarmstrong/seed/commit/a0b5ced1aa69fef87e33c411783b140ca2a59d37))

## [0.199.0](https://github.com/krisarmstrong/seed/compare/v0.198.0...v0.199.0) (2026-05-27)


### Features

* **i18n:** add useLocale hook + migrate VulnerabilitySettings plural ([#1200](https://github.com/krisarmstrong/seed/issues/1200)) ([f8ad517](https://github.com/krisarmstrong/seed/commit/f8ad517ab43c6ce7c6bd582dc3cb735ec6f65eeb))
* **i18n:** port shared validator + check-keys + add phase 6 i18n tests ([#1203](https://github.com/krisarmstrong/seed/issues/1203)) ([46379ff](https://github.com/krisarmstrong/seed/commit/46379ffa7d55021d5b4eabec04557e75cd3e59fe))
* **license:** mirror keygen v2.2.0 — add sso + drop legacy multi_site/starter multi_interface ([#1197](https://github.com/krisarmstrong/seed/issues/1197)) ([726668e](https://github.com/krisarmstrong/seed/commit/726668e19ef0bb6b3ba74ad7b7a32b1f68077ee0))
* **seed#1192:** multi_interface gate + Ethernet[] / WiFiList[] config ([#1206](https://github.com/krisarmstrong/seed/issues/1206)) ([59fd51d](https://github.com/krisarmstrong/seed/commit/59fd51d6b775c2c74b7c38b34ad41ac9f6b9ff73))
* **seed#1196:** wire multi_client gate on profile-create paths ([#1205](https://github.com/krisarmstrong/seed/issues/1205)) ([2a1b2e7](https://github.com/krisarmstrong/seed/commit/2a1b2e7ae1ab56fa48103abde0406731379e24aa))


### Bug Fixes

* **e2e:** add header-logout testid + retire SVG class fallback in auth-complete ([#1177](https://github.com/krisarmstrong/seed/issues/1177)) ([1a6c9ef](https://github.com/krisarmstrong/seed/commit/1a6c9ef729dd5e3ecfeb44b0cffc9ed1b3fb2701))
* **e2e:** isolate responsive logout tests so they don't poison shared storageState ([#1176](https://github.com/krisarmstrong/seed/issues/1176)) ([935f318](https://github.com/krisarmstrong/seed/commit/935f318a96add5823e80e275088412d9e4a2c51d))
* **e2e:** remove garbage JS in theme-and-help.spec.ts (closes [#1169](https://github.com/krisarmstrong/seed/issues/1169)) ([#1171](https://github.com/krisarmstrong/seed/issues/1171)) ([7cd0d74](https://github.com/krisarmstrong/seed/commit/7cd0d74ca7910fe3e078583cb0b7eead63fe3ee3))
* **e2e:** replace brittle text regexes with stable id selectors (Category B) ([#1174](https://github.com/krisarmstrong/seed/issues/1174)) ([879ee93](https://github.com/krisarmstrong/seed/commit/879ee93fd7fef910620307b9d2526a37e533dc48))
* **e2e:** replace per-page H1 heading regexes with getByTestId (Category C) ([#1173](https://github.com/krisarmstrong/seed/issues/1173)) ([484f6a4](https://github.com/krisarmstrong/seed/commit/484f6a49c6d786d2099aa50afc2ee847e23647ad))
* **e2e:** replace remaining settings-drawer text regexes with testid ([#1179](https://github.com/krisarmstrong/seed/issues/1179)) ([bb36015](https://github.com/krisarmstrong/seed/commit/bb36015816d9e410609b1fb12f2999b7a46ff5ff))
* **e2e:** rewrite global-setup to run login in a single chromium context ([#1172](https://github.com/krisarmstrong/seed/issues/1172)) ([385220c](https://github.com/krisarmstrong/seed/commit/385220cc6bd90f0619f1624fb1d43dd93b1773d6))
* **e2e:** rewrite system-theme test with colorScheme emulation (real assertion) ([#1189](https://github.com/krisarmstrong/seed/issues/1189)) ([6dc2c80](https://github.com/krisarmstrong/seed/commit/6dc2c80c67330db25d820142e7b6f39e0fd1515d))
* **e2e:** sync FAB tests on data-running attribute, not animate-spin (Category D) ([#1175](https://github.com/krisarmstrong/seed/issues/1175)) ([e939137](https://github.com/krisarmstrong/seed/commit/e9391370833c5267dddc8e6b3bf3a3991818bb43))
* **e2e:** use #profile-modal-title id for profile-management modal assertion ([#1178](https://github.com/krisarmstrong/seed/issues/1178)) ([e7fec8d](https://github.com/krisarmstrong/seed/commit/e7fec8d734cd3a3ccc7247f4b050928af28889be))
* **i18n:** replace banned 'open source' with 'source-available' per CLAUDE.md ([#1184](https://github.com/krisarmstrong/seed/issues/1184)) ([ac207b3](https://github.com/krisarmstrong/seed/commit/ac207b3ced7c666f7c4293256505e70dbe0868f8))
* **i18n:** resolve 329 t() calls referencing missing EN locale keys ([#1211](https://github.com/krisarmstrong/seed/issues/1211)) ([3502972](https://github.com/krisarmstrong/seed/commit/3502972c501803da7adee56a74833abb9279d279))
* **i18n:** update document.lang on locale change for a11y ([#1186](https://github.com/krisarmstrong/seed/issues/1186)) ([c357405](https://github.com/krisarmstrong/seed/commit/c3574055da426d1b55244c23c69a59385614daa8))

## [0.198.0](https://github.com/krisarmstrong/seed/compare/v0.197.1...v0.198.0) (2026-05-26)


### Features

* **i18n:** add errors.license.* keys for tier-gating UI ([#1160](https://github.com/krisarmstrong/seed/issues/1160)) ([7392e31](https://github.com/krisarmstrong/seed/commit/7392e3179538406e62ef95ce25f9cb95a7cd9e2e))
* **license:** add feature-gating framework ([#1153](https://github.com/krisarmstrong/seed/issues/1153)) ([cc6a1fa](https://github.com/krisarmstrong/seed/commit/cc6a1fa9298ff24c17f93e1f4252ce2da863d19a))
* **license:** gate /harvest/export and ReportsPage on export_csv_json (PR-B2) ([#1156](https://github.com/krisarmstrong/seed/issues/1156)) ([a41567a](https://github.com/krisarmstrong/seed/commit/a41567a5eb2c99f1bbad92eb91da86ded2880109))
* **license:** gate /sap/health-checks/anomalies on anomaly_detection (PR-B3) ([#1158](https://github.com/krisarmstrong/seed/issues/1158)) ([dff5269](https://github.com/krisarmstrong/seed/commit/dff5269ea815128b809b2175ebb9845cb9cccce1))
* **license:** gate AirMapper baseline-diff import behind Pro tier (PR-B1) ([#1157](https://github.com/krisarmstrong/seed/issues/1157)) ([05ef48b](https://github.com/krisarmstrong/seed/commit/05ef48b931d46d957231d9fb4957db80084c1c4a))
* **license:** gate path_analysis (Roots) behind Pro tier (PR-B5) ([#1155](https://github.com/krisarmstrong/seed/issues/1155)) ([550f088](https://github.com/krisarmstrong/seed/commit/550f0882923117fa6eae050e4b9d713b6ffacff9))
* **license:** gate shell active-scan endpoints on compliance_advanced (PR-B4) ([#1159](https://github.com/krisarmstrong/seed/issues/1159)) ([e43dab1](https://github.com/krisarmstrong/seed/commit/e43dab17e516f3193862fe0924f652840a5353ed))


### Bug Fixes

* **e2e:** bulk-replace brittle heading regexes with getByTestId ([#1162](https://github.com/krisarmstrong/seed/issues/1162)) ([22f9c06](https://github.com/krisarmstrong/seed/commit/22f9c067623f14036b90f2a24704456ec9efbb74))
* **e2e:** use data-testid for auth login + page header selectors ([#1161](https://github.com/krisarmstrong/seed/issues/1161)) ([f6a0848](https://github.com/krisarmstrong/seed/commit/f6a0848a48cdae0a6d5a28f55f04290cc3972c50))
* **ui:** Add data-testid card + update e2e selector (kill last pre-existing E2E flake) ([#1154](https://github.com/krisarmstrong/seed/issues/1154)) ([e5293dc](https://github.com/krisarmstrong/seed/commit/e5293dcd871907ca3b9a4f3c25fca640a88cdbb2))

## [0.197.1](https://github.com/krisarmstrong/seed/compare/v0.197.0...v0.197.1) (2026-05-26)


### Bug Fixes

* **e2e,test:** repair v1-API URL drift + EventSource polyfill ([#1146](https://github.com/krisarmstrong/seed/issues/1146)) ([972f1e3](https://github.com/krisarmstrong/seed/commit/972f1e3fec89e74ed1dabd16eeb7fc214ec8b478))
* **license:** add RWMutex to Manager for safe concurrent access ([#1152](https://github.com/krisarmstrong/seed/issues/1152)) ([810cfd9](https://github.com/krisarmstrong/seed/commit/810cfd9ead9d4d13b17e1a43909f4e5798f0bfcc))
* **scripts:** clean up all shellcheck warnings + pin severity=warning ([#1144](https://github.com/krisarmstrong/seed/issues/1144)) ([0be82a4](https://github.com/krisarmstrong/seed/commit/0be82a4e80f9f9683debf69563e5baea7a0ab500))

## [0.197.0](https://github.com/krisarmstrong/seed/compare/v0.196.0...v0.197.0) (2026-05-25)


### Features

* **api:** add decodeJSONStrict + HandlerContext.DecodeJSONOrFail ([#1125](https://github.com/krisarmstrong/seed/issues/1125)) ([15cd859](https://github.com/krisarmstrong/seed/commit/15cd8590b96f39931a69f2d29091e3aae9449fe2)), closes [#1100](https://github.com/krisarmstrong/seed/issues/1100)
* **api:** add go-playground/validator + tags on hot DTOs ([#1132](https://github.com/krisarmstrong/seed/issues/1132)) ([15a2ce1](https://github.com/krisarmstrong/seed/commit/15a2ce185f3ca6a980d406fd66d7987102fd82cd))
* **api:** port invopop/jsonschema generator from NIAC ([#1135](https://github.com/krisarmstrong/seed/issues/1135)) ([6babf2c](https://github.com/krisarmstrong/seed/commit/6babf2cdfc6bc755498577cfec12c6c14fe32d4b))
* **canopy/survey:** validate AirMapper .serial JSON with valibot ([#1133](https://github.com/krisarmstrong/seed/issues/1133)) ([2839cec](https://github.com/krisarmstrong/seed/commit/2839cec96d73688af3d378889f37f000927c8a05)), closes [#1106](https://github.com/krisarmstrong/seed/issues/1106)
* **ui:** generate TypeScript types from JSON Schemas ([#1137](https://github.com/krisarmstrong/seed/issues/1137)) ([c259663](https://github.com/krisarmstrong/seed/commit/c25966389b37158379f9a5e355cf238932079451))
* **ui:** validate SSE frames with valibot in useSse ([#1134](https://github.com/krisarmstrong/seed/issues/1134)) ([8b92a0d](https://github.com/krisarmstrong/seed/commit/8b92a0d2229b17566c0773d2661612ec1aad9377)), closes [#1107](https://github.com/krisarmstrong/seed/issues/1107)


### Bug Fixes

* **ci:** add .gitkeep to internal/api/ui + remove vite emptyOutDir ([#1118](https://github.com/krisarmstrong/seed/issues/1118)) ([2459473](https://github.com/krisarmstrong/seed/commit/2459473ef2d724445063fa19697a4bafb1e08b81))
* **ci:** inject UIBuildHash ldflag (Universal Build Contract) ([#1119](https://github.com/krisarmstrong/seed/issues/1119)) ([a1b1bc3](https://github.com/krisarmstrong/seed/commit/a1b1bc30ce9e5c8d2dc9e8646ecc3e5df8a5f330))
* **ci:** verify UIBuildHash embedded in built binary ([#1123](https://github.com/krisarmstrong/seed/issues/1123)) ([393e9f5](https://github.com/krisarmstrong/seed/commit/393e9f52db51d4e8ef6eea3c69fd572dec3b9ab7))
* **docs:** correct PR template 'cd web' -&gt; 'cd ui' ([#1120](https://github.com/krisarmstrong/seed/issues/1120)) ([cd6c4b6](https://github.com/krisarmstrong/seed/commit/cd6c4b693b352ebf584e4a7b20c69c88ce4062d0))
* **ui:** enable erasableSyntaxOnly + refactor logger.ts TS-only syntax ([#1127](https://github.com/krisarmstrong/seed/issues/1127)) ([b667587](https://github.com/krisarmstrong/seed/commit/b667587664de09378366b41ce7159d7b480a3384)), closes [#1122](https://github.com/krisarmstrong/seed/issues/1122)

## [0.196.0](https://github.com/krisarmstrong/seed/compare/v0.195.0...v0.196.0) (2026-05-25)


### Features

* **api:** add personal-access tokens for programmatic API access (Pro tier) ([#1096](https://github.com/krisarmstrong/seed/issues/1096)) ([15bb20c](https://github.com/krisarmstrong/seed/commit/15bb20c204b742f37b037c8b93ae947c3d55a53b))
* **license:** add offline license framework with trial and keygen contract ([#1095](https://github.com/krisarmstrong/seed/issues/1095)) ([3f23b27](https://github.com/krisarmstrong/seed/commit/3f23b2704d853722edbcb8f918f5c630d863c1f2))
* **ui:** add Settings → API Tokens panel with Pro-gated mint UX ([#1098](https://github.com/krisarmstrong/seed/issues/1098)) ([ffeac16](https://github.com/krisarmstrong/seed/commit/ffeac162d582970bebd9123abc3eacb68efecbc3))


### Bug Fixes

* **netif:** parallelize per-interface scoring in detector.DetectAll ([#1097](https://github.com/krisarmstrong/seed/issues/1097)) ([eed5979](https://github.com/krisarmstrong/seed/commit/eed59799be5e7c12fafdf41bd21ac0b6237e3040))
* **security:** Real-code fixes for all 27 seed gosec issues ([#1070](https://github.com/krisarmstrong/seed/issues/1070)) ([#1090](https://github.com/krisarmstrong/seed/issues/1090)) ([8bb115e](https://github.com/krisarmstrong/seed/commit/8bb115e0a3941e97d1648ad92e9c09b5509d44be))
* **ui:** add data-testid + aria-label to theme quick-toggle button ([#1109](https://github.com/krisarmstrong/seed/issues/1109)) ([6f44a6f](https://github.com/krisarmstrong/seed/commit/6f44a6f2faef3bbe384bcce5806cf95f00413698))


### Performance Improvements

* **e2e:** bump CI workers 1-&gt;2 and retries 2-&gt;1 ([#1072](https://github.com/krisarmstrong/seed/issues/1072)) ([#1080](https://github.com/krisarmstrong/seed/issues/1080)) ([fcefbe6](https://github.com/krisarmstrong/seed/commit/fcefbe682bcc11226eab961813aa9e07d050634c))

## [0.195.0](https://github.com/krisarmstrong/seed/compare/v0.194.0...v0.195.0) (2026-05-22)


### Features

* **theme:** adopt botanical-earth surface palette (Phase 4) ([ba20ddd](https://github.com/krisarmstrong/seed/commit/ba20dddd9aa47252665ede817e99e31a4fc54fa4))
* **theme:** Apply 2026-05-22 brand audit — botanical-earth + Seed identity ([4b041d8](https://github.com/krisarmstrong/seed/commit/4b041d805a1ef89ffc7dd3a7e8094a1ee9fb81dc))
* **theme:** fix button contrast against constant brand anchor (Phase 7) ([16650ed](https://github.com/krisarmstrong/seed/commit/16650ed81c5c18828fd4fdefd0d2016f90e300ec))
* **theme:** lock brand anchor to seed-500 constant across modes (Phase 5) ([2693d6e](https://github.com/krisarmstrong/seed/commit/2693d6e0b2be617c985d565d5537ccf3b282bae1))
* **theme:** self-host Inter + JetBrains Mono via [@fontsource-variable](https://github.com/fontsource-variable) (Phase 2) ([cbd0268](https://github.com/krisarmstrong/seed/commit/cbd026822ef29cc5cc53d70f81eac420d68914a8))
* **theme:** swap status palette to canonical brand-tied anchors (Phase 1) ([4334ef4](https://github.com/krisarmstrong/seed/commit/4334ef4520f364cbedbf2cb85c3eaff43187714c))

## [0.194.0](https://github.com/krisarmstrong/seed/compare/v0.193.1...v0.194.0) (2026-05-22)


### Features

* **stories:** activate a11y addon + fix decorator (Wave 5 / seed-W5-3) ([#1063](https://github.com/krisarmstrong/seed/issues/1063)) ([97fbc3e](https://github.com/krisarmstrong/seed/commit/97fbc3ec7b074734d93969a9072f6e916a8936db))


### Bug Fixes

* **auth+e2e:** stabilize auth.spec + dashboard.spec for [@smoke](https://github.com/smoke) ([#1053](https://github.com/krisarmstrong/seed/issues/1053)) ([#1065](https://github.com/krisarmstrong/seed/issues/1065)) ([d48943b](https://github.com/krisarmstrong/seed/commit/d48943bc4043e7c871f9d5f421dac122072e6cd7))
* **e2e:** Broaden smoke filter to exclude 404/Failed-to-load-resource ([#1068](https://github.com/krisarmstrong/seed/issues/1068)) ([571ef76](https://github.com/krisarmstrong/seed/commit/571ef7618cd5724d972d70c7f670bf00082f4deb))
* **lint:** disable noShadow in stories (Storybook decorator convention) ([#1067](https://github.com/krisarmstrong/seed/issues/1067)) ([3822887](https://github.com/krisarmstrong/seed/commit/38228872e13c66b72c8fb48ce10d73a240626838))

## [0.193.1](https://github.com/krisarmstrong/seed/compare/v0.193.0...v0.193.1) (2026-05-21)


### Bug Fixes

* **e2e:** bypass login modal for non-auth specs and lift rate limit for CI ([#1049](https://github.com/krisarmstrong/seed/issues/1049)) ([bd98c6b](https://github.com/krisarmstrong/seed/commit/bd98c6b6f2571b20e97e516c5643a15b8018d55b))
* **ui:** Version every backend fetch under /api/v1/* (P0 silent auth failure) ([#1050](https://github.com/krisarmstrong/seed/issues/1050)) ([0c6b68a](https://github.com/krisarmstrong/seed/commit/0c6b68a9d6cdbd92d6340106ea71477d1bc74aad))

## [0.193.0](https://github.com/krisarmstrong/seed/compare/v0.192.0...v0.193.0) (2026-05-20)


### Features

* **auth:** Argon2id password hashing + zxcvbn strength + HIBP breach check (Wave 2) ([#1047](https://github.com/krisarmstrong/seed/issues/1047)) ([b746151](https://github.com/krisarmstrong/seed/commit/b746151fe0d79ff20f654f831a63a28f7ed79709))
* **auth:** argon2id totp mfa + webauthn passkeys (wave 3) ([#1048](https://github.com/krisarmstrong/seed/issues/1048)) ([d8749b7](https://github.com/krisarmstrong/seed/commit/d8749b7ab386dc22adbb332f002f9e330ed6ebd9))
* **ci:** Add provenance_only mode for SLSA backfill ([#75](https://github.com/krisarmstrong/seed/issues/75)) ([#1040](https://github.com/krisarmstrong/seed/issues/1040)) ([ef45f8f](https://github.com/krisarmstrong/seed/commit/ef45f8f056cdec57aa58c3f18c5ef92a0af5ec13))
* **tls:** Trust-store install UX + cert fingerprint + 308 redirect (Wave 1) ([#1046](https://github.com/krisarmstrong/seed/issues/1046)) ([efcfa12](https://github.com/krisarmstrong/seed/commit/efcfa12e3970884fe46d72c5db172fbbf6c1356e))


### Bug Fixes

* **ci:** add target_tag input to SLSA backfill ([#75](https://github.com/krisarmstrong/seed/issues/75) follow-up) ([#1042](https://github.com/krisarmstrong/seed/issues/1042)) ([c946a2c](https://github.com/krisarmstrong/seed/commit/c946a2cce5f3085141cb102c6baba8fd5ae45f45))
* **ci:** unescape apostrophe in target_tag description ([#1043](https://github.com/krisarmstrong/seed/issues/1043)) ([2238293](https://github.com/krisarmstrong/seed/commit/22382930a919a441316627b4e7b7b5f96e77e22a))

## [0.192.0](https://github.com/krisarmstrong/seed/compare/v0.191.2...v0.192.0) (2026-05-19)


### Features

* Graceful port fallback when canonical port is in use ([#69](https://github.com/krisarmstrong/seed/issues/69)) ([#1038](https://github.com/krisarmstrong/seed/issues/1038)) ([4327f97](https://github.com/krisarmstrong/seed/commit/4327f97cac3e939ed8c8626bd35a5eb73d55539f))


### Bug Fixes

* **setup:** point wizard at /api/v1/setup/* (was /api/setup/*) ([#1033](https://github.com/krisarmstrong/seed/issues/1033)) ([bc724bd](https://github.com/krisarmstrong/seed/commit/bc724bdf510c5794b092f71513d1b70b5cd46933))

## [0.191.2](https://github.com/krisarmstrong/seed/compare/v0.191.1...v0.191.2) (2026-05-18)


### Bug Fixes

* **release:** Correct stale goreleaser-config header comments ([#1026](https://github.com/krisarmstrong/seed/issues/1026)) ([3eae712](https://github.com/krisarmstrong/seed/commit/3eae712a7852a5449886dbb6d6f0b14784cbcb8d))

## [0.191.1](https://github.com/krisarmstrong/seed/compare/v0.191.0...v0.191.1) (2026-05-18)


### Bug Fixes

* **release:** Replace broken SLSA generator with attest-build-provenance ([#1023](https://github.com/krisarmstrong/seed/issues/1023)) ([1eec860](https://github.com/krisarmstrong/seed/commit/1eec860cc2866a8e576ce04c279005319e786645))

## [0.191.0](https://github.com/krisarmstrong/seed/compare/v0.190.0...v0.191.0) (2026-05-18)


### Features

* **ui:** storybook stories for phase B UI primitives ([#1021](https://github.com/krisarmstrong/seed/issues/1021)) ([56b04e7](https://github.com/krisarmstrong/seed/commit/56b04e7a8e6f9d8fd3d0b83246a0c4ea0ed56fbc))

## [0.190.0](https://github.com/krisarmstrong/seed/compare/v0.189.1...v0.190.0) (2026-05-18)


### Features

* **make:** add capability-aware dev-run target ([#1018](https://github.com/krisarmstrong/seed/issues/1018)) ([3d26875](https://github.com/krisarmstrong/seed/commit/3d26875f9afac36f6ca76ad6019e0bb2dfda1ddc))

## [0.189.1](https://github.com/krisarmstrong/seed/compare/v0.189.0...v0.189.1) (2026-05-18)


### Bug Fixes

* **ui:** unit tests handle Phase A sidebar buttons ([e9ff155](https://github.com/krisarmstrong/seed/commit/e9ff1558be5f7a5b6c76ef486d5d67d01130aeff))

## [0.189.0](https://github.com/krisarmstrong/seed/compare/v0.188.1...v0.189.0) (2026-05-18)


### Features

* **ui:** comprehensive tooltip parity — improve ~8 tooltips on key icon-only actions ([3b39977](https://github.com/krisarmstrong/seed/commit/3b3997753d33060e3a4a8aadb92deadf8d06faec))

## [0.188.1](https://github.com/krisarmstrong/seed/compare/v0.188.0...v0.188.1) (2026-05-17)


### Bug Fixes

* **ci:** bump Dockerfile go-build to golang:1.26-bookworm ([65c237e](https://github.com/krisarmstrong/seed/commit/65c237ef812d12bf44709b1007bfb611581fa737))
* **ci:** copy internal/i18n/locales into ui-build stage ([6737eed](https://github.com/krisarmstrong/seed/commit/6737eed6bc5f0a266baa6a676fbc456af61b45bc))
* **ci:** delete stale ui/vite.config.js (hand-maintained duplicate) ([d59f447](https://github.com/krisarmstrong/seed/commit/d59f4475a09ce0833a72793e3eb3610e246d5ad2))

## [0.188.0](https://github.com/krisarmstrong/seed/compare/v0.187.1...v0.188.0) (2026-05-17)


### Features

* **security:** guest network isolation audit ([#397](https://github.com/krisarmstrong/seed/issues/397)) ([#1003](https://github.com/krisarmstrong/seed/issues/1003)) ([81be6a8](https://github.com/krisarmstrong/seed/commit/81be6a8e9342994d4e8756b900b564d2f7102465))


### Bug Fixes

* **auth:** clear stale state, gate setup completion, match SSO contract ([#996](https://github.com/krisarmstrong/seed/issues/996)) ([e6280cf](https://github.com/krisarmstrong/seed/commit/e6280cf4edf9d4084fde127f5e0a76fd8ddc26a8))
* **setup:** enforce password complexity rules with live checklist ([#997](https://github.com/krisarmstrong/seed/issues/997)) ([073eb35](https://github.com/krisarmstrong/seed/commit/073eb35563e30e4f19f8fefc044e1d36347f9614))
* **survey:** client-side validation for ids, coords, floorplan size ([#999](https://github.com/krisarmstrong/seed/issues/999)) ([83ad1e9](https://github.com/krisarmstrong/seed/commit/83ad1e99a8d942c73d776f42f258a57fbf9d1ed7))
* **survey:** persist AirMapper-imported placements + criteria ([#727](https://github.com/krisarmstrong/seed/issues/727)) ([#1000](https://github.com/krisarmstrong/seed/issues/1000)) ([acffbd7](https://github.com/krisarmstrong/seed/commit/acffbd7a0e5adf3fa235a3d6a2b35ab72a8d5010))

## [0.187.1](https://github.com/krisarmstrong/seed/compare/v0.187.0...v0.187.1) (2026-05-16)


### Bug Fixes

* **ui:** gate Cable Test card on link absence ([#740](https://github.com/krisarmstrong/seed/issues/740)) ([fa5e028](https://github.com/krisarmstrong/seed/commit/fa5e0280b2643724bb7a5b1755137495ec517e54))

## [0.186.0](https://github.com/krisarmstrong/seed/compare/v0.185.13...v0.186.0) (2026-05-16)


### Features

* **ci:** restore Windows ARM64 in release matrix ([#944](https://github.com/krisarmstrong/seed/issues/944)) ([de8c595](https://github.com/krisarmstrong/seed/commit/de8c5957160214aba3d1ff2bf143e357ef49044a))
* implement Universal Build Contract for seed ([#946](https://github.com/krisarmstrong/seed/issues/946)) ([0c6870f](https://github.com/krisarmstrong/seed/commit/0c6870f7313e0981ce393194a0dd930c261c0653))


### Bug Fixes

* **ci:** pre-commit hook masks failing tests ([#947](https://github.com/krisarmstrong/seed/issues/947)) ([e8840f8](https://github.com/krisarmstrong/seed/commit/e8840f8db66f13bd07fb24feb4f6680b29689ebd))

## [0.185.13](https://github.com/krisarmstrong/seed/compare/v0.185.12...v0.185.13) (2026-05-15)


### Bug Fixes

* **ci:** stabilize seed release artifact matrix ([cd9b368](https://github.com/krisarmstrong/seed/commit/cd9b368df37ab223921748a435871fb97184a641))

## [0.185.12](https://github.com/krisarmstrong/seed/compare/v0.185.11...v0.185.12) (2026-05-14)


### Bug Fixes

* **ci:** skip seed docker publish without dockerfile ([8e1a075](https://github.com/krisarmstrong/seed/commit/8e1a075ba946c7fd1ea0b2618272e35a19194b56))

## [0.185.11](https://github.com/krisarmstrong/seed/compare/v0.185.10...v0.185.11) (2026-05-14)


### Bug Fixes

* **ci:** align seed setup e2e with current UI ([2505626](https://github.com/krisarmstrong/seed/commit/25056260412df6c420dba4ac4102d7ab3a31ff5b))
* **ci:** align seed validation steps ([34c03bb](https://github.com/krisarmstrong/seed/commit/34c03bb5fe5ccbc61989bcf1ee0e516d59e623a7))
* **ci:** allow MPL npm dependencies ([07f5e24](https://github.com/krisarmstrong/seed/commit/07f5e241da445e10400a30125621de2896e5deca))
* **ci:** build seed amd64 before arm64 deps ([774536b](https://github.com/krisarmstrong/seed/commit/774536b223205131a2b976b57f4623c6f15067ba))
* **ci:** exclude private npm packages from license scan ([ec78b14](https://github.com/krisarmstrong/seed/commit/ec78b14607daf21050ac8751962abcf147e8a46d))
* **ci:** fetch full history for security scans ([f2d00e4](https://github.com/krisarmstrong/seed/commit/f2d00e492814e6f2492e08aad6ca16e77e26fd21))
* **ci:** format tracked go sources only ([bbb36f0](https://github.com/krisarmstrong/seed/commit/bbb36f0ef63ba98539d6037a7c1470d89b64c8ba))
* **ci:** install arm64 kernel headers for seed builds ([e9a72a9](https://github.com/krisarmstrong/seed/commit/e9a72a9a43fefc1df71b08b0f8d22ebc705f9296))
* **ci:** keep seed lighthouse gate focused ([976b507](https://github.com/krisarmstrong/seed/commit/976b507ff1113c2573bed96a70bb423e6cda85ef))
* **ci:** keep seed setup e2e focused ([fdecc42](https://github.com/krisarmstrong/seed/commit/fdecc42b3ec5b48ba7e5f66c583d2371eacde3d6))
* **ci:** prepare assets before backend validation ([42fa3fd](https://github.com/krisarmstrong/seed/commit/42fa3fd57a6016473fe3747a24bfdcc18edc2454))
* **ci:** prepare seed data dir for browser jobs ([aab9b37](https://github.com/krisarmstrong/seed/commit/aab9b378c6586c86e0b0660ac7cd274473cbb777))
* **ci:** repair buildpacks project metadata ([863b7c7](https://github.com/krisarmstrong/seed/commit/863b7c7b4ee52411b49cf2eef79bad7c8a2116b6))
* **ci:** repair label sync workflow ([8711e8a](https://github.com/krisarmstrong/seed/commit/8711e8ab07960cdc6ada9951777c078973fcff61))
* **ci:** report seed gosec findings ([ce9b018](https://github.com/krisarmstrong/seed/commit/ce9b0186cb287e67236b1a42d71d0d1edf87f61a))
* **ci:** resolve seed validation blockers ([d34a4cf](https://github.com/krisarmstrong/seed/commit/d34a4cf96d76d584aefb432f68087e0fee2319f4))
* **ci:** scope seed browser smoke tests ([a7043f2](https://github.com/krisarmstrong/seed/commit/a7043f2207d104699813ac0a68ef90c949e8ab11))
* **ci:** scope seed license checks ([fbb9c7b](https://github.com/krisarmstrong/seed/commit/fbb9c7b34577882231f20ffadf41c667be4c5845))
* **ci:** skip seed docker publish without dockerfile ([fbd0962](https://github.com/krisarmstrong/seed/commit/fbd096287786d75e35c92d97e2da721d014e7989))
* **ci:** stabilize automated validation ([c822698](https://github.com/krisarmstrong/seed/commit/c8226987bce86539e8ffdc9647b0f418db860ece))
* **ci:** stabilize seed backend suite ([c92d728](https://github.com/krisarmstrong/seed/commit/c92d728558a652fea4c3f0294a1116b22b1fdf02))
* **ci:** stabilize seed backend tests ([d4cb236](https://github.com/krisarmstrong/seed/commit/d4cb236bd46eea39d0e2b0b8686101e1f9fa69e8))
* **ci:** stabilize seed reporting gates ([21edd25](https://github.com/krisarmstrong/seed/commit/21edd2572f4ed8548a0892325959c500d595f668))
* **ci:** use compatible labeler action ([92fed97](https://github.com/krisarmstrong/seed/commit/92fed972599e8cba169c5e1f284c2158488bbd04))
* **ci:** use labeler yaml format ([4629c5f](https://github.com/krisarmstrong/seed/commit/4629c5f7b2f36a799622fa4119ffbf59d776d6da))
* **ci:** use target dependencies for seed arm build ([1bf940f](https://github.com/krisarmstrong/seed/commit/1bf940f6cc63e8a758c9a38a03f462bd2693251b))
* **ci:** use writable seed config for browser jobs ([7a7a40b](https://github.com/krisarmstrong/seed/commit/7a7a40b8382d767e9b5fee3ba51f2229aed348be))
* **services:** reject dhcp tests for missing interfaces ([d205b88](https://github.com/krisarmstrong/seed/commit/d205b88f199ac8afb5848b7dfc095d8736d9b24f))

## [0.12.1](https://github.com/krisarmstrong/seed/compare/v0.12.0...v0.12.1) (2025-12-09)

### Bug Fixes

- **ci:** move libpcap-dev install to backend job for golangci-lint
  ([298d305](https://github.com/krisarmstrong/seed/commit/298d30511d4faaf900e0caf43fb3511eb75a20e6))
- **ci:** remove 'shadow' linter from .golangci.yml
  ([24ed597](https://github.com/krisarmstrong/seed/commit/24ed597ca9cb01d5d266f3408437243635eaa060))
- **ci:** remove accidental automerge.yml
  ([33a2b3f](https://github.com/krisarmstrong/seed/commit/33a2b3f76eb6c0e7d77b04da4c469bd5bc62b89b))
- **ci:** update golangci-lint version and format code
  ([5e58e96](https://github.com/krisarmstrong/seed/commit/5e58e964055fc884a1064cec71e051f060214d4c))
- **ci:** upgrade golangci-lint-action to v6
  ([2496c06](https://github.com/krisarmstrong/seed/commit/2496c060114d726daba19579034ec335159e6007))
- **ci:** use goinstall for golangci-lint to resolve go version incompatibility
  ([1ecd63f](https://github.com/krisarmstrong/seed/commit/1ecd63f988c010966931598c6f7ac55c6e82da70))
- **frontend:** debug eslint tsconfig path
  ([c86ab94](https://github.com/krisarmstrong/seed/commit/c86ab9493bec0d525affb96a05340147d6327a65))
- **frontend:** remove parserOptions.project from eslint config
  ([5a4d710](https://github.com/krisarmstrong/seed/commit/5a4d710f6c34fcc8343ff9838b52345e3d19bfd6))
- make DNS tester thread-safe for race tests
  ([31d74bf](https://github.com/krisarmstrong/seed/commit/31d74bfec7793b26d74d9bc02af616a9afa7980d))
- **release:** remove deprecated inputs from release-please config
  ([a602821](https://github.com/krisarmstrong/seed/commit/a6028217a9036068516b4f34ca468665a66957e8))

## [0.12.0](https://github.com/krisarmstrong/seed/compare/v0.11.9...v0.12.0) (2025-12-08)

### Features

- **release:** add debian packaging and systemd service
  ([bd1ed1a](https://github.com/krisarmstrong/seed/commit/bd1ed1a38430ee3344bc390850c90468791d7ba3))
- **release:** add docker containerization
  ([0b865ce](https://github.com/krisarmstrong/seed/commit/0b865cee356d3cc491247517567dcf918d0f9e5e))
- **release:** add fedora rpm packaging
  ([9353217](https://github.com/krisarmstrong/seed/commit/93532173f2fbf112cfbb0e5cf1dbacdc48d7383f))
- **web:** upgrade react to v19
  ([dec0cb9](https://github.com/krisarmstrong/seed/commit/dec0cb9deaa215cbc8b332b5760e9f5bf9198951))

### Bug Fixes

- **ci:** explicitly pass GITHUB_TOKEN to release-please
  ([f1f183e](https://github.com/krisarmstrong/seed/commit/f1f183e15108495e1cf15f93817ba1c5ae2075ef))
- **ci:** update golangci-lint to a compatible version
  ([8f97797](https://github.com/krisarmstrong/seed/commit/8f977974f47c6dc084177dd17c6b5e3c52c03c5c))
- **ci:** use PAT for release-please
  ([c0da65e](https://github.com/krisarmstrong/seed/commit/c0da65eeba1543de7a6bb58e0e2c8bf8a8943856))
- **frontend:** correct eslint tsconfig path
  ([31fe551](https://github.com/krisarmstrong/seed/commit/31fe55141d4932ae0284ace5cb169eebe60e547f))

## [Unreleased]

## [0.1.0] - 2025-12-03

### Added

#### Backend (Go)

- HTTP/HTTPS server with auto-generated self-signed TLS certificates
- WebSocket server for real-time card updates with heartbeat/ping-pong
- JWT authentication with bcrypt password hashing
- Network interface detection and management
- Configuration loading from YAML with sensible defaults
- Graceful shutdown handling

#### Frontend (React + TypeScript)

- WebSocket hook with auto-reconnect and connection status
- Authentication hook with login/logout flow
- Card component system with status indicators (green/yellow/red)
- 8 diagnostic cards: Link, Cable, VLAN, Switch, Wi-Fi, DHCP, DNS, Gateway
- Login form with default credentials hint
- Connection status indicator in header
- Responsive grid layout (mobile-friendly)
- WiFi Vigilante color scheme (dark mode default)

#### Infrastructure

- CI/CD pipeline with GitHub Actions
- Security scanning with CodeQL
- Dependabot for automated dependency updates
- Conventional commits enforcement
- BSL 1.1 license (converts to Apache 2.0 on 2029-12-01)

---

## [0.0.0] - 2025-12-02

### Added

- Initial project structure
- Project plan and architecture documentation

---

For detailed commit history, see: https://github.com/krisarmstrong/seed/commits/main
