# Changelog

## 0.1.0-beta.3 - 2026-09-23

### Added

- Added asynchronous-task success, layer-build completion, execution completion, and trigger enabled/disabled waits, including documented failure outcomes.

### Changed

- Required core range `>=0.4.0 <1.0.0` → `>=0.5.0 <1.0.0` for command-bound waiter metadata.

## 0.1.0-beta.2 - 2026-07-21

### Changed

- Normalized generated technical casing and replaced mixed-language table and help labels with concise source-backed labels.
- Required core range `>=0.3.1 <1.0.0` → `>=0.4.0 <1.0.0` because typed request-body options rely on non-string JSON serialization introduced in core 0.4.0.

## 0.1.0-beta.1 - 2026-07-17

- Added the initial generated Cloud Function plugin with all 62 first-class-node APIs from the selected public portal revision.
- Added function, version, alias, trigger, reserved-instance, concurrency, layer, asynchronous invocation, custom-domain, workflow, execution, and task-event commands with profile-region overrides and confirmations for state-changing operations.
- Added localized help, typed request inputs, source-backed command examples, official response fixtures, and table metadata for all commands.
- Added normalized source and promoted baseline evidence for reproducible API inventory and provenance checks.
