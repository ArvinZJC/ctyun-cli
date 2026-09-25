# Changelog

## Unreleased

### Added

- Add 154 commands from the published Server Security (Native Edition) API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 154 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Add 9 bounded waiters using documented terminal states from resource queries with explicit identity bindings.
- Require core `>=0.5.1 <1.0.0` for the declared response contracts and metadata features.

See `openapi-catalogs/server-security/coverage.json` for the waiter assessment and upstream evidence limitations.
