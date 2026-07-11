# Changelog

## 0.1.0-beta.1 - 2026-07-11

### Added

- Generated Cloud Assistant command, API, table, fixture, and help metadata for
  all 11 official ECS-section APIs whose URI starts with
  `/v4/cloud-assistant/`.
- Tracked OpenAPI `source.json` and promoted `baseline.json` evidence for the
  Cloud Assistant API scope.
- Concise generated Chinese parameter and argument help labels instead of noisy
  upstream documentation prose.
- Commands that use `regionID` now expose optional `--region` overrides while
  continuing to read the selected profile `region` by default.
- Metadata omits generated `900` accepted-status rules based on error-envelope
  fields so API failures surface as API status errors.
