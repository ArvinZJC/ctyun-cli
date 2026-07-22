# Changelog

## 0.1.0-beta.2 - 2026-07-21

### Changed

- Normalized generated English casing and Simplified Chinese table labels against the tracked OpenAPI source.
- Raised the required core range to `>=0.4.0 <1.0.0` because typed request-body options rely on non-string JSON serialization introduced in core 0.4.0.

## 0.1.0-beta.1 - 2026-07-17

### Added

- Generated 22 first-class-node Cloud Backup and Recovery commands from the official public OpenAPI records, covering storage repositories, backup and restore tasks, policies, snapshots, clients, protected resources, and regional service activation.
- Tracked normalized `source.json` evidence and a semantically identical promoted `baseline.json` snapshot for the complete `/v4/backup/` API scope.
- Added natural Simplified Chinese, American English, and British English help, including reviewed JSON policy inputs that preserve documented ranges, units, enum tokens, defaults, and conditional requirements.
- Added official response fixtures and matching output tables for all 22 commands, including documented deprecated response fields and their replacements.
- Commands that use `regionID` read the selected profile region by default and expose an optional `--region` override.
- Added confirmation requirements for every state-changing operation, retryability only for retrieval operations, and guarded handling for documented order-in-progress responses that contain a master order ID.
