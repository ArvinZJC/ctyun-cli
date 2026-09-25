# Changelog

## Unreleased

### Added

- Add 43 commands from the published Application High Availability API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 41 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Require core `>=0.5.1 <1.0.0` for response contracts, metadata support and independent request bindings.

### Known limitations

- `application-ha authorization show`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `application-ha authorization confirm`: No parseable successful JSON response example is published; no success fixture is fabricated.

See `openapi-catalogs/application-ha/coverage.json` for the waiter assessment and upstream evidence limitations.
