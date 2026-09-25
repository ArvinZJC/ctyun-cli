# Changelog

## Unreleased

### Added

- Add 269 commands from the published Container Registry Service API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 268 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Add 11 bounded waiters using documented terminal states from resource queries with explicit identity bindings.
- Require core `>=0.5.1 <1.0.0` for the declared response contracts and metadata features.

### Known limitations

- `crs import rule show`: The published example omits the statusCode/returnObj envelope required by its response schema. The command uses the explicit schema success contract; no successful fixture is fabricated.

See `openapi-catalogs/crs/coverage.json` for the waiter assessment and upstream evidence limitations.
