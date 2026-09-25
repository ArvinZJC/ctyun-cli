# Changelog

## Unreleased

### Added

- Add 61 commands from the published Cloud Dedicated Access API catalog, with Chinese and English help, explicit request bindings and response success checks.
- Include 59 captured response fixtures and require confirmation for state-changing operations; mutations are not retried.
- Require core `>=0.5.0 <1.0.0` for response contracts and metadata support.

### Known limitations

- `cda gateway list`: No parseable successful JSON response example is published; no success fixture is fabricated.
- `cda account-authorization list`: No parseable successful JSON response example is published; no success fixture is fabricated.

See `openapi-catalogs/cda/coverage.json` for the waiter assessment and upstream evidence limitations.
