# Changelog

## Unreleased

### Changed

- Declared the ECS plugin API scope as all official ECS APIs whose URI starts
  with `/v4/ecs/`, keeping the generated surface boundary machine-readable in
  both the OpenAPI catalog evidence and plugin manifest.

## 0.1.0-beta.1 - 2026-07-05

Compared with `0.1.0-alpha.1`, this release replaces the small hand-prepared
alpha surface with generated metadata from the official ECS OpenAPI source.

### Added

- Tracked OpenAPI `source.json` and promoted `baseline.json` evidence for ECS.
- Command, API, table, fixture, and help metadata for all 220 official ECS APIs whose URI starts with `/v4/ecs`, expanded from the alpha `ecs instance list`, `ecs instance show`, and `ecs instance start` commands.

### Changed

- Changed the release channel from `alpha` to `beta` while keeping metadata quality at `generated` pending deeper ECS command review.
- Updated the required core range from `>=0.1.0-alpha.1 <1.0.0` to `>=0.2.0 <1.0.0`.
- Rebuilt command option descriptions so English metadata no longer carries Chinese-only upstream prose.
- Rebuilt table labels and localized help text for the expanded generated ECS surface, including cleaner Chinese fallback spacing for generated labels.
- Rebased table mappings on documented fields such as `instanceStatus` and `instanceName`.
- Kept the alpha instance-state waiters (`ecs.instance.running` and `ecs.instance.stopped`) and derived them from the generated `ecs instance show` response metadata.
- Kept manual live validation scope to safe retrieval commands while preserving generated command metadata for state-changing `/v4/ecs` APIs.

## 0.1.0-alpha.1 - 2026-06-21

### Added

- ECS instance list/show/start command metadata.
- Table definitions, waiters, localized help text, and offline fixtures.
- Live CTyun API mapping for supported ECS commands.
