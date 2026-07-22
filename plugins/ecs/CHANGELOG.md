# Changelog

## 0.1.0-beta.4 - 2026-07-21

### Changed

- Normalized generated technical casing and Simplified Chinese table and help labels against the tracked OpenAPI source.
- Generated deprecation guidance now names visible replacement options, and unsupported generated examples are omitted.
- Removed examples that only repeated the visible command path, including unresolved path-placeholder forms.
- Raised the required core range to `>=0.4.0 <1.0.0` because typed request-body options rely on non-string JSON serialization introduced in core 0.4.0.

## 0.1.0-beta.3 - 2026-07-17

### Added

- Added five remote attestation operations under `ecs remote-attestation`, covering policy creation, update, listing, deletion, and evidence submission.
- Added typed command option metadata for booleans, numbers, JSON arrays, JSON objects, and maps so structured OpenAPI inputs retain their documented request shape.

### Changed

- Expanded the declared ECS API scope to include `/global-trust-authority/` alongside `/v4/ecs/`.
- Rewrote generated English command descriptions across the ECS surface as concise operation-specific help text.
- Rebuilt command examples from captured official request and parameter evidence so required options are present and structured values use executable shell quoting.

## 0.1.0-beta.2 - 2026-07-11

### Changed

- Added deprecation metadata to ECS parameters that official OpenAPI docs describe as deprecated, obsolete, or planned for shutdown.
- Rebuilt generated Chinese parameter and argument help to use concise CLI labels instead of noisy upstream documentation prose.
- Declared the ECS `/v4/ecs/` API scope in both the OpenAPI catalog evidence and plugin manifest.
- Added optional `--region` overrides to commands that map `regionID` from the selected profile.
- Generated examples now fill path placeholders from captured official example responses when a matching scalar value is available.
- Raised the required core range to `>=0.3.1 <1.0.0` for deprecation-warning, generated-region, and API-scope metadata behaviour.

## 0.1.0-beta.1 - 2026-07-05

### Added

- Tracked OpenAPI `source.json` and promoted `baseline.json` evidence for ECS.
- Command, API, table, fixture, and help metadata for all 220 official ECS APIs whose URI starts with `/v4/ecs`, expanded from the alpha `ecs instance list`, `ecs instance show`, and `ecs instance start` commands.

### Changed

- Changed the release channel from `alpha` to `beta`.
- Updated the required core range from `>=0.1.0-alpha.1 <1.0.0` to `>=0.2.0 <1.0.0`.
- Rebuilt command option descriptions so English metadata no longer carries Chinese-only upstream prose.
- Rebuilt table labels and localized help text for the expanded generated ECS surface, including cleaner Chinese fallback spacing for generated labels.
- Rebased table mappings on documented fields such as `instanceStatus` and `instanceName`.
- Derived the instance-state waiters (`ecs.instance.running` and `ecs.instance.stopped`) from the generated `ecs instance show` response metadata.

## 0.1.0-alpha.1 - 2026-06-21

### Added

- ECS instance list/show/start command metadata.
- Table definitions, waiters, localized help text, and offline fixtures.
- Live CTyun API mapping for supported ECS commands.
