# Changelog

## Unreleased

### Added

- Cloud Assistant command, API, table, fixture, and help metadata for all 11
  official ECS-section APIs whose URI starts with `/v4/cloud-assistant/`.
- Tracked OpenAPI `source.json` and promoted `baseline.json` evidence for the
  Cloud Assistant API scope.

### Changed

- Rebuilt generated Chinese parameter and argument help to use concise CLI
  labels instead of noisy upstream documentation prose.
- Commands that use `regionID` now expose optional `--region` overrides while
  continuing to read the selected profile `region` by default.
- Removed generated `900` accepted-status metadata based on error-envelope
  fields so API failures surface as API status errors.
