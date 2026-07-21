# Changelog

## 0.1.0-beta.1 - 2026-07-18

### Added

- Added the initial 40-command HPFS surface for file systems, directories, FILESETs, dataflow policies and tasks, protocol services, labels, clusters, performance baselines, regions, availability zones, and quotas.
- Added source-faithful Simplified Chinese, American English, and British English command help and table labels with canonical HPFS, FILESET, operating-system, and identifier casing plus documented finite values, defaults, constraints, and acronym-safe option names.
- Added official request examples, response fixtures, and output tables for every command, including conditional dataflow, VPC endpoint subnet, and subscription billing inputs.
- Commands read `regionID` from the selected profile by default and expose the shared optional `--region` override.
- Added confirmation requirements for all 15 state-changing operations and retryability for the 24 GET operations plus the read-only label query.
- Preserved the portal's `mountCount` response-field deprecation without inventing unsupported API or command lifecycle guidance.
- Set the required core range to `>=0.4.0 <1.0.0` so typed request-body options use non-string JSON serialization.
