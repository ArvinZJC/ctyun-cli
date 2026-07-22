# Changelog

## 0.1.0-beta.2 - 2026-07-21

### Changed

- Raised the required core range to `>=0.4.0 <1.0.0` because typed request-body options rely on non-string JSON serialization introduced in core 0.4.0.

## 0.1.0-beta.1 - 2026-07-17

### Added

- Generated eight first-class-node Application Cloud Server commands from the official public OpenAPI records, retaining four legacy Light Cloud Host APIs under `lite-instance` and `lite-flavor` alongside four successor commands under `instance` and `flavor`.
- Tracked normalized `source.json` evidence and a semantically identical promoted `baseline.json` snapshot for the complete `/v4/ecs/lite/` API scope, including the official legacy shutdown and successor mappings.
- Added natural Simplified Chinese, American English, and British English command and option help, typed scalar and structured inputs, and official offline response fixtures.
- Commands that use `regionID` read the selected profile region by default and expose an optional `--region` override.
- Added confirmation requirements for all create, rebuild, and package-upgrade operations while keeping package-list retrieval retryable and omitting unguarded accepted error statuses.
