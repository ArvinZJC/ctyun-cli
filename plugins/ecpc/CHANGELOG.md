# Changelog

## 0.1.0-beta.3 - 2026-09-23

### Added

- Added fifteen commands for custom-image preflight checks, tenant settings, and resource-pool listing.
- Added VPC and product-instance filters to image and enhanced-compute queries.
- Added nine Cloud Assistant v3 commands for execution history, host logs, script execution and retry, and script creation, listing, update, and deletion.
- Script execution exposes `--execution-timeout` separately from the global HTTP `--timeout` option.
- Added service, volume, peering, desktop, compute-desktop, VPC, and subnet lifecycle waits; collection waits select explicit resource IDs.

### Changed

- Aligned floating-IP renewal pricing with required billing-cycle inputs and removed its obsolete bandwidth input.
- Aligned peering request and response fields with current documentation and refreshed renewal-price response data.
- Corrected bandwidth-monitor timestamp units to seconds and clarified QoS burst constraints.
- Required core range `>=0.4.0 <1.0.0` → `>=0.5.0 <1.0.0` for command-bound waiter metadata.

## 0.1.0-beta.2 - 2026-07-21

### Changed

- Replaced repeated raw identifiers, mixed-language text, and portal prose with concise source-backed Simplified Chinese table and help labels.
- Required core range `>=0.3.1 <1.0.0` → `>=0.4.0 <1.0.0` because typed request-body options rely on non-string JSON serialization introduced in core 0.4.0.

## 0.1.0-beta.1 - 2026-07-17

- Added the initial generated CTyun Cloud Computer (Enterprise Edition) plugin with all 279 first-class-node APIs from the selected public portal revision.
- Added cloud-computer, service, resource-pack, desktop-pool, user, organisation, network, image, snapshot, backup, volume, security, policy, monitoring, billing, and enhanced-compute commands with profile-region overrides and confirmations for state-changing operations.
- Added localized help, typed request inputs, source-backed command examples, official response fixtures, and table metadata for all commands.
- Added normalized source and promoted baseline evidence for reproducible API inventory and provenance checks.
