# Changelog

## Unreleased

### Added

- Add 157 commands from the published Microservice Cloud Application Platform API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 157 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Add 2 bounded waiters using documented terminal states from resource queries with explicit identity bindings.
- Require core `>=0.5.1 <1.0.0` for the declared response contracts and metadata features.

See `openapi-catalogs/msap/coverage.json` for the waiter assessment and upstream evidence limitations.
