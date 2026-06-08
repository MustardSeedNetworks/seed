# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
