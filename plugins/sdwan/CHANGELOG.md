# Changelog

## Unreleased

### Added

- Add 138 commands from the published SD-WAN API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 138 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Add 1 bounded waiters using documented terminal states from resource queries with explicit identity bindings.
- Require core `>=0.5.0 <1.0.0` for the declared response contracts and metadata features.

See `openapi-catalogs/sdwan/coverage.json` for the waiter assessment and upstream evidence limitations.

### Fixed

- Remove incorrect field and option deprecation warnings from documented offline device states.
