# Changelog

## Unreleased

### Added

- Add 47 commands from the published Distributed Hybrid Transactional and Analytical Processing Database API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 45 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Add 2 bounded waiters using documented terminal states from resource queries with explicit identity bindings.
- Require core `>=0.5.1 <1.0.0` for response contracts, metadata support and independent request bindings.

### Known limitations

- `htap parameter-template apply`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `htap parameter-template description update`: No parseable successful JSON response example is published; no success fixture is fabricated.

See `openapi-catalogs/htap/coverage.json` for the waiter assessment and upstream evidence limitations.
