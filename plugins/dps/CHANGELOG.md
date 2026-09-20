# Changelog

## Unreleased

### Added

- Added batch tag binding/unbinding and recovery of unsubscribed subscription physical servers.
- Added physical super-node stock lookup and subscription physical-server `instance auto-renew show|update` commands from the current official documentation.

### Removed

- Removed the eight CPU, memory, disk, and network-interface monitoring commands whose APIs are no longer published in the official inventory.

## 0.1.0-beta.2 - 2026-07-21

### Changed

- Normalized generated technical casing, units, and Simplified Chinese table and help labels against the tracked OpenAPI source.
- Raised the required core range to `>=0.4.0 <1.0.0` because typed request-body options rely on non-string JSON serialization introduced in core 0.4.0.

## 0.1.0-beta.1 - 2026-07-17

- Added the initial generated Dedicated Physical Server plugin with all 62 first-class-node APIs from the selected public portal revision.
- Added physical-server, flavour, stock, image, RAID, volume, interface, security-group, monitoring, metadata, billing, operating-system, private-image, and disk-type commands with profile-region overrides and confirmations for state-changing operations.
- Added localized help, typed request inputs, source-backed command examples, official response fixtures, and table metadata for all commands.
- Added normalized source and promoted baseline evidence for reproducible API inventory and provenance checks.
