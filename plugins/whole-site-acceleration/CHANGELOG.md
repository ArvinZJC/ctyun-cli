# Changelog

## Unreleased

### Added

- Add 96 commands from the published Whole Site Acceleration API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 94 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Add 3 bounded waiters using documented terminal states from resource queries with explicit identity bindings.
- Require core `>=0.5.0 <1.0.0` for response contracts and metadata support.

### Known limitations

- `whole-site-acceleration report-subscription update`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `whole-site-acceleration block area list`: No parseable successful JSON response example is published; no success fixture is fabricated.

See `openapi-catalogs/whole-site-acceleration/coverage.json` for the waiter assessment and upstream evidence limitations.
