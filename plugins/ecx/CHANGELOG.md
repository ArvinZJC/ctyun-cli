# Changelog

## Unreleased

### Added

- Add 252 commands from the published Intelligent Edge Cloud API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 250 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Add 10 bounded waiters using documented terminal states from resource queries with explicit identity bindings.
- Require core `>=0.5.0 <1.0.0` for the declared response contracts and metadata features.

### Known limitations

- `ecx order nat create`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `ecx v1 backend-group available-instance list`: The captured example omits the status object required by its response schema; no successful fixture is fabricated.

See `openapi-catalogs/ecx/coverage.json` for the waiter assessment and upstream evidence limitations.
