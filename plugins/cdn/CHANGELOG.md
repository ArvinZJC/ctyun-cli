# Changelog

## Unreleased

### Added

- Add 92 commands from the published Content Delivery Network Acceleration API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 91 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Add 2 bounded waiters using documented terminal states from resource queries with explicit identity bindings.
- Require core `>=0.5.0 <1.0.0` for response contracts and metadata support.

### Known limitations

- `cdn edge-script staging show`: No parseable successful JSON response example is published; no success fixture is fabricated.

See `openapi-catalogs/cdn/coverage.json` for the waiter assessment and upstream evidence limitations.
