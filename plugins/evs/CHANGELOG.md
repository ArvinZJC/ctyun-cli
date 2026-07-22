# Changelog

## 0.1.0-beta.1 - 2026-07-18

### Added

- Added the initial 30-command Elastic Volume Service surface for volumes, snapshots, automatic snapshot policies, snapshot-service activation, storage types, and volume-backed private images.
- Added source-faithful Simplified Chinese, American English, and British English command help and table labels with canonical technical casing, typed inputs, documented defaults and allowed values, and conditional billing and retention requirements.
- Added official response fixtures and output tables for every command, including preserved task-ID deprecation guidance and documented wrapped and order-in-progress results.
- Commands read `regionID` from the selected profile by default and expose the shared optional `--region` override.
- Added confirmation requirements for all 22 state-changing operations and retryability only for the eight retrieval operations.
- Set the required core range to `>=0.4.0 <1.0.0` so typed request-body options use non-string JSON serialization.
