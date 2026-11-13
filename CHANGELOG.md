# Changelog

## [0.6.1](https://github.com/alexpatil/sukami/compare/v0.6.0...v0.6.1) (2026-11-13)


### Bug Fixes

* **auth:** remove the seeded default key from the in-memory key store ([#48](https://github.com/alexpatil/sukami/issues/48)) ([81068cd](https://github.com/alexpatil/sukami/commit/81068cdc851e1c33e8702cfb9516c32d2ca2665f)), closes [#44](https://github.com/alexpatil/sukami/issues/44)
* **bench:** measure through the proxy and report only measured numbers ([#58](https://github.com/alexpatil/sukami/issues/58)) ([34d3008](https://github.com/alexpatil/sukami/commit/34d3008b3cfc3585a76c9d69df6e7a24283a1e78)), closes [#57](https://github.com/alexpatil/sukami/issues/57)
* **cache:** include request parameters in the cache key ([#60](https://github.com/alexpatil/sukami/issues/60)) ([d207bb2](https://github.com/alexpatil/sukami/commit/d207bb23708283a54ed664588361740769f9913e)), closes [#51](https://github.com/alexpatil/sukami/issues/51)
* **cache:** isolate cache per caller and per conversation context ([#47](https://github.com/alexpatil/sukami/issues/47)) ([b34a903](https://github.com/alexpatil/sukami/commit/b34a903eff2f11923684bf0db3101c4e36849e8f)), closes [#43](https://github.com/alexpatil/sukami/issues/43)
* **proxy:** forward request fields the gateway does not model ([#55](https://github.com/alexpatil/sukami/issues/55)) ([84b20bb](https://github.com/alexpatil/sukami/commit/84b20bb6923f59728f4b91f4daaac77640f05431)), closes [#52](https://github.com/alexpatil/sukami/issues/52)

## [0.6.0](https://github.com/alexpatil/sukami/compare/v0.5.0...v0.6.0) (2026-11-02)


### Features

* **cli:** add -v and --version flags to print binary version ([#41](https://github.com/alexpatil/sukami/issues/41)) ([793cd71](https://github.com/alexpatil/sukami/commit/793cd7186c79d573a317af9c11e7163acbf052d5))


### Bug Fixes

* **ci:** configure standard v-tags and automated binary uploads in release-please ([#39](https://github.com/alexpatil/sukami/issues/39)) ([cc669b3](https://github.com/alexpatil/sukami/commit/cc669b38f01f47cad80015acd084eb3ad2c792cd))
* **ci:** evaluate release-created condition strictly and enable pre-major patch bumps ([#42](https://github.com/alexpatil/sukami/issues/42)) ([4e049dd](https://github.com/alexpatil/sukami/commit/4e049ddbd962016347610d717d6a478081e2ea42))

## [0.5.0](https://github.com/alexpatil/sukami/compare/sukami-v0.4.0...sukami-v0.5.0) (2026-11-01)


### Features

* **auth,proxy:** implement flexible dual-key resolution with header passthrough and UI key support ([2b11e7a](https://github.com/alexpatil/sukami/commit/2b11e7a24bd5d27bbc8cb2165e3cb6cc7d9f433e))
* **ci:** add GitHub Actions multi-OS release workflow ([#16](https://github.com/alexpatil/sukami/issues/16)) ([3739c03](https://github.com/alexpatil/sukami/commit/3739c03cc7b669e120896b9b17e2a50be5a11c1d))
* **cli:** add CLI flags for port and upstream with priority hierarchy (flag &gt; env &gt; default) ([d27f4c7](https://github.com/alexpatil/sukami/commit/d27f4c766cec91ad1b2d2fd61edae84fdf32f7c6))
* **docker:** add lightweight multi-stage Dockerfile and docker-compo… ([515cb5d](https://github.com/alexpatil/sukami/commit/515cb5d5fb56b3f589d76c7546814904d9be65a6))
* **docker:** add lightweight multi-stage Dockerfile and docker-compose setup ([6d8aea1](https://github.com/alexpatil/sukami/commit/6d8aea1653e2ef96a0de7396b2b39332ecca2532))
* initial commit for Sukami AI token optimization gateway ([eb525a4](https://github.com/alexpatil/sukami/commit/eb525a417b4b222f110303aa8ca4f690c3fd6059))
* **metrics:** add standard Prometheus exposition endpoint /metrics in pure Go ([7934810](https://github.com/alexpatil/sukami/commit/7934810803e3fe6a356d2bb64a1d31cc5380442a))
* **metrics:** add standard Prometheus exposition endpoint /metrics in pure Go ([189c61b](https://github.com/alexpatil/sukami/commit/189c61befb37d47904c91ce14e9854a63c3f7866))
* **optimizer:** eliminate tiktoken in favor of model-aware pure Go tokenizer ([204c959](https://github.com/alexpatil/sukami/commit/204c9593acfcadc11b7741f3d1367fa567cbd8a0))
* **optimizer:** eliminate tiktoken in favor of model-aware pure Go tokenizer ([ba943cf](https://github.com/alexpatil/sukami/commit/ba943cf06dc406013e7692860fdb30d03f34ca80))
* **optimizer:** implement upstream-aware dynamic smart model routing ([0936514](https://github.com/alexpatil/sukami/commit/0936514859ead1cce4298b9d3c4c8b23193e24d0))
* **pricing:** implement 100% dynamic API-driven model pricing engine ([f6b3d2f](https://github.com/alexpatil/sukami/commit/f6b3d2f9730bf8b0454227f1ae15e48f6e2c1688))
* **pricing:** implement 100% dynamic API-driven model pricing engine ([0184aea](https://github.com/alexpatil/sukami/commit/0184aeafda671a2637861b272935ffec3845eb14))
* prompt-complexity smart model router (80-95% cost savings) ([b81c874](https://github.com/alexpatil/sukami/commit/b81c8747a439b3e2aa594ae964883c91da3580f3))
* **proxy,ui:** add dynamic multi-provider routing and live model catalog fetching ([d0e4714](https://github.com/alexpatil/sukami/commit/d0e47145d78525b6c9dc5094c1ad3200cd76a80c))
* **proxy:** harden connection pooling, smart model routing, and 401 auth ([085e5d6](https://github.com/alexpatil/sukami/commit/085e5d691d1010db31cd93ed7fac72f84e7778cd))
* **proxy:** harden connection pooling, smart model routing, and 401 auth ([9a8e6f4](https://github.com/alexpatil/sukami/commit/9a8e6f4cb31ea658a678a5d019f71ce38bf38688))
* **proxy:** stream-through cache interception and synthesis for SSE streaming completions ([#30](https://github.com/alexpatil/sukami/issues/30)) ([b61ae93](https://github.com/alexpatil/sukami/commit/b61ae93b3e11b91877196e0100458a84796aaf1b)), closes [#29](https://github.com/alexpatil/sukami/issues/29)
* **proxy:** upstream automatic retries with exponential backoff for transient errors ([#36](https://github.com/alexpatil/sukami/issues/36)) ([8182dad](https://github.com/alexpatil/sukami/commit/8182dad8ab51cb19d3dbd0f6b88c2955fe46be79))
* **router:** add sub-millisecond prompt complexity classifier and smart model router ([d3c09bf](https://github.com/alexpatil/sukami/commit/d3c09bf4d94748a8f6a5c14cf88f5b16264f9c3e))
* **test:** add end-to-end integration test suite and modularize gateway router ([15677c7](https://github.com/alexpatil/sukami/commit/15677c797c7148b6e03b8f5f1e0657d1d5bec879))
* **ui:** redesign developer dashboard with polished typography, crystal-clear professional copy, and interactive sandbox ([b292f7f](https://github.com/alexpatil/sukami/commit/b292f7f966006bc0c3acd900ecd0f8288505aad4))
* **web:** add smart model routing telemetry and visual badges to playground ([#17](https://github.com/alexpatil/sukami/issues/17)) ([8d030eb](https://github.com/alexpatil/sukami/commit/8d030eb5b4f5146493721aedb1e9ee93d2b6da76))


### Bug Fixes

* **core:** harden edge cases across optimizer, cache, auth, and proxy ([5c42408](https://github.com/alexpatil/sukami/commit/5c42408b459842e488080bd2fd397a5387eaa3ec))
* **core:** harden edge cases across optimizer, cache, auth, and proxy ([49b1f78](https://github.com/alexpatil/sukami/commit/49b1f78fed1f819b40b396f2cfbd2327c19e8e39))
* **docker:** update Dockerfile for headless architecture and add GHCR multi-arch workflow (closes [#27](https://github.com/alexpatil/sukami/issues/27)) ([6b70862](https://github.com/alexpatil/sukami/commit/6b7086258f3ce4693579fc4cbc185dc7453b1017))
* **docker:** update Dockerfile for headless architecture and add GHCR workflow ([eeaef59](https://github.com/alexpatil/sukami/commit/eeaef594408b384eb20e61d4cd25a712e25e989b))
* **optimizer,cache:** implement real tiktoken BPE and harden semantic cache against entity collisions ([635c91b](https://github.com/alexpatil/sukami/commit/635c91b9eb64f6db694bc9a4716536d3032c1f67))
* **proxy:** remove DefaultUpstreamURL and fake sukami.ai domain headers ([a17cb8c](https://github.com/alexpatil/sukami/commit/a17cb8c29704c00ea5a374e4b23d43a49ddf0d2d))
* **proxy:** transparently forward /v1/models and eliminate silent model hijacking ([b9fdc06](https://github.com/alexpatil/sukami/commit/b9fdc06919356a0b983aedb0ba4c6e8c7cbe7571))
* remove unused file and clean up repository structure ([c5cde08](https://github.com/alexpatil/sukami/commit/c5cde084a15fc2ba0257cf8b7e7486ade729005e))
* set minimum supported Go version to 1.22+ in go.mod and README ([f8e322e](https://github.com/alexpatil/sukami/commit/f8e322e689336e49d29fb75f229c4c648f087309))
* set minimum supported Go version to 1.22+ in go.mod and README ([199ec65](https://github.com/alexpatil/sukami/commit/199ec65a28ee24153e42603384532930ddad425c))


### Reverts

* **docs:** restore README.md to PR [#25](https://github.com/alexpatil/sukami/issues/25) specification ([178344b](https://github.com/alexpatil/sukami/commit/178344b4759ea0bad8b475ab166cd6c480112631))

## [0.4.0](https://github.com/alexpatil/sukami/compare/sukami-v0.3.0...sukami-v0.4.0) (2026-11-01)


### Features

* **auth,proxy:** implement flexible dual-key resolution with header passthrough and UI key support ([2b11e7a](https://github.com/alexpatil/sukami/commit/2b11e7a24bd5d27bbc8cb2165e3cb6cc7d9f433e))
* **ci:** add GitHub Actions multi-OS release workflow ([#16](https://github.com/alexpatil/sukami/issues/16)) ([3739c03](https://github.com/alexpatil/sukami/commit/3739c03cc7b669e120896b9b17e2a50be5a11c1d))
* **cli:** add CLI flags for port and upstream with priority hierarchy (flag &gt; env &gt; default) ([d27f4c7](https://github.com/alexpatil/sukami/commit/d27f4c766cec91ad1b2d2fd61edae84fdf32f7c6))
* **docker:** add lightweight multi-stage Dockerfile and docker-compo… ([515cb5d](https://github.com/alexpatil/sukami/commit/515cb5d5fb56b3f589d76c7546814904d9be65a6))
* **docker:** add lightweight multi-stage Dockerfile and docker-compose setup ([6d8aea1](https://github.com/alexpatil/sukami/commit/6d8aea1653e2ef96a0de7396b2b39332ecca2532))
* initial commit for Sukami AI token optimization gateway ([eb525a4](https://github.com/alexpatil/sukami/commit/eb525a417b4b222f110303aa8ca4f690c3fd6059))
* **metrics:** add standard Prometheus exposition endpoint /metrics in pure Go ([7934810](https://github.com/alexpatil/sukami/commit/7934810803e3fe6a356d2bb64a1d31cc5380442a))
* **metrics:** add standard Prometheus exposition endpoint /metrics in pure Go ([189c61b](https://github.com/alexpatil/sukami/commit/189c61befb37d47904c91ce14e9854a63c3f7866))
* **optimizer:** eliminate tiktoken in favor of model-aware pure Go tokenizer ([204c959](https://github.com/alexpatil/sukami/commit/204c9593acfcadc11b7741f3d1367fa567cbd8a0))
* **optimizer:** eliminate tiktoken in favor of model-aware pure Go tokenizer ([ba943cf](https://github.com/alexpatil/sukami/commit/ba943cf06dc406013e7692860fdb30d03f34ca80))
* **optimizer:** implement upstream-aware dynamic smart model routing ([0936514](https://github.com/alexpatil/sukami/commit/0936514859ead1cce4298b9d3c4c8b23193e24d0))
* **pricing:** implement 100% dynamic API-driven model pricing engine ([f6b3d2f](https://github.com/alexpatil/sukami/commit/f6b3d2f9730bf8b0454227f1ae15e48f6e2c1688))
* **pricing:** implement 100% dynamic API-driven model pricing engine ([0184aea](https://github.com/alexpatil/sukami/commit/0184aeafda671a2637861b272935ffec3845eb14))
* prompt-complexity smart model router (80-95% cost savings) ([b81c874](https://github.com/alexpatil/sukami/commit/b81c8747a439b3e2aa594ae964883c91da3580f3))
* **proxy,ui:** add dynamic multi-provider routing and live model catalog fetching ([d0e4714](https://github.com/alexpatil/sukami/commit/d0e47145d78525b6c9dc5094c1ad3200cd76a80c))
* **proxy:** harden connection pooling, smart model routing, and 401 auth ([085e5d6](https://github.com/alexpatil/sukami/commit/085e5d691d1010db31cd93ed7fac72f84e7778cd))
* **proxy:** harden connection pooling, smart model routing, and 401 auth ([9a8e6f4](https://github.com/alexpatil/sukami/commit/9a8e6f4cb31ea658a678a5d019f71ce38bf38688))
* **proxy:** stream-through cache interception and synthesis for SSE streaming completions ([#30](https://github.com/alexpatil/sukami/issues/30)) ([b61ae93](https://github.com/alexpatil/sukami/commit/b61ae93b3e11b91877196e0100458a84796aaf1b)), closes [#29](https://github.com/alexpatil/sukami/issues/29)
* **proxy:** upstream automatic retries with exponential backoff for transient errors ([#36](https://github.com/alexpatil/sukami/issues/36)) ([8182dad](https://github.com/alexpatil/sukami/commit/8182dad8ab51cb19d3dbd0f6b88c2955fe46be79))
* **router:** add sub-millisecond prompt complexity classifier and smart model router ([d3c09bf](https://github.com/alexpatil/sukami/commit/d3c09bf4d94748a8f6a5c14cf88f5b16264f9c3e))
* **test:** add end-to-end integration test suite and modularize gateway router ([15677c7](https://github.com/alexpatil/sukami/commit/15677c797c7148b6e03b8f5f1e0657d1d5bec879))
* **ui:** redesign developer dashboard with polished typography, crystal-clear professional copy, and interactive sandbox ([b292f7f](https://github.com/alexpatil/sukami/commit/b292f7f966006bc0c3acd900ecd0f8288505aad4))
* **web:** add smart model routing telemetry and visual badges to playground ([#17](https://github.com/alexpatil/sukami/issues/17)) ([8d030eb](https://github.com/alexpatil/sukami/commit/8d030eb5b4f5146493721aedb1e9ee93d2b6da76))


### Bug Fixes

* **core:** harden edge cases across optimizer, cache, auth, and proxy ([5c42408](https://github.com/alexpatil/sukami/commit/5c42408b459842e488080bd2fd397a5387eaa3ec))
* **core:** harden edge cases across optimizer, cache, auth, and proxy ([49b1f78](https://github.com/alexpatil/sukami/commit/49b1f78fed1f819b40b396f2cfbd2327c19e8e39))
* **docker:** update Dockerfile for headless architecture and add GHCR multi-arch workflow (closes [#27](https://github.com/alexpatil/sukami/issues/27)) ([6b70862](https://github.com/alexpatil/sukami/commit/6b7086258f3ce4693579fc4cbc185dc7453b1017))
* **docker:** update Dockerfile for headless architecture and add GHCR workflow ([eeaef59](https://github.com/alexpatil/sukami/commit/eeaef594408b384eb20e61d4cd25a712e25e989b))
* **optimizer,cache:** implement real tiktoken BPE and harden semantic cache against entity collisions ([635c91b](https://github.com/alexpatil/sukami/commit/635c91b9eb64f6db694bc9a4716536d3032c1f67))
* **proxy:** remove DefaultUpstreamURL and fake sukami.ai domain headers ([a17cb8c](https://github.com/alexpatil/sukami/commit/a17cb8c29704c00ea5a374e4b23d43a49ddf0d2d))
* **proxy:** transparently forward /v1/models and eliminate silent model hijacking ([b9fdc06](https://github.com/alexpatil/sukami/commit/b9fdc06919356a0b983aedb0ba4c6e8c7cbe7571))
* remove unused file and clean up repository structure ([c5cde08](https://github.com/alexpatil/sukami/commit/c5cde084a15fc2ba0257cf8b7e7486ade729005e))
* set minimum supported Go version to 1.22+ in go.mod and README ([f8e322e](https://github.com/alexpatil/sukami/commit/f8e322e689336e49d29fb75f229c4c648f087309))
* set minimum supported Go version to 1.22+ in go.mod and README ([199ec65](https://github.com/alexpatil/sukami/commit/199ec65a28ee24153e42603384532930ddad425c))


### Reverts

* **docs:** restore README.md to PR [#25](https://github.com/alexpatil/sukami/issues/25) specification ([178344b](https://github.com/alexpatil/sukami/commit/178344b4759ea0bad8b475ab166cd6c480112631))
