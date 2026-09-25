# Changelog

## Unreleased

### Added

- Add 38 commands from the published Video on Demand API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 38 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Add 2 bounded waiters using documented terminal states from resource queries with explicit identity bindings.
- Require core `>=0.5.0 <1.0.0` for response contracts and metadata support.

See `openapi-catalogs/vod/coverage.json` for the waiter assessment and upstream evidence limitations.
