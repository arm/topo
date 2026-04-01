# Change Log

## [2.0.0](https://github.com/arm/topo/compare/v1.4.1...v2.0.0) (2026-04-01)


### ⚠ BREAKING CHANGES

* lack of remoteproc endpoints is an info, not a warning ([#159](https://github.com/arm/topo/issues/159))
* remove `--dry-run` support ([#142](https://github.com/arm/topo/issues/142))

### Features

* only allow one user per Host/alias when setting up keys ([#156](https://github.com/arm/topo/issues/156)) ([7bd3a2b](https://github.com/arm/topo/commit/7bd3a2b23e0e9a92923d6df2347a9f6381623ac7))
* remove `--dry-run` support ([#142](https://github.com/arm/topo/issues/142)) ([0021b68](https://github.com/arm/topo/commit/0021b689d45a1e0c1af4a9b553128f0a5e3829ca))
* standardised structured logging ([#161](https://github.com/arm/topo/issues/161)) ([cf3cfdf](https://github.com/arm/topo/commit/cf3cfdf3cc0a59f6fe53a363a4f3c87274e50040))
* suggest running `topo health` when deployment fails ([#148](https://github.com/arm/topo/issues/148)) ([8467798](https://github.com/arm/topo/commit/846779837a160bd2e7c0d0595ffa54c532e9d184))


### Bug Fixes

* lack of remoteproc endpoints is an info, not a warning ([#159](https://github.com/arm/topo/issues/159)) ([887f1a1](https://github.com/arm/topo/commit/887f1a1cf14ba1628b0070e0189fa046b741e960))
* scope template build args per service ([#140](https://github.com/arm/topo/issues/140)) ([f595e48](https://github.com/arm/topo/commit/f595e4876de59897cd117725eacf34accb3ed1bb))

## [Unreleased]

### Added

- Initial release
- Operations supported: `help`, `clone`, `completion`, `deploy`, `describe`, `extend`, `health`, `init`, `install`, `service`, `setup-keys`, `stop`, and `templates`.
