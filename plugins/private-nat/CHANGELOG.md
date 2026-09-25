# Changelog

## Unreleased

### Fixed

- Correct gateway and transit-network name labels so they do not describe workflows.

### Changed

- Raise the required core range `>=0.5.0 <1.0.0` → `>=0.5.1 <1.0.0` so explicit header, query and body bindings remain independent.

## 0.1.0-beta.1 - 2026-09-23

### Added

- Add the Private NAT Gateway plugin with 21 commands covering private gateways, SNAT, DNAT, transit IPs, transit subnets and pricing APIs.
- Include Chinese and English help, captured official offline responses, structured array inputs, region overrides, confirmation for state changes and gateway, SNAT and transit-IP waiters. Polling requires an exact resource identity.
