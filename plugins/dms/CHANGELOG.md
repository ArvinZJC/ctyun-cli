# Changelog

## Unreleased

### Added

- Add 78 commands from the published Data Management Service API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 77 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Add 4 bounded waiters using documented terminal states from resource queries with explicit identity bindings.
- Require core `>=0.5.1 <1.0.0` for response contracts, metadata support and independent request bindings.

### Known limitations

- `dms organization instance list`: The published example uses status_code while the explicit schema declares statusCode; preserve the documented statusCode 200 contract without fabricating a matching fixture.

See `openapi-catalogs/dms/coverage.json` for the waiter assessment and upstream evidence limitations.
