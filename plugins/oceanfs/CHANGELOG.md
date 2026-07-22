# Changelog

## 0.1.0-beta.1 - 2026-07-22

### Added

- Added the initial 34-command OceanFS surface for file systems, subdirectories, storage types, mounted clients, VPC permission-group bindings, permission groups and rules, pricing quotes, and cross-region replication.
- Added source-faithful Simplified Chinese, American English, and British English command help and table labels with localized mount-path fields, canonical technical casing, documented finite values, defaults, input constraints, and acronym-safe option names.
- Added official request examples, response fixtures, and output tables for every command, including conditional subscription billing and VPC endpoint subnet requirements.
- Commands read `regionID` from the selected profile by default and expose the shared optional `--region` override; source and destination region inputs remain explicit for replication creation.
- Added confirmation requirements for all 19 state-changing operations and retryability for the 12 GET operations and three read-only pricing quotes.
- Set the required core range to `>=0.4.0 <1.0.0` so typed request-body options use non-string JSON serialization.

### Changed

- Aligned the Simplified Chinese product name with the official title `海量文件服务 OceanFS` while retaining `OceanFS` as the English brand name.
