# Changelog

## [1.2.0](https://github.com/nicholls-inc/claude-code-marketplace/compare/xylem-v1.1.0...xylem-v1.2.0) (2026-03-29)


### Features

* **bootstrap:** add multi-level instruction hierarchy with convention detection ([#110](https://github.com/nicholls-inc/claude-code-marketplace/issues/110)) ([d2e13f6](https://github.com/nicholls-inc/claude-code-marketplace/commit/d2e13f6f818082a1125a7f35007e3d8134ca33d1))
* **xylem:** add behavioral signals package ([#78](https://github.com/nicholls-inc/claude-code-marketplace/issues/78)) ([18a365d](https://github.com/nicholls-inc/claude-code-marketplace/commit/18a365dc0a69350c57af685cafc1fa22dc6bf23c))
* **xylem:** add context management package ([#77](https://github.com/nicholls-inc/claude-code-marketplace/issues/77)) ([79c36ca](https://github.com/nicholls-inc/claude-code-marketplace/commit/79c36ca85d278e96ab420161e0218b49f55640ee))
* **xylem:** add ContractPoster interface, FormatContractMarkdown, and SaveAndPost ([06516d4](https://github.com/nicholls-inc/claude-code-marketplace/commit/06516d442d740d64dfd295746ef623173fb73e6e))
* **xylem:** add cost tracking package ([#85](https://github.com/nicholls-inc/claude-code-marketplace/issues/85)) ([57edace](https://github.com/nicholls-inc/claude-code-marketplace/commit/57edaceb45d8a1246b0a3a537ecabc1b9494dbb6))
* **xylem:** add deterministic intermediary package ([#79](https://github.com/nicholls-inc/claude-code-marketplace/issues/79)) ([561945c](https://github.com/nicholls-inc/claude-code-marketplace/commit/561945c05f7c0a5b37cd19d0d82f4157769bfcf1))
* **xylem:** add enhanced bootstrap package ([#86](https://github.com/nicholls-inc/claude-code-marketplace/issues/86)) ([b9384fd](https://github.com/nicholls-inc/claude-code-marketplace/commit/b9384fddd70006d333207c6fdb0f6aee1161d4f1))
* **xylem:** add evaluation system package ([#81](https://github.com/nicholls-inc/claude-code-marketplace/issues/81)) ([919e66d](https://github.com/nicholls-inc/claude-code-marketplace/commit/919e66d9de0c09a660860d780f18d39067b13191))
* **xylem:** add failure handling to orchestrator ([#104](https://github.com/nicholls-inc/claude-code-marketplace/issues/104)) ([0d896b2](https://github.com/nicholls-inc/claude-code-marketplace/commit/0d896b27ea4b62520ee0caeab745e5ce2aeb136b))
* **xylem:** add generation pipeline functions to bootstrap package ([#106](https://github.com/nicholls-inc/claude-code-marketplace/issues/106)) ([aff2bb5](https://github.com/nicholls-inc/claude-code-marketplace/commit/aff2bb5b1668017a46eec13ba50735f90012d687))
* **xylem:** add memory system package ([#82](https://github.com/nicholls-inc/claude-code-marketplace/issues/82)) ([5c33e8d](https://github.com/nicholls-inc/claude-code-marketplace/commit/5c33e8d5e9beb07e59e3431bb6d6f223a7a07c6f))
* **xylem:** add mission and sprint contract package ([#80](https://github.com/nicholls-inc/claude-code-marketplace/issues/80)) ([d2c7f75](https://github.com/nicholls-inc/claude-code-marketplace/commit/d2c7f75d23e5cdb9f6c09ffc988ac06345be3c21))
* **xylem:** add multi-agent orchestration package ([#84](https://github.com/nicholls-inc/claude-code-marketplace/issues/84)) ([0c49cf6](https://github.com/nicholls-inc/claude-code-marketplace/commit/0c49cf65d6a4ad698337b3cab78198a3d53bbedd))
* **xylem:** add observability foundation package ([#102](https://github.com/nicholls-inc/claude-code-marketplace/issues/102)) ([aff1313](https://github.com/nicholls-inc/claude-code-marketplace/commit/aff13136d111cd0ba862c10d3ad1152f8c875173))
* **xylem:** add OTel SDK wiring to observability package ([#109](https://github.com/nicholls-inc/claude-code-marketplace/issues/109)) ([c2b2284](https://github.com/nicholls-inc/claude-code-marketplace/commit/c2b22848a2a08952b596b28efacb9ab227374b56))
* **xylem:** add progress file tracking and session startup ritual ([#103](https://github.com/nicholls-inc/claude-code-marketplace/issues/103)) ([31a7f85](https://github.com/nicholls-inc/claude-code-marketplace/commit/31a7f85a0f769eef19516a741be51402557b4db3))
* **xylem:** add semantic validation for memory entries ([#108](https://github.com/nicholls-inc/claude-code-marketplace/issues/108)) ([8d25086](https://github.com/nicholls-inc/claude-code-marketplace/commit/8d250867e082a4139c75102bcac242dddcddcb16))
* **xylem:** add ShouldEvaluate and HealthString to SignalSet ([#100](https://github.com/nicholls-inc/claude-code-marketplace/issues/100)) ([b5f5279](https://github.com/nicholls-inc/claude-code-marketplace/commit/b5f52798f825eacf6968f2ddcd4b82c3d3fcdc88))
* **xylem:** add tool catalog package ([#83](https://github.com/nicholls-inc/claude-code-marketplace/issues/83)) ([eb9b658](https://github.com/nicholls-inc/claude-code-marketplace/commit/eb9b65816df4a75a62df2d06b066e20715660bd1))
* **xylem:** bridge signal package to observability span attributes ([#107](https://github.com/nicholls-inc/claude-code-marketplace/issues/107)) ([8ef95a9](https://github.com/nicholls-inc/claude-code-marketplace/commit/8ef95a9e7700ee39de6573cb02297aabc90dbf85))
* **xylem:** implement windowed budget tracking in cost package ([#105](https://github.com/nicholls-inc/claude-code-marketplace/issues/105)) ([264c53e](https://github.com/nicholls-inc/claude-code-marketplace/commit/264c53e2484719b73444699dc4dcaee33456e913))
* **xylem:** wire cost tracking into orchestrator as sub-agent token firewall ([#111](https://github.com/nicholls-inc/claude-code-marketplace/issues/111)) ([02effe9](https://github.com/nicholls-inc/claude-code-marketplace/commit/02effe96a9f73a74184f8c9fabac93879706f731))
* **xylem:** wire SelectIntensity into eval loop ([#101](https://github.com/nicholls-inc/claude-code-marketplace/issues/101)) ([991a291](https://github.com/nicholls-inc/claude-code-marketplace/commit/991a29186371eec5604de85c96722f309dbae6c9))


### Bug Fixes

* **observability:** add OTLP gRPC exporter path to NewTracer ([#113](https://github.com/nicholls-inc/claude-code-marketplace/issues/113)) ([2140fd3](https://github.com/nicholls-inc/claude-code-marketplace/commit/2140fd34e43dd1c78ca1b0b0e7297a0bf8141cac))
* **orchestrator:** record delta tokens in UpdateAgent instead of full amount ([#112](https://github.com/nicholls-inc/claude-code-marketplace/issues/112)) ([22db479](https://github.com/nicholls-inc/claude-code-marketplace/commit/22db479228ee29efa1f0123a19f17f5ce895cdba))
* **xylem:** remove space from genTool description regex to prevent empty strings ([#98](https://github.com/nicholls-inc/claude-code-marketplace/issues/98)) ([109a964](https://github.com/nicholls-inc/claude-code-marketplace/commit/109a96460bc7b9f502ffee79197df1e5448e9725))
* **xylem:** rename SummaryMaxTokens to SummaryMaxChars in orchestrator ([#99](https://github.com/nicholls-inc/claude-code-marketplace/issues/99)) ([7fc2f96](https://github.com/nicholls-inc/claude-code-marketplace/commit/7fc2f96cc06e26ac83f5780a69df0c50689d362d))

## [1.1.0](https://github.com/nicholls-inc/claude-code-marketplace/compare/xylem-v1.0.1...xylem-v1.1.0) (2026-03-29)


### Features

* **xylem:** add daemon and retry commands ([#74](https://github.com/nicholls-inc/claude-code-marketplace/issues/74)) ([5098e25](https://github.com/nicholls-inc/claude-code-marketplace/commit/5098e25c9de31d6c767f2096e436d66a00a40c76))
* **xylem:** add gate package for command and label gate execution ([#67](https://github.com/nicholls-inc/claude-code-marketplace/issues/67)) ([4d6a85f](https://github.com/nicholls-inc/claude-code-marketplace/commit/4d6a85feb032d9589440e290ed7b97722f983526))
* **xylem:** add phase package for template data types and prompt rendering ([#66](https://github.com/nicholls-inc/claude-code-marketplace/issues/66)) ([4266bc9](https://github.com/nicholls-inc/claude-code-marketplace/commit/4266bc97d4149a341182d24950d4047097207283))
* **xylem:** add reporter package for posting phase progress to GitHub issues ([#68](https://github.com/nicholls-inc/claude-code-marketplace/issues/68)) ([23cf8ea](https://github.com/nicholls-inc/claude-code-marketplace/commit/23cf8eabac3febdb0339897b445d3ce46175484a))
* **xylem:** add skill package for v2 phase-based execution ([#73](https://github.com/nicholls-inc/claude-code-marketplace/issues/73)) ([28bf58b](https://github.com/nicholls-inc/claude-code-marketplace/commit/28bf58b212cf22b5ebeae4fe95ff323c7534d10c))
* **xylem:** add v2 queue states, vessel fields, and methods for phase-based execution ([#69](https://github.com/nicholls-inc/claude-code-marketplace/issues/69)) ([2c39ec4](https://github.com/nicholls-inc/claude-code-marketplace/commit/2c39ec44e6101778824bb63a928efd5c0ccde9e5))
* **xylem:** add waiting/timed_out states to status and phase cleanup ([#70](https://github.com/nicholls-inc/claude-code-marketplace/issues/70)) ([d92f2ee](https://github.com/nicholls-inc/claude-code-marketplace/commit/d92f2eedb3d800ec9e22723c789d9772e6a6c6a6))
* **xylem:** expand init to scaffold v2 skills, prompts, and harness ([#72](https://github.com/nicholls-inc/claude-code-marketplace/issues/72)) ([dde1968](https://github.com/nicholls-inc/claude-code-marketplace/commit/dde1968331494bb95ae5cd66e4fccb38e6a94194))
* **xylem:** replace template config with flags/env for v2 ([#71](https://github.com/nicholls-inc/claude-code-marketplace/issues/71)) ([48a0ac6](https://github.com/nicholls-inc/claude-code-marketplace/commit/48a0ac65582a3a03ed3889d8d8a9a0a4cffd13c1))
* **xylem:** rewrite runner for v2 phase-based execution ([#75](https://github.com/nicholls-inc/claude-code-marketplace/issues/75)) ([63e9dbe](https://github.com/nicholls-inc/claude-code-marketplace/commit/63e9dbe8557c86b455500bf31cef794586349eaf))


### Bug Fixes

* **xylem:** forward --ref value to Claude in direct-prompt mode ([#57](https://github.com/nicholls-inc/claude-code-marketplace/issues/57)) ([#63](https://github.com/nicholls-inc/claude-code-marketplace/issues/63)) ([df5a590](https://github.com/nicholls-inc/claude-code-marketplace/commit/df5a5909e19c5e3e4943288a3818360b829adab4))
* **xylem:** pass allowed_tools as --allowedTools flags to Claude sessions ([#58](https://github.com/nicholls-inc/claude-code-marketplace/issues/58)) ([#65](https://github.com/nicholls-inc/claude-code-marketplace/issues/65)) ([8158810](https://github.com/nicholls-inc/claude-code-marketplace/commit/8158810d41f168f370865e9c26aeb486418a2839))

## [1.0.1](https://github.com/nicholls-inc/claude-code-marketplace/compare/xylem-v1.0.0...xylem-v1.0.1) (2026-03-16)


### Bug Fixes

* **xylem:** respect --config flag in init command ([#56](https://github.com/nicholls-inc/claude-code-marketplace/issues/56)) ([4b2cb92](https://github.com/nicholls-inc/claude-code-marketplace/commit/4b2cb923ca8dac27018fb102ccb5bb78ee1511c8))

## 1.0.0 (2026-03-14)


### Features

* **xylem:** add xylem plugin — autonomous agent scheduling for GitHub issues ([#48](https://github.com/nicholls-inc/claude-code-marketplace/issues/48)) ([a384e29](https://github.com/nicholls-inc/claude-code-marketplace/commit/a384e2972f3aaede9f992490afbf93cc2754fe5c))
