# Changelog

## Unreleased

### Added

- Add 104 commands from the published Cloud Firewall (Native Edition) API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 103 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Add 8 bounded waiters using documented terminal states from resource queries with explicit identity bindings.
- Require core `>=0.5.1 <1.0.0` for the declared response contracts and metadata features.

### Known limitations

- `native-firewall n100 order upgrade`: The published upgrade-order example has a null statusCode despite the explicit 800-success/900-failure schema; no successful fixture is fabricated.

See `openapi-catalogs/native-firewall/coverage.json` for the waiter assessment and upstream evidence limitations.
