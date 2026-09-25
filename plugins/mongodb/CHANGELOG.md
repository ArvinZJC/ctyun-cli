# Changelog

## Unreleased

### Added

- Add 180 commands for Document Database Service, covering instances, accounts, backups, recovery, networking, parameters, logs, monitoring, tags and orders; preserve older published operations under `mongodb v1`.
- Include Chinese and English help, captured response fixtures, explicit API scopes and two bounded instance-running waiters, including exact-ID list selection and operation-specific documented failure states.
- Require confirmation for mutations, including GET-based billing conversion and instance destruction, and disable retries for state-changing operations.
- Complete parameter-reset and legacy host-alarm fixtures from supplemental official API reference examples, preserving pagination and the host-alarm response array.
- Require core `>=0.5.1 <1.0.0` for response contracts and waiters.

### Fixed

- Correct response labels against operation-specific meanings, including parameter modification flags, backup task results, historical-data synchronization and log line numbers.

### Known limitations

- Three export APIs remain blocked because published documentation does not establish an unambiguous binary response contract; see `openapi-catalogs/mongodb/coverage.json`.
- Task collections lack a unique task input for safe polling. Instance overview queries accept multiple IDs and do not select one arbitrary row for waiting.
