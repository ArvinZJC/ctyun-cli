# Changelog

## Unreleased

### Added

- Added data-volume type changes and batch tag binding/unbinding.
- Added the volume-creation automatic-renewal option and ten multi-AZ-only volume-list filters and ordering inputs.
- Added volume automatic-renewal query and update commands and batch attach and detach commands from the current official documentation.
- Preserved the documented deprecated batch-operation task ID alongside its current replacement field.
- Added volume readiness, available/in-use targets, and snapshot availability waits selected by snapshot ID.

### Changed

- Refreshed snapshot-creation and automatic-renewal response metadata and added volume descriptions to listing tables.
- Updated snapshot-deletion applicability guidance to follow the official supported-region feature matrix.
- Required core range `>=0.4.0 <1.0.0` → `>=0.5.0 <1.0.0` for command-bound waiter metadata.

## 0.1.0-beta.1 - 2026-07-18

### Added

- Added the initial 30-command Elastic Volume Service surface for volumes, snapshots, automatic snapshot policies, snapshot-service activation, storage types, and volume-backed private images.
- Added source-faithful Simplified Chinese, American English, and British English command help and table labels with canonical technical casing, typed inputs, documented defaults and allowed values, and conditional billing and retention requirements.
- Added official response fixtures and output tables for every command, including preserved task-ID deprecation guidance and documented wrapped and order-in-progress results.
- Commands read `regionID` from the selected profile by default and expose the shared optional `--region` override.
- Added confirmation requirements for all 22 state-changing operations and retryability only for the eight retrieval operations.
- Set the required core range to `>=0.4.0 <1.0.0` so typed request-body options use non-string JSON serialization.
