# Changelog

## Unreleased

### Added

- Add 90 commands from the published Live Streaming API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 88 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Add 8 bounded waiters using documented terminal states from resource queries with explicit identity bindings.
- Require core `>=0.5.0 <1.0.0` for response contracts and metadata support.

### Known limitations

- `live-streaming pull-config update`: The published response example is a parameter-validation failure (200002); the documented success code is 100000. No successful response fixture is fabricated.
- `live-streaming media-template binding update`: The response example is the placeholder 111 rather than a response object; success uses the documented statusCode 100000 without a fabricated fixture.

See `openapi-catalogs/live-streaming/coverage.json` for the waiter assessment and upstream evidence limitations.
