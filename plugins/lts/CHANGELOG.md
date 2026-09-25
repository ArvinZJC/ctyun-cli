# Changelog

## Unreleased

### Added

- Add 171 commands from the published Log Service API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 168 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Add 15 bounded waiters using documented terminal states from resource queries with explicit identity bindings.
- Require core `>=0.5.1 <1.0.0` for the declared response contracts and metadata features.

### Known limitations

- `lts log search`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `lts host-group connected-host list`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `lts collection-rule host-group list`: No parseable successful JSON response example is published; no success fixture is fabricated.

See `openapi-catalogs/lts/coverage.json` for the waiter assessment and upstream evidence limitations.
