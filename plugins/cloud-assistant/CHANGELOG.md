# Changelog

## 0.1.0-beta.3 - 2026-07-21

### Changed

- Normalized generated technical casing and Simplified Chinese table and help labels against the tracked OpenAPI source.
- Removed examples that only repeated the visible command path, including unresolved path-placeholder forms.
- Raised the required core range to `>=0.4.0 <1.0.0` because typed request-body options rely on non-string JSON serialization introduced in core 0.4.0.

## 0.1.0-beta.2 - 2026-07-17

### Changed

- Rebuilt command examples from captured official request and parameter evidence so every required option has a concrete or explicitly reviewed value.
- Standardized English command descriptions and normalized example enum spelling to the declared command values.
- Added typed command option metadata for structured and scalar request values.

## 0.1.0-beta.1 - 2026-07-11

### Added

- Generated Cloud Assistant command, API, table, fixture, and help metadata for all 11 official ECS-section APIs whose URI starts with `/v4/cloud-assistant/`.
- Tracked OpenAPI `source.json` and promoted `baseline.json` evidence for the Cloud Assistant API scope.
- Concise generated Chinese parameter and argument help labels instead of noisy upstream documentation prose.
- Commands that use `regionID` now expose optional `--region` overrides while continuing to read the selected profile `region` by default.
- Metadata omits generated `900` accepted-status rules based on error-envelope fields so API failures surface as API status errors.
