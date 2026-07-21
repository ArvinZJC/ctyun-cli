# Changelog

## 0.1.0-beta.2 - 2026-07-21

### Changed

- Raised the required core range to `>=0.4.0 <1.0.0` because typed request-body options rely on non-string JSON serialization introduced in core 0.4.0.

## 0.1.0-beta.1 - 2026-07-17

- Added the initial generated Image Management Service plugin with all 27 first-class-node APIs from the selected public portal revision.
- Added image lifecycle, import and export, copy, sharing, labels, task, and destination-region commands with profile-region overrides and confirmations for state-changing operations.
- Added localized help, typed request inputs, official response fixtures, table metadata, recommendation evidence, and the documented `errorFree` deprecation.
- Added normalized source and promoted baseline evidence for reproducible API inventory and provenance checks.
- Added a DPS image-list recommendation to IMS list and show help, qualified to physical-machine image retrieval.
