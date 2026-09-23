# Changelog

## 0.1.0-beta.3 - 2026-09-23

### Added

- Added scheduled scaling-policy listing with an optional scaling-group filter.
- Added activity and rule completion waits, a scaling-group modifiable wait, and enabled/disabled waits selected by group ID.

### Changed

- Required core range `>=0.4.0 <1.0.0` → `>=0.5.0 <1.0.0` for command-bound waiter metadata.

## 0.1.0-beta.2 - 2026-07-21

### Changed

- Normalized generated English casing and Simplified Chinese table and help labels against the tracked OpenAPI source.
- Required core range `>=0.3.1 <1.0.0` → `>=0.4.0 <1.0.0` because typed request-body options rely on non-string JSON serialization introduced in core 0.4.0.

## 0.1.0-beta.1 - 2026-07-17

- Added the initial generated Auto Scaling plugin with all 62 first-class-node APIs from the selected public portal revision.
- Added scaling configuration, group, instance, policy, activity, load-balancer, protection, service-status, and quota commands with profile-region overrides and confirmations for state-changing operations.
- Added localized help, typed request inputs, source-backed command examples, official response fixtures, and table metadata for all commands.
- Added normalized source and promoted baseline evidence for reproducible API inventory and provenance checks.
