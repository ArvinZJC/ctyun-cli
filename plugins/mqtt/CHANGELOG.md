# Changelog

## Unreleased

### Added

- Add 61 commands from the published Distributed Message Queuing Telemetry Transport API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 59 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Add 4 bounded waiters using documented terminal states from resource detail queries.
- Require core `>=0.5.1 <1.0.0` for response contracts, metadata support and independent request bindings.

### Known limitations

- `mqtt user permission revoke`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `mqtt v1 client status list`: The example omits the documented statusCode success marker. Retain the documented statusCode=800 contract without fabricating a complete response fixture.

See `openapi-catalogs/mqtt/coverage.json` for the waiter assessment and upstream evidence limitations.

### Fixed

- Remove incorrect API deprecation warnings from client disconnection commands.
