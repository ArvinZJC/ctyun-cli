# Changelog

## Unreleased

### Fixed

- Correct NAT gateway name labels so they do not describe workflows.

### Changed

- Raise the required core range `>=0.5.0 <1.0.0` → `>=0.5.1 <1.0.0` so explicit header, query and body bindings remain independent.

## 0.1.0-beta.1 - 2026-09-23

### Added

- Add the NAT Gateway plugin with 22 commands covering the official product's linked first-class-node public NAT gateway, SNAT, DNAT, EIP association and pricing APIs.
- Include Chinese and English help, captured official offline responses, structured array inputs, region overrides, confirmation for state changes and gateway-state waiters. Collection polling requires an exact gateway ID.
