# Changelog

## Unreleased

### Added

- Added image active, deactivated, accepted, and integrity-checked waits selected by image ID; HMAC calculation alone remains pending.

### Changed

- Corrected private-image deactivation, reactivation, and import-task deletion to send their documented inputs in JSON request bodies.
- Aligned remaining image-status descriptions with the official private-image lifecycle terminology.
- Aligned the private-image disable and re-enable command titles with the official `弃用私有镜像` and `取消弃用私有镜像` terminology without treating those lifecycle actions as deprecated CLI commands.
- Required core range `>=0.4.0 <1.0.0` → `>=0.5.0 <1.0.0` for command-bound waiter metadata.

## 0.1.0-beta.2 - 2026-07-22

### Changed

- Aligned the Simplified Chinese product name with the official title `镜像服务 IMS`.
- Required core range `>=0.3.1 <1.0.0` → `>=0.4.0 <1.0.0` because typed request-body options rely on non-string JSON serialization introduced in core 0.4.0.

## 0.1.0-beta.1 - 2026-07-17

- Added the initial generated Image Management Service plugin with all 27 first-class-node APIs from the selected public portal revision.
- Added image lifecycle, import and export, copy, sharing, labels, task, and destination-region commands with profile-region overrides and confirmations for state-changing operations.
- Added localized help, typed request inputs, official response fixtures, table metadata, recommendation evidence, and the documented `errorFree` deprecation.
- Added normalized source and promoted baseline evidence for reproducible API inventory and provenance checks.
- Added a DPS image-list recommendation to IMS list and show help, qualified to physical-machine image retrieval.
