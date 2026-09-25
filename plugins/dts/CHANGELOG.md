# Changelog

## Unreleased

### Added

- Add 72 commands from the published Data Transmission Service API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 72 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Add 6 bounded waiters using documented terminal states from resource queries with explicit identity bindings.
- Require core `>=0.5.1 <1.0.0` for response contracts, metadata support and independent request bindings.

See `openapi-catalogs/dts/coverage.json` for the waiter assessment and upstream evidence limitations.
