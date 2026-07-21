# Changelog

## 0.1.0-beta.1 - 2026-07-18

### Added

- Added the initial 37-command Volume Backup Service surface for backups, repositories, policies, and tasks within the reviewed `/v4/ebs-backup/` scope.
- Added source-faithful Simplified Chinese, American English, and British English command help and table labels with canonical technical casing, documented defaults, finite values, conditional policy and billing inputs, and acronym-safe option names.
- Added one official response fixture and output table for every command, including wrapped task results and useful nested rows for backup, repository, policy, and task lists.
- Preserved the portal's retirement notice on all ten legacy operations and grouped them under visible `legacy` command paths without inventing replacement guidance.
- Commands read `regionID` from the selected profile by default and expose the shared optional `--region` override.
- Added confirmation requirements for all 23 state-changing operations and retryability only for the 14 retrieval operations.
- Set the required core range to `>=0.4.0 <1.0.0` so typed request-body options use non-string JSON serialization.
