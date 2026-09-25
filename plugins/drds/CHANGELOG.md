# Changelog

## Unreleased

### Added

- Add 125 commands from the published Distributed Relational Database API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 120 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Add 8 bounded waiters using documented terminal states from resource queries with explicit identity bindings.
- Require core `>=0.5.1 <1.0.0` for response contracts, metadata support and independent request bindings.

### Known limitations

- `drds schema create`: The captured example uses statusCode=0, contradicting the explicit authoritative 800-success/900-failure schema. Preserve the raw evidence and support the documented contract without rewriting its status into a fabricated success fixture.
- `drds table create`: The captured example uses statusCode=0, contradicting the explicit authoritative 800-success/900-failure schema. Preserve the raw evidence and support the documented contract without rewriting its status into a fabricated success fixture.
- `drds node list`: The captured example uses statusCode=0, contradicting the explicit authoritative 800-success/900-failure schema. Preserve the raw evidence and support the documented contract without rewriting its status into a fabricated success fixture.
- `drds role permission list`: The captured example uses statusCode=0, contradicting the explicit authoritative 800-success/900-failure schema. Preserve the raw evidence and support the documented contract without rewriting its status into a fabricated success fixture.
- `drds instance create`: The malformed response example describes disk types instead of the documented order result. The explicit statusCode=200 response schema is supported; no order fixture is fabricated.

See `openapi-catalogs/drds/coverage.json` for the waiter assessment and upstream evidence limitations.
