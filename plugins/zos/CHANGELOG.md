# Changelog

## 0.1.0-beta.1 - 2026-07-18

### Added

- Added the initial 106-command ZOS surface across the documented `/v4/oss/` object-storage APIs and `/v4/zms/` assessment, migration, and semi-managed migration-agent APIs.
- Added source-faithful Simplified Chinese, American English, and British English command help and table labels with canonical object-storage technical tokens, typed request inputs, documented finite values, fixed defaults, patterns, cross-field rules, and resource-pool boundaries.
- Added source-backed command examples, official response fixtures, and output tables for every command, including captured ACL alternatives, bucket quotas, cross-region replication, and migration requests.
- Commands read request-scoping `regionID` from the selected profile by default and expose the shared optional `--region` override; source, destination, and target region inputs remain explicit resource fields.
- Added retryability for all 45 GET operations and the three read-only POST operations that generate temporary object links or quote resource-package prices; all 58 state-changing POST operations require confirmation and remain non-retryable.
- Preserved multipart, empty-bucket deletion, object-lock, retention, statistics, replication, role-policy, and ZMS state, quota, agent, and worker prerequisites as localized help without inventing unsupported lifecycle metadata.
- Set the required core range to `>=0.4.0 <1.0.0` so typed request-body options use non-string JSON serialization.
