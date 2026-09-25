# Changelog

## Unreleased

### Added

- Add 169 commands from the published Cloud Monitoring Service API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 169 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Add 11 bounded waiters using documented terminal states from resource queries with explicit identity bindings.
- Require core `>=0.5.1 <1.0.0` for the declared response contracts and metadata features.

See `openapi-catalogs/cm/coverage.json` for the waiter assessment and upstream evidence limitations.

### Fixed

- Remove incorrect option and column deprecation warnings from data subscription offline states.
