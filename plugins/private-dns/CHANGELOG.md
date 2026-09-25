# Changelog

## Unreleased

### Added

- Add 40 commands from the published Private DNS API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 40 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Add 4 bounded waiters using documented terminal states from resource queries with explicit identity bindings.
- Require core `>=0.5.0 <1.0.0` for the declared response contracts and metadata features.

See `openapi-catalogs/private-dns/coverage.json` for the waiter assessment and upstream evidence limitations.
