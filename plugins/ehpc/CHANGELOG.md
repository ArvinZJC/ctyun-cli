# Changelog

## 0.1.0-beta.1 - 2026-07-17

### Added

- Generated 24 first-class-node Elastic High Performance Computing commands from the official public OpenAPI records, covering clusters, nodes, users, queues, regions, cluster types, images, base images, and scheduler information.
- Tracked normalized `source.json` evidence and a semantically identical promoted `baseline.json` snapshot for the complete `/v4/cthpc/` API scope.
- Added natural Simplified Chinese, American English, and British English command and parameter help with documented ranges, enum tokens, structured user inputs, and fixed pagination defaults.
- Added official response fixtures and matching output tables for all 24 commands.
- Commands that use `regionID` read the selected profile region by default and expose an optional `--region` override.
- Added confirmation requirements for every state-changing operation and retryability only for retrieval operations, with no unguarded accepted error statuses.
