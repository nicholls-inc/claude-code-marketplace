# Changelog

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
