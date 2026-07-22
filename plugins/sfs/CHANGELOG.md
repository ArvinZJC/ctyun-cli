# Changelog

## 0.1.0-beta.1 - 2026-07-18

### Added

- Added the initial 56-command SFS surface for file systems, capacities, mount points, VPC permission-group bindings, permission groups and rules, access modes, AD domains, cross-region replication, labels, subdirectories, pricing, storage types, regions, and quotas.
- Added source-faithful Simplified Chinese, American English, and British English command help and table labels with localized mount-path and AD-domain fields, canonical technical casing, documented finite values, defaults, cross-field constraints, resource-pool support boundaries, and acronym-safe option names.
- Added source-backed request examples, official response fixtures, and output tables for every command, including conditional subscription billing and encryption-key inputs while keeping resource-pool-dependent subnet inputs optional in usage.
- Commands read `regionID` from the selected profile by default and expose the shared optional `--region` override; source and destination regions remain explicit for replication creation.
- Added confirmation requirements for all 27 state-changing POST operations and retryability for the 25 GET operations, three read-only pricing inquiries, and AD-domain legality validation.
- Preserved five published response-field deprecations with their documented replacements while keeping public-beta, whitelist, support-boundary, and recommendation-only notices as help rather than lifecycle metadata.
- Set the required core range to `>=0.4.0 <1.0.0` so typed request-body options use non-string JSON serialization.
